package main

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"minecraft-accounts-service/github.com/SocialMinecraft/protos"
	"net/http"
	"os"
	"time"
)

func main() {

	config, err := getConfig()
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	db, err := Connect(config.PostgresUrl)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	nc, err := nats.Connect(config.NatsUrl)
	defer nc.Drain()

	addSub, err := nc.Subscribe("accounts.minecraft.add", func(msg *nats.Msg) {
		e := &protos.AddMinecraftAccountRequest{}
		if err := proto.Unmarshal(msg.Data, e); err != nil {
			log.Fatalln(err)
			return
		}

		// Check that the minecraft name is not in use already
		existingAccount, err := db.GetAccountByUsername(e.MinecraftUsername)
		if err != nil {
			log.Fatalln(err)
			return
		}
		if existingAccount != nil {
			errMsg := "Minecraft Account is already registered."
			re := &protos.ChangeMinecraftAccountResponse{}
			re.Success = false
			re.ErrorMessage = &errMsg
			buf, err := proto.Marshal(re)
			if err != nil {
				log.Fatalln(err)
				return
			}
			msg.Respond(buf)
			return
		}

		// Is their first account (and thus main account)
		accounts, err := db.GetAccountsByUser(e.UserId)
		if err != nil {
			log.Fatalln(err)
			return
		}
		isMain := len(accounts) <= 0

		// Get UUID
		res, err := http.Get("https://api.mojang.com/users/profiles/minecraft/" + e.MinecraftUsername)
		if err != nil {
			log.Fatalln(err)
			return
		}
		if res.StatusCode == 404 {
			errMsg := "Minecraft Account was not found"
			re := &protos.ChangeMinecraftAccountResponse{}
			re.Success = false
			re.ErrorMessage = &errMsg
			buf, err := proto.Marshal(re)
			if err != nil {
				log.Fatalln(err)
				return
			}
			msg.Respond(buf)
			return
		}
		if res.StatusCode == 429 {
			errMsg := "User System is overload, please try again in a minute"
			re := &protos.ChangeMinecraftAccountResponse{}
			re.Success = false
			re.ErrorMessage = &errMsg
			buf, err := proto.Marshal(re)
			if err != nil {
				log.Fatalln(err)
				return
			}
			msg.Respond(buf)
			return
		}
		if res.StatusCode != 200 {
			errMsg := "Unknown error when looking up username"
			re := &protos.ChangeMinecraftAccountResponse{}
			re.Success = false
			re.ErrorMessage = &errMsg
			buf, err := proto.Marshal(re)
			if err != nil {
				log.Fatalln(err)
				return
			}
			msg.Respond(buf)
			return
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalln(err)
			return
		}
		var apiResp struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		}
		err = json.Unmarshal(body, &apiResp)
		if err != nil {
			log.Fatalln(err)
			return
		}
		uuid := apiResp.Id

		// Try to whitelist the account - should this really just be an effect of the event instead of a direct dep?
		{
			buf, err := proto.Marshal(&protos.WhitelistMinecraftAccount{Uuid: uuid})
			if err != nil {
				log.Fatalln(err)
				return
			}
			_, err = nc.Request("minecraft.whitelist.add", buf, time.Second*3)
			if err != nil {
				log.Fatalln(err)
				return
			}
			// reply can actually not fail... as long as we get a response.
		}

		// Save account to database
		var account Account
		account.IsMain = isMain
		account.UserId = e.UserId
		account.MinecraftUsername = e.MinecraftUsername
		account.MinecraftUuid = uuid
		err = db.AddAccount(account)
		if err != nil {
			log.Fatalln(err)
			return
		}

		// Send respones
		errMsg := "Unknown error when looking up username"
		re := &protos.ChangeMinecraftAccountResponse{}
		re.Success = false
		re.ErrorMessage = &errMsg
		buf, err := proto.Marshal(re)
		if err != nil {
			log.Fatalln(err)
			return
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
			},
		})
		if err != nil {
			log.Fatalln(err)
			return
		}
		nc.Publish("accounts.minecraft.changed", buf)

	})
	defer addSub.Unsubscribe()
	if err != nil {
		log.Fatalln(err)
		return
	}

}
