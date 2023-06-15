package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	nocache "github.com/alexander-melentyev/gin-nocache"
	"github.com/gin-gonic/gin"
	"github.com/jrudio/go-plex-client"
)

// Find all items in movieMap1 that are not in movieMap2
func findNotIn(userObjects []plex.Metadata, moviesMap2 map[string]Movie) []Movie {
	var notIn []Movie

	for _, movie := range userObjects {
		movieObject := Movie{movie}
		_, exists := moviesMap2[movieObject.getTitle()]

		if !exists {
			notIn = append(notIn, movieObject)
		}
	}

	return notIn

}

func extractMetadata(movies []Movie) []plex.Metadata {
	var output []plex.Metadata
	for _, movie := range movies {
		output = append(output, movie.getMetadata())
	}
	return output
}

func compareMovies(user1Data []byte, user2Data []byte) string {

	var user1Objects []plex.Metadata
	var user2Objects []plex.Metadata
	//var user1Movies []Movie
	//var user2Movies []Movie
	//var user1Map map[string]Movie
	//var user2Map map[string]Movie
	//var diff []Movie
	//var json_string_diff string
	user1Map := make(map[string]Movie)
	user2Map := make(map[string]Movie)

	err := json.Unmarshal(user1Data, &user1Objects)
	if err != nil {
		log.Fatalf("Could not unmarshal JSON: %v", err)
	}
	err = json.Unmarshal(user2Data, &user2Objects)
	if err != nil {
		log.Fatalf("Could not unmarshal JSON: %v", err)
	}

	initMap(&user1Objects, user1Map)
	initMap(&user2Objects, user2Map)

	diff := findNotIn(user1Objects, user2Map)
	output, err := json.Marshal(extractMetadata(diff))
	if err != nil {
		log.Fatalf("Could not marshal JSON: %v", err)
	}
	return string(output)
}

func runServer(conf map[string]string) {

	if !(fileExists("./cert.pem") && fileExists("./private.key")) {
		generateSSL()
	}

	dataDir := "./data/"
	ensureFolderExists(dataDir)

	var users map[string]User
	userJSONFilePath := dataDir + "users.json"
	ensureFileExists(userJSONFilePath)
	usersJSON, err := os.ReadFile(userJSONFilePath)
	if err != nil {
		log.Fatalf("Could not read the user database: %v", err)
	}
	//Init the userJSON if the file is empty
	if string(usersJSON) == "" {
		users = make(map[string]User)
	} else {
		json.Unmarshal(usersJSON, &users)
	}

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode) // Turn off debugging mode
	r := gin.Default()           // Initialize Gin
	r.Use(nocache.NoCache())     // Sets gin to disable browser caching

	//Route for health check
	r.GET("/ping", func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})

	})

	r.POST("/upload/movies", func(c *gin.Context) {

		username := c.Request.Header.Get("username")

		_, exists := users[username]
		if exists {
			data, err := io.ReadAll(c.Request.Body)
			if err != nil {
				log.Fatalf("Could not read request body: %v", err)
			}
			ensureFolderExists(dataDir + "dumps/" + username)
			filename := fmt.Sprintf("%v/dumps/%v/movies.json.gz", dataDir, username)
			os.WriteFile(filename, data, 0644)
			c.Data(200, "text/plain", []byte("Success"))
		} else {
			c.Data(404, "text/plain", []byte("User not found"))
		}

	})

	r.POST("/upload/shows", func(c *gin.Context) {

		username := c.Request.Header.Get("username")

		_, exists := users[username]
		if exists {
			data, err := io.ReadAll(c.Request.Body)
			if err != nil {
				log.Fatalf("Could not read request body: %v", err)
			}
			ensureFolderExists(dataDir + "dumps/" + username)
			filename := fmt.Sprintf("%vdumps/%v/episodes.json.gz", dataDir, username)
			os.WriteFile(filename, data, 0644)
		} else {
			c.Data(404, "text/plain", []byte("User not found"))
		}

	})

	r.POST("/user/new", func(c *gin.Context) {
		firstName := c.Request.Header.Get("firstname")
		lastName := c.Request.Header.Get("lastname")
		username := c.Request.Header.Get("username")

		_, exists := users[username]
		if exists {
			c.Data(400, "text/plain", []byte("User already exists!"))
		} else {
			users[username] = User{firstName, lastName, username}

			userDump, err := json.Marshal(users)
			if err != nil {
				fmt.Println("There was an error marshalling user json data")
			}

			os.WriteFile("./data/users.json", userDump, 0644)
			c.Data(200, "text/plain", []byte("User created!"))
		}

	})

	r.GET("/compare/:user1/:user2", func(c *gin.Context) {
		user1 := c.Param("user1")
		user2 := c.Param("user2")

		var logOutput bytes.Buffer
		log.SetOutput(&logOutput)

		user1Dump, err := os.ReadFile(dataDir + "dumps/" + user1 + "/movies.json.gz")
		if err != nil {
			log.Fatalf("Could not read file: %v", err)
		}
		user2Dump, err := os.ReadFile(dataDir + "dumps/" + user2 + "/movies.json.gz")
		if err != nil {
			log.Fatalf("Could not read file: %v", err)
		}

		diff1 := compareMovies(decompressData(user1Dump), decompressData(user2Dump))
		diff2 := compareMovies(decompressData(user2Dump), decompressData(user1Dump))

		diff1Name := user2 + "_no_have.json"
		diff2Name := user1 + "_no_have.json"

		//Add files to File struct list. This will be fed into createTarArchive
		var outputFiles []File                                                        //Holds the file objects
		outputFiles = append(outputFiles, File{Data: []byte(diff1), Name: diff1Name}) //Append diff1
		outputFiles = append(outputFiles, File{Data: []byte(diff2), Name: diff2Name}) //Apped diff2
		outputFiles = append(outputFiles, File{Data: logOutput.Bytes(), Name: "compare.log"})

		diffArchive, err := createTarArchive(outputFiles)
		if err != nil {
			log.Fatalf("Could not create Tar archive: %v", err)
		}

		sendAsFile(c, compressData(diffArchive), "archive.tar.gz")

	})

	fmt.Printf("Listening on port %v...\n", conf["port"])     //Notifies that server is running on X port
	r.RunTLS(":"+conf["port"], "./cert.pem", "./private.key") //Start running the Gin server
}
