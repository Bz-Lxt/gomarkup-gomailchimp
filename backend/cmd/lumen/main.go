package main

import (
	"log"
	"os"

	"github.com/lumen/relay/internal/app"
	"github.com/lumen/relay/internal/config"
)

func main() {
	rt, err := app.Boot()
	if err != nil {
		log.Fatal(err)
	}
	role := os.Getenv("APP_ROLE")
	if role == "" {
		role = config.Load().Role
	}
	switch role {
	case "sender":
		if err := rt.RunSender(); err != nil {
			log.Fatal(err)
		}
	case "tracker":
		if err := rt.RunTracker(); err != nil {
			log.Fatal(err)
		}
	default:
		if err := rt.RunAPI(); err != nil {
			log.Fatal(err)
		}
	}
}
