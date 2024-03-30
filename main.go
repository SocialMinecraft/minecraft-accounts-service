package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"log"
	"os"
	"runtime"
)

var (
	nc        *nats.Conn
	addSub    *nats.Subscription
	removeSub *nats.Subscription
	listSub   *nats.Subscription
	db        *Db
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatalln(err)
		return
	}

	config, err := getConfig()
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	db, err = Connect(config.PostgresUrl)
	if err != nil {
		log.Fatalln(err)
		return
	}
	//defer db.Close()

	nc, err = nats.Connect(config.NatsUrl)
	if err != nil {
		log.Fatalln(err)
		return
	}
	//defer nc.Drain()

	addSub, err = nc.Subscribe("accounts.minecraft.add", func(msg *nats.Msg) {
		log.Println("Got Account Add Request")
		if err := addAccount(nc, db, msg); err != nil {
			log.Println(err)
		}
	})
	//defer addSub.Unsubscribe()
	if err != nil {
		log.Fatalln(err)
		return
	}

	removeSub, err = nc.Subscribe("accounts.minecraft.remove", func(msg *nats.Msg) {
		log.Println("Got Account Remove Request")
		if err := removeAccount(nc, db, msg); err != nil {
			log.Println(err)
		}
	})
	//defer removeSub.Unsubscribe()
	if err != nil {
		log.Fatalln(err)
		return
	}
	listSub, err = nc.Subscribe("accounts.minecraft.list", func(msg *nats.Msg) {
		log.Println("Got Account List Request")
		if err := listAccounts(nc, db, msg); err != nil {
			log.Println(err)
		}
	})
	//defer listSub.Unsubscribe()
	if err != nil {
		log.Fatalln(err)
		return
	}

	log.Println("Running")
	// Keep the program running
	runtime.Goexit()
}
