package main

import (
	"bore/internal/cmdline"
	"bore/internal/server"
	"bore/internal/util"
	"flag"
	"log"
)

func main() {
	config := cmdline.InitialServerConfig()
	if config.Help {
		flag.PrintDefaults()
		return
	}

	if config.Debug {
		util.SetDebug(true)
	}

	log.Printf("starting server: TCP/%s, UDP/%s", config.TCPAddress, config.UDPAddress)
	s, err := server.NewP2PServer(config.TCPAddress.String(),
		config.UDPAddress.String())

	if err != nil {
		log.Fatal("Error: ", err)
		return
	}

	err = s.Start()
	if err != nil {
		log.Fatal("start failure: ", err)
	}

	go s.Loop()

	cmdline.HoldUntilInt()

	log.Printf("closing the server")
	if err := s.Close(); err != nil {
		util.Debug.Log(err)
	}
}
