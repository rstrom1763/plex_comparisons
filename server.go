package main

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	utils "github.com/rstrom1763/goUtils"
	"github.com/rstrom1763/plex_comparisons/DAOS"
	"github.com/rstrom1763/plex_comparisons/constants"
	"github.com/rstrom1763/plex_comparisons/structs"
)

func RunServer() error {

	err := godotenv.Load(constants.DOTENV_PATH)
	if err != nil {
		return fmt.Errorf("could not load .env: %s", err.Error())
	}

	port := os.Getenv("PORT")
	protocol := os.Getenv("PROTOCOL")
	plexDbPath := os.Getenv("PLEX_DB_PATH")
	plexDataPath := os.Getenv("PLEX_DATA_PATH")
	localDbPath := os.Getenv("LOCAL_DB_PATH")
	if localDbPath == "" {
		localDbPath = "./local_state.db"
	}

	localDAO, err := DAOS.NewLocalStateDAO(localDbPath)
	if err != nil {
		return fmt.Errorf("could not initialize local state DAO: %w", err)
	}
	defer func() {
		_ = localDAO.Close()
	}()

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

	gin.SetMode(gin.ReleaseMode) // Turn off debugging mode
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

	r.GET("/dump/movies", func(c *gin.Context) {
		jsonData, err := os.ReadFile("./test_movies.json")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "could not read test movies file: " + err.Error(),
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

	r.GET("/dump/songs", func(c *gin.Context) {
		songs, err := structs.GetSongs(plexDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		jsonData, err := json.Marshal(songs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "could not marshal songs: " + err.Error(),
			})
			return
		}

		compressedData := utils.GzipData(jsonData)
		if compressedData == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "could not gzip songs data",
			})
			return
		}

		c.Header("Content-Encoding", "gzip")
		c.Data(http.StatusOK, "application/json", compressedData)
	})

	r.GET("/thumb/:hash", func(c *gin.Context) {
		hash := c.Param("hash")

		if plexDataPath == "" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "PLEX_DATA_PATH not set in environment",
			})
			return
		}

		if hash == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "no metadata hash provided",
			})
			return
		}

		posterDir := plexMoviePosterDir(plexDataPath, hash)
		posterPath, err := findPosterFile(posterDir)
		if err != nil {
			// Serve a placeholder SVG for missing thumbnails
			placeholder := `<svg width="200" height="300" xmlns="http://www.w3.org/2000/svg">
				<rect width="100%" height="100%" fill="#333"/>
				<text x="50%" y="50%" font-family="Arial" font-size="16" fill="#fff" text-anchor="middle" dy=".3em">No Poster</text>
			</svg>`
			c.Data(http.StatusOK, "image/svg+xml", []byte(placeholder))
			return
		}

		c.File(posterPath)
	})

	r.Static("/static", "./static")

	r.GET("/movies/gallery", func(c *gin.Context) {
		c.File("./static/html/index.html")
	})

	r.GET("/compare", func(c *gin.Context) {
		c.File("./static/html/compare.html")
	})

	r.GET("/add-server", func(c *gin.Context) {
		c.File("./static/html/add_server.html")
	})

	// Local State API
	r.GET("/api/servers", func(c *gin.Context) {
		servers, err := localDAO.GetServers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, servers)
	})

	r.POST("/api/servers", func(c *gin.Context) {
		var server structs.Server
		if err := c.ShouldBindJSON(&server); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := localDAO.AddServer(server); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusCreated)
	})

	r.DELETE("/api/servers/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, _ := strconv.Atoi(idStr)
		if err := localDAO.DeleteServer(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.GET("/api/compare/:id", func(c *gin.Context) {
		// Comparison logic will be triggered here
		// For now, just a placeholder that fetches the other server's dump
		idStr := c.Param("id")
		id, _ := strconv.Atoi(idStr)
		servers, _ := localDAO.GetServers()
		var target structs.Server
		for _, s := range servers {
			if s.ID == id {
				target = s
				break
			}
		}

		if target.Address == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
			return
		}

		// Simplified comparison: just return what we have vs what they have
		localMovies, err := structs.GetMovies(plexDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Disable TLS verification for remote servers
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Get(target.Address + "/test/dump/movies")
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "could not reach remote server: " + err.Error()})
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		var remoteMovies []*structs.Movie
		if err := json.NewDecoder(resp.Body).Decode(&remoteMovies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode remote dump: " + err.Error()})
			return
		}

		remoteOnly, localOnly := compareDumps(localMovies, remoteMovies)

		c.JSON(http.StatusOK, gin.H{
			"local_only":  localOnly,
			"remote_only": remoteOnly,
		})
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

func plexMoviePosterDir(plexDataDir string, hash string) string {
	if hash == "" {
		return ""
	}

	return filepath.Join(
		plexDataDir,
		"Metadata",
		"Movies",
		hash[:1],
		hash[1:]+".bundle",
		"Contents",
		"_combined",
		"posters",
	)
}

func findPosterFile(posterDir string) (string, error) {
	entries, err := os.ReadDir(posterDir)
	if err != nil {
		return "", err
	}

	var largestFile string
	var maxSize int64 = -1

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		if info.Size() > maxSize {
			maxSize = info.Size()
			largestFile = filepath.Join(posterDir, e.Name())
		}
	}

	if largestFile == "" {
		return "", fmt.Errorf("no poster file found in %s", posterDir)
	}

	return largestFile, nil
}
