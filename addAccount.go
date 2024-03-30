package main

import (
	"encoding/json"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"io"
	"minecraft-accounts-service/github.com/SocialMinecraft/protos"
	"net/http"
	"time"
)

func addAccount(nc *nats.Conn, db *Db, msg *nats.Msg) error {
	e := &protos.AddMinecraftAccountRequest{}
	if err := proto.Unmarshal(msg.Data, e); err != nil {
		return err
	}

	// Check that the minecraft name is not in use already
	existingAccount, err := db.GetAccountByUsername(e.MinecraftUsername)
	if err != nil {
		return err
	}
	if existingAccount != nil {
		errMsg := "Minecraft Account is already registered."
		re := &protos.ChangeMinecraftAccountResponse{}
		re.Success = false
		re.ErrorMessage = &errMsg
		buf, err := proto.Marshal(re)
		if err != nil {
			return err
		}
		return msg.Respond(buf)
	}

	// Is their first account (and thus main account)
	accounts, err := db.GetAccountsByUser(e.UserId)
	if err != nil {
		return err
	}
	isMain := len(accounts) <= 0

	// Get UUID
	res, err := http.Get("https://api.mojang.com/users/profiles/minecraft/" + e.MinecraftUsername)
	if err != nil {
		return err
	}
	if res.StatusCode == 404 {
		errMsg := "Minecraft Account was not found"
		re := &protos.ChangeMinecraftAccountResponse{}
		re.Success = false
		re.ErrorMessage = &errMsg
		buf, err := proto.Marshal(re)
		if err != nil {
			return err
		}
		return msg.Respond(buf)
	}
	if res.StatusCode == 429 {
		errMsg := "Minecraft Account Lookup is overload, please try again in a minute"
		re := &protos.ChangeMinecraftAccountResponse{}
		re.Success = false
		re.ErrorMessage = &errMsg
		buf, err := proto.Marshal(re)
		if err != nil {
			return err
		}
		return msg.Respond(buf)
	}
	if res.StatusCode != 200 {
		errMsg := "Unknown error when looking up username"
		re := &protos.ChangeMinecraftAccountResponse{}
		re.Success = false
		re.ErrorMessage = &errMsg
		buf, err := proto.Marshal(re)
		if err != nil {
			return err
		}
		return msg.Respond(buf)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var apiResp struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	}
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return err
	}
	uuid := apiResp.Id

	// Try to whitelist the account - should this really just be an effect of the event instead of a direct dep?
	{
		buf, err := proto.Marshal(&protos.WhitelistMinecraftAccount{Uuid: uuid})
		if err != nil {
			return err
		}
		_, err = nc.Request("minecraft.whitelist.add", buf, time.Second*1)
		if err != nil {
			return err
		}
		// reply can actually not fail... as long as we get a response.
	}

	// Save account to database
	var account Account
	account.IsMain = isMain
	account.UserId = e.UserId
	account.MinecraftUsername = e.MinecraftUsername
	account.MinecraftUuid = uuid
	account.FirstName = e.FirstName
	err = db.AddAccount(account)
	if err != nil {
		return err
	}

	// Send respones
	re := &protos.ChangeMinecraftAccountResponse{}
	re.Success = true
	buf, err := proto.Marshal(re)
	if err != nil {
		return err
	}
	msg.Respond(buf)

	// Announce the change
	buf, err = proto.Marshal(&protos.MinecraftAccountChanged{
		UserId: e.UserId,
		Change: protos.MinecraftAccountChangeType_ADDED,
		Account: &protos.MinecraftAccount{
			MinecraftUuid:     uuid,
			MinecraftUsername: e.MinecraftUsername,
			IsMain:            isMain,
			FirstName:         e.FirstName,
		},
	})
	if err != nil {
		return err
	}

	//log.Println(base64.StdEncoding.EncodeToString(buf))

	return nc.Publish("accounts.minecraft.changed", buf)

}
