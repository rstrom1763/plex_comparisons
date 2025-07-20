package main

import (
	"log"
	"os"
)

func main() {

	//conf := initConf()

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
		err := dump()
		if err != nil {
			log.Fatalf("there was an error doing the csv dump: " + err.Error())
		}
	}
}
