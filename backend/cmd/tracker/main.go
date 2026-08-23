package main

import (
	"log"
	"os"

	"github.com/lumen/relay/internal/app"
)

func main() {
	_ = os.Setenv("APP_ROLE", "tracker")
	rt, err := app.Boot()
	if err != nil {
		log.Fatal(err)
	}
	if err := rt.RunTracker(); err != nil {
		log.Fatal(err)
	}
}
