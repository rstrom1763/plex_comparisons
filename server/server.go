package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	nocache "github.com/alexander-melentyev/gin-nocache"
	"github.com/gin-gonic/gin"
)

func compare() {

}

func main() {

	dataDir := "./data/"

	//Read config json contents
	raw_conf, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalf("Could not read config file: %v", err)
	}

	//Unmarshall the raw json config into a string map for use in the code
	var conf map[string]string
	err = json.Unmarshal(raw_conf, &conf)
	if err != nil {
		log.Fatalf("Could not unmarshall conf json: %v", err)
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

		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Fatalf("Could not read request body: %v", err)
		}

		os.WriteFile(dataDir+"movies.json.gz", data, 0644)

	})

	r.POST("/upload/shows/episodes", func(c *gin.Context) {
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Fatalf("Could not read request body: %v", err)
		}

		os.WriteFile(dataDir+"episodes.json.gz", data, 0644)
	})

	r.POST("/upload/shows/seasons", func(c *gin.Context) {
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Fatalf("Could not read request body: %v", err)
		}

		os.WriteFile(dataDir+"seasons.json.gz", data, 0644)
	})

	fmt.Printf("Listening on port %v...\n", conf["port"]) //Notifies that server is running on X port
	r.Run(":" + conf["port"])                             //Start running the Gin server
}
