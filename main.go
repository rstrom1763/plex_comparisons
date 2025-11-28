package main

import (
	"log"
	"os"
	"slices"
)

func main() {

	args := os.Args

	if len(args) == 1 {
		log.Fatal("Not enough arguments!")
	}

	switch arg := args[1]; arg {
	case "dump":
		plexDbPath, err := env("PLEX_DB_PATH")
		if err != nil {
			log.Fatalf("env variable PLEX_DB_PATH is empty")
		}
		err = dump(plexDbPath)
		if err != nil {
			log.Fatalf("there was an error performing the csv dump: " + err.Error())
		}
	case "compare":
		if len(args) != 5 {
			log.Fatalf("Expected 4 arguments, got %d", len(args)-1)
		}

		acceptedMediaTypes := []string{"movie", "show", "song"}
		providedMediaType := args[4]

		if !slices.Contains(acceptedMediaTypes, providedMediaType) {
			log.Fatalf("Not an accepted media type: %s", providedMediaType)
		}

		dump1Path := args[2]
		dump2Path := args[3]

		compare(dump1Path, dump2Path, providedMediaType)
	default:
		log.Fatalf("Unknown command: %s", arg)
	}
}
