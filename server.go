package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	utils "github.com/rstrom1763/goUtils"
	"github.com/rstrom1763/plex_comparisons/structs"
)

func StartServer() error {

	err := godotenv.Load(".env")
	if err != nil {
		return fmt.Errorf("could not load .env: %s", err.Error())
	}

	port := os.Getenv("PORT")
	protocol := os.Getenv("PROTOCOL")
	plexDbPath := os.Getenv("PLEX_DB_PATH")

	plexDB, err := initDB(plexDbPath)
	if err != nil {
		return fmt.Errorf("there was an error initializing the DB connection: " + err.Error())
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal("could not close Plex database: ", err)
		}
	}(plexDB)

	//Generate TLS keys if they do not already exist
	if !(utils.FileExists("./cert.pem") && utils.FileExists("./private.key")) && protocol == "https" {
		utils.GenerateSSL()
	}

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/dump/movies", func(c *gin.Context) {
		movies, err := structs.GetMovies(plexDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		jsonData, err := json.Marshal(movies)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "could not marshal movies: " + err.Error(),
			})
			return
		}

		compressedData := utils.GzipData(jsonData)
		if compressedData == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "could not gzip movies data",
			})
			return
		}

		c.Header("Content-Encoding", "gzip")
		c.Data(http.StatusOK, "application/json", compressedData)
	})

	r.GET("/dump/episodes", func(c *gin.Context) {
		episodes, err := structs.GetEpisodes(plexDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		jsonData, err := json.Marshal(episodes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "could not marshal episodes: " + err.Error(),
			})
			return
		}

		compressedData := utils.GzipData(jsonData)
		if compressedData == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "could not gzip episodes data",
			})
			return
		}

		c.Header("Content-Encoding", "gzip")
		c.Data(http.StatusOK, "application/json", compressedData)
	})

	fmt.Printf("Listening for %v on port %v...\n", protocol, port) //Notifies that server is running on X port
	if protocol == "http" {                                        //Start running the Gin server
		err = r.Run(":" + port)
		if err != nil {
			fmt.Println(err)
		}
	} else if protocol == "https" {
		err = r.RunTLS(":"+port, "./cert.pem", "./private.key")
		if err != nil {
			fmt.Println(err)
		}
	} else {
		log.Fatal("Something went wrong starting the Gin server")
	}

	return nil
}
