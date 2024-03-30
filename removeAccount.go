package main

import (
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"minecraft-accounts-service/github.com/SocialMinecraft/protos"
	"time"
)

func removeAccount(nc *nats.Conn, db *Db, msg *nats.Msg) error {
	var buf []byte

	e := &protos.RemoveMinecraftAccountRequest{}
	if err := proto.Unmarshal(msg.Data, e); err != nil {
		return err
	}

	// Check that the account exists and is owned by the user
	existingAccount, err := db.GetAccountByUsername(e.MinecraftUsername)
	if err != nil {
		return err
	}
	if existingAccount == nil {
		errMsg := "Minecraft Account is not registered."
		re := &protos.ChangeMinecraftAccountResponse{}
		re.Success = false
		re.ErrorMessage = &errMsg
		if buf, err = proto.Marshal(re); err != nil {
			return err
		}
		return msg.Respond(buf)
	}
	if existingAccount.UserId != e.UserId {
		errMsg := "Minecraft Account is registered to a different user."
		re := &protos.ChangeMinecraftAccountResponse{}
		re.Success = false
		re.ErrorMessage = &errMsg
		if buf, err = proto.Marshal(re); err != nil {
			return err
		}
		return msg.Respond(buf)
	}

	// Remove the account from the whitelist... should this really jsut be a side effect.
	if buf, err = proto.Marshal(&protos.UnwhitelistMinecraftAccount{Uuid: existingAccount.MinecraftUuid}); err != nil {
		return err
	}
	if _, err = nc.Request("minecraft.whitelist.remove", buf, time.Second*3); err != nil {
		return err
	}
	// reply can actually not fail... as long as we get a response.

	// Remove the account from the database.
	if err = db.DeleteAccount(existingAccount.Id); err != nil {
		return err
	}

	// Send response
	re := &protos.ChangeMinecraftAccountResponse{}
	re.Success = true
	if buf, err = proto.Marshal(re); err != nil {
		return err
	}
	msg.Respond(buf)

	// Announce the change
	buf, err = proto.Marshal(&protos.MinecraftAccountChanged{
		UserId: e.UserId,
		Change: protos.MinecraftAccountChangeType_REMOVED,
		Account: &protos.MinecraftAccount{
			MinecraftUuid:     existingAccount.MinecraftUuid,
			MinecraftUsername: existingAccount.MinecraftUsername,
			IsMain:            existingAccount.IsMain,
			FirstName:         existingAccount.FirstName,
		},
	})
	if err != nil {
		return err
	}
	nc.Publish("accounts.minecraft.changed", buf)

	// Do we need to also change the users main account?
	if existingAccount.IsMain {
		if err = UpdateUsersMain(nc, db, existingAccount.UserId); err != nil {
			return err
		}
	}

	return nil
}

func UpdateUsersMain(nc *nats.Conn, db *Db, userId string) error {
	accounts, err := db.GetAccountsByUser(userId)
	if err != nil {
		return err
	}

	if len(accounts) <= 0 {
		// nothing to do.
		return nil
	}

	// double check they don't have a main account.
	for _, a := range accounts {
		if a.IsMain {
			return nil
		}
	}

	account := accounts[0]
	account.IsMain = true

	if err = db.UpdateAccount(account); err != nil {
		return err
	}

	// Announce the change
	buf, err := proto.Marshal(&protos.MinecraftAccountChanged{
		UserId: userId,
		Change: protos.MinecraftAccountChangeType_UPDATED,
		Account: &protos.MinecraftAccount{
			MinecraftUuid:     account.MinecraftUuid,
			MinecraftUsername: account.MinecraftUsername,
			IsMain:            account.IsMain,
			FirstName:         account.FirstName,
		},
	})
	if err != nil {
		return err
	}
	return nc.Publish("accounts.minecraft.changed", buf)
}
