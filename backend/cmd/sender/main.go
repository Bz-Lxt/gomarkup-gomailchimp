package main

import (
	"log"
	"os"

	"github.com/lumen/relay/internal/app"
)

func main() {
	_ = os.Setenv("APP_ROLE", "sender")
	rt, err := app.Boot()
	if err != nil {
		log.Fatal(err)
	}
	if err := rt.RunSender(); err != nil {
		log.Fatal(err)
	}
}
