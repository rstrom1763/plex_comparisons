package main

import (
	"encoding/json"
	"log"
	"os"

	plex "github.com/jrudio/go-plex-client"
)

func runService(conf map[string]string) {

	var movie_library_keys []string
	var show_library_keys []string
	var final_movie_library []plex.Metadata
	var final_show_library []plex.Metadata

	clear()

	dataDir := "./data/"

	//Ensure that the dataDir actually exists
	err := ensureFolderExists(dataDir)
	if err != nil {
		log.Fatalf(err.Error())
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
		final_movie_library = append(final_movie_library, movies.MediaContainer.MediaContainer.Metadata...)

	}

	for _, show_library_key := range show_library_keys {
		shows, err := plexClient.GetLibraryContent(show_library_key, "")
		if err != nil {
			log.Fatalf("Could not get show library contents: %v", err)
		}
		final_show_library = append(final_show_library, shows.MediaContainer.MediaContainer.Metadata...)

	}

	var show_episodes []plex.MediaContainer //Each MediaContainer is an episode
	for _, show := range final_show_library {

		//With these keys, GetEpisodes actually gets the seasons
		//We will use it to get the seasons then get the season keys from the output
		season_data, err := plexClient.GetEpisodes(show.RatingKey)
		if err != nil {
			log.Fatalf("Could not get show season data: %v", err)
		}

		for _, season := range season_data.MediaContainer.Metadata {
			episode_data, err := plexClient.GetEpisodes(season.RatingKey)
			if err != nil {
				log.Fatalf("Could not get episode data: %v", err)
			}
			show_episodes = append(show_episodes, episode_data.MediaContainer)
		}

	}

	movies_json, err := json.Marshal(final_movie_library)
	if err != nil {
		log.Fatalf("Could not marshal json: %v", err)
	}

	episodes_json, err := json.Marshal(show_episodes)
	if err != nil {
		log.Fatalf("Could not marshal json: %v", err)
	}

	//Post the data to the server
	postData(conf["server_url"]+"/upload/movies", compressData(movies_json), conf["username"], false)
	postData(conf["server_url"]+"/upload/shows", compressData(episodes_json), conf["username"], false)

	os.WriteFile(dataDir+"local_dumps/movies.json.gz", compressData(movies_json), 0644)
	os.WriteFile(dataDir+"local_dumps/episodes.json.gz", compressData(episodes_json), 0644)

}
