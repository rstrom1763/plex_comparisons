package main

import (
	"fmt"
	"log"
	"os"
	"slices"

	utils "github.com/rstrom1763/goUtils"
)

func main() {
	exitOnError(runCLI(os.Args))
}

func exitOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func runCLI(args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("not enough arguments")
	}

	switch arg := args[1]; arg {
	case "dump":
		if len(args) != 2 {
			return fmt.Errorf("expected 1 argument, got %d", len(args)-1)
		}

		plexDbPath, err := env("PLEX_DB_PATH")
		if err != nil {
			return fmt.Errorf("env variable PLEX_DB_PATH is empty")
		}
		err = dump(plexDbPath)
		if err != nil {
			return fmt.Errorf("there was an error performing the csv dump: %w", err)
		}
	case "compare":
		if len(args) != 5 {
			return fmt.Errorf("expected 4 arguments, got %d", len(args)-1)
		}

		acceptedMediaTypes := []string{"movie", "show", "song"}
		providedMediaType := args[4]

		if !slices.Contains(acceptedMediaTypes, providedMediaType) {
			return fmt.Errorf("not an accepted media type: %s", providedMediaType)
		}

		dump1Path := args[2]
		dump2Path := args[3]

		if !utils.FileExists(dump1Path) {
			return fmt.Errorf("%s does not exist", dump1Path)
		}
		if !utils.FileExists(dump2Path) {
			return fmt.Errorf("%s does not exist", dump2Path)
		}

		if err := compare(dump1Path, dump2Path, providedMediaType); err != nil {
			return err
		}
	case "byteSum":
		if len(args) != 4 {
			return fmt.Errorf("expected 3 arguments, got %d", len(args)-1)
		}

		dumpPath := args[2]
		mediaType := args[3]

		byteCount, err := getByteSumFromDumpFile(dumpPath, mediaType)
		if err != nil {
			return fmt.Errorf("there was an error calculating the byte sum: %w", err)
		}

		byteCountString := utils.HumanReadableByteCountString(byteCount)

		fmt.Printf("Bytes: %s\n", byteCountString)

	case "server":
		err := RunServer()
		if err != nil {
			return fmt.Errorf("could not start server: %w", err)
		}

	default:
		return fmt.Errorf("unknown command: %s", arg)
	}

	return nil
}
