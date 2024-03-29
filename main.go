package main

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"log"
	"os"
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
		if err := addAccount(nc, db, msg); err != nil {
			log.Fatalln(err)
		}
	})
	defer addSub.Unsubscribe()
	if err != nil {
		log.Fatalln(err)
		return
	}

	removeSub, err := nc.Subscribe("accounts.minecraft.remove", func(msg *nats.Msg) {
		if err := removeAccount(nc, db, msg); err != nil {
			log.Fatalln(err)
		}
	})
	defer removeSub.Unsubscribe()
	if err != nil {
		log.Fatalln(err)
		return
	}

}
