package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"

	plex "github.com/jrudio/go-plex-client"
)

// Clears the screen
func clear() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}

func main() {

	var movie_library_keys []string
	var show_library_keys []string
	var final_movie_library []plex.Metadata
	var final_show_library []plex.Metadata

	clear()

	//Read config json contents
	raw_conf, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalf("Could not read config file: %v", err)
	}

	//Unmarshall the raw json into a string map for use in the code
	var conf map[string]string
	err = json.Unmarshal(raw_conf, &conf)
	if err != nil {
		log.Fatalf("Could not unmarshall conf json: %v", err)
	}

	//Connect to the Plex server
	plexClient, err := plex.New(conf["plex_server_url"], conf["plex_token"])
	if err != nil {
		log.Fatalf("Plex client connection test failed %v", err)
	}

	//Test the connection to the Plex server
	success, err := plexClient.Test()
	if err != nil || !success {
		log.Fatalf("Connection test to the plex server was not successful: %v", err)
	}

	//I had to manually alter the GetLibraries() function to allow not validating ssl cert
	//Pull request into the upstream library is pending
	sections, err := plexClient.GetLibraries(false)
	if err != nil {
		log.Fatalf("Could not get libraries from Plex server: %v", err)
	}

	//Get the id's of all movie or show libraries
	//Put the id's into a corresponding slice
	for _, library := range sections.MediaContainer.Directory {

		if library.Type == "movie" {
			//An extra check to see if the library is using the movie scanners
			//This is to distinguish the movies from the videos libraries
			if library.Scanner == "Plex Movie Scanner" || library.Scanner == "Plex Movie" {
				movie_library_keys = append(movie_library_keys, library.Key)
			}
		} else if library.Type == "show" {
			show_library_keys = append(show_library_keys, library.Key)
		}

	}

	for _, movie_library_key := range movie_library_keys {
		movies, err := plexClient.GetLibraryContent(movie_library_key, "")
		if err != nil {
			log.Fatalf("Could not get movie library contents: %v", err)
		}
		for _, movie := range movies.MediaContainer.MediaContainer.Metadata {
			final_movie_library = append(final_movie_library, movie)
		}
	}

	//Testing with the shows
	for _, show_library_key := range show_library_keys {
		shows, err := plexClient.GetLibraryContent(show_library_key, "")
		if err != nil {
			log.Fatalf("Could not get movie library contents: %v", err)
		}
		for _, show := range shows.MediaContainer.MediaContainer.Metadata {
			final_show_library = append(final_show_library, show)
		}

	}

	json, err := json.Marshal(final_show_library)
	if err != nil {
		log.Fatalf("Could not marshal json: %v", err)
	}
	os.WriteFile("./test.json", json, 0644)
	/*
		json, err := json.Marshal(final_movie_library)
		if err != nil {
			log.Fatalf("Could not marshall movie data dump: %v", err)
		}
		fmt.Println(json)
	*/
}
