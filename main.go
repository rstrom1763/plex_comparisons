package main

import (
	"log"
	"os"
)

func main() {

	args := os.Args

	if len(args) == 1 {
		log.Fatal("Not enough arguments!")
	}

	switch arg := args[1]; arg {
	case "server":
		//runServer(conf)
	case "client":
		//runClient(conf)
	case "dump":
		plexDbPath, err := env("PLEX_DB_PATH")
		if err != nil {
			log.Fatalf("env variable PLEX_DB_PATH is empty")
		}
		err = dump(plexDbPath)
		if err != nil {
			log.Fatalf("there was an error performing the csv dump: " + err.Error())
		}
	}
}
