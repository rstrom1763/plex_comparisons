package main

import (
	"fmt"
	"log"
	"os"
	"slices"

	utils "github.com/rstrom1763/goUtils"
)

func main() {

	args := os.Args

	if len(args) == 1 {
		log.Fatal("Not enough arguments!")
	}

	switch arg := args[1]; arg {
	case "dump":
		if len(args) != 2 {
			log.Fatalf("Expected 1 argument, got %d", len(args)-1)
		}

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

		if !utils.FileExists(dump1Path) {
			log.Fatalf("%s does not exist", dump1Path)
		}
		if !utils.FileExists(dump2Path) {
			log.Fatalf("%s does not exist", dump2Path)
		}

		compare(dump1Path, dump2Path, providedMediaType)
	case "byteSum":
		if len(args) != 4 {
			log.Fatalf("Expected 3 arguments, got %d", len(args)-1)
		}

		dumpPath := args[2]
		mediaType := args[3]

		byteCount, err := getByteSumFromDumpFile(dumpPath, mediaType)
		if err != nil {
			log.Fatalf("There was an error calculating the byte sum: %s", err)
		}

		byteCountString := utils.HumanReadableByteCountString(byteCount)

		fmt.Printf("Bytes: %s\n", byteCountString)

	case "server":
		RunServer()

	default:
		log.Fatalf("Unknown command: %s", arg)
	}
}
