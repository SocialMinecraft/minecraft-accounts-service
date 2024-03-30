package main

import (
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"minecraft-accounts-service/github.com/SocialMinecraft/protos"
)

func listAccounts(nc *nats.Conn, db *Db, msg *nats.Msg) error {
	var buf []byte

	e := &protos.ListMinecraftAccountsRequest{}
	if err := proto.Unmarshal(msg.Data, e); err != nil {
		return err
	}

	accounts, err := db.GetAccountsByUser(e.UserId)
	if err != nil {
		return err
	}

	// Send response
	re := &protos.ListMinecraftAccountsResponse{}
	re.UserId = e.UserId
	for _, account := range accounts {
		re.Accounts = append(re.Accounts, &protos.MinecraftAccount{
			MinecraftUuid:     account.MinecraftUuid,
			MinecraftUsername: account.MinecraftUsername,
			IsMain:            account.IsMain,
			FirstName:         account.FirstName,
		})
	}
	if buf, err = proto.Marshal(re); err != nil {
		return err
	}
	return msg.Respond(buf)
}
