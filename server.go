package main

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	utils "github.com/rstrom1763/goUtils"
	"github.com/rstrom1763/plex_comparisons/DAOS"
	"github.com/rstrom1763/plex_comparisons/auth"
	"github.com/rstrom1763/plex_comparisons/constants"
	"github.com/rstrom1763/plex_comparisons/structs"
)

func AuthMiddleware(localDAO *DAOS.LocalStateDAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		protocol := os.Getenv("PROTOCOL")
		secure := protocol == "https"

		// 1. Check for shared token in header (Server-to-Server)
		sharedToken := c.GetHeader("X-Server-Token")
		if sharedToken != "" {
			trustedServers, err := localDAO.GetTrustedServers()
			if err == nil {
				for _, ts := range trustedServers {
					if auth.VerifySecret(sharedToken, ts.TokenHash) {
						c.Next()
						return
					}
				}
			}
		}

		// 2. Check for session cookie (UI)
		sessionToken, err := c.Cookie("session_token")
		if err == nil {
			if _, ok := auth.ValidateSession(sessionToken); ok {
				// CSRF Protection for state-changing methods
				if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" {
					csrfCookie, _ := c.Cookie("csrf_token")
					csrfHeader := c.GetHeader("X-CSRF-Token")
					if csrfCookie == "" || csrfHeader == "" || csrfCookie != csrfHeader {
						c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token mismatch"})
						return
					}
				}

				c.SetSameSite(http.SameSiteLaxMode)
				c.SetCookie("session_token", sessionToken, 1800, "/", "", secure, true)
				c.Next()
				return
			}
		}

		// 3. Fallback: If it's an API request, return 401. If it's a UI request, redirect to login.
		if c.Request.URL.Path == "/login" || c.Request.URL.Path == "/auth/login" {
			c.Next()
			return
		}

		// If API request, return 401
		if strings.HasPrefix(c.Request.URL.Path, "/api/") || strings.HasPrefix(c.Request.URL.Path, "/video/") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Allow public access to static files (you might want to secure these too, but usually /login needs them)
		// Or just redirect everything else to /login
		c.Redirect(http.StatusSeeOther, "/login")
		c.Abort()
	}
}

func RunServer() error {
	// Ensure thumb_cache directory exists
	if err := os.MkdirAll("thumb_cache", 0755); err != nil {
		log.Printf("Warning: could not create thumb_cache directory: %v", err)
	}

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

	// Initial user setup check
	userCount, _ := localDAO.GetUserCount()

	// Register global static files before auth
	r.Static("/static", "./static")
	r.LoadHTMLGlob("static/html/*")

	r.GET("/login", func(c *gin.Context) {
		if userCount == 0 {
			c.HTML(http.StatusOK, "setup.html", nil)
			return
		}
		c.HTML(http.StatusOK, "login.html", nil)
	})

	r.POST("/setup", func(c *gin.Context) {
		if userCount > 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "setup already completed"})
			return
		}
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		hash, _ := auth.HashSecret(req.Password)
		if err := localDAO.AddUser(req.Username, hash); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		userCount = 1
		c.JSON(http.StatusOK, gin.H{"message": "user created"})
	})

	r.POST("/login", func(c *gin.Context) {
		protocol := os.Getenv("PROTOCOL")
		secure := protocol == "https"
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := localDAO.GetUserByUsername(req.Username)
		if err != nil || user == nil || !auth.VerifySecret(req.Password, user.PasswordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		token, _ := auth.CreateSession(user.Username)
		csrfToken, _ := auth.GenerateRandomToken()

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("session_token", token, 1800, "/", "", secure, true)
		c.SetCookie("csrf_token", csrfToken, 1800, "/", "", secure, false) // Accessible by JS
		c.JSON(http.StatusOK, gin.H{"message": "logged in"})
	})

	// Secure routes
	authorized := r.Group("/")
	authorized.Use(AuthMiddleware(localDAO))

	authorized.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	authorized.POST("/logout", func(c *gin.Context) {
		protocol := os.Getenv("PROTOCOL")
		secure := protocol == "https"
		token, _ := c.Cookie("session_token")
		auth.DeleteSession(token)
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("session_token", "", -1, "/", "", secure, true)
		c.SetCookie("csrf_token", "", -1, "/", "", secure, false)
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	})

	authorized.GET("/trusted-servers", func(c *gin.Context) {
		c.HTML(http.StatusOK, "trusted_servers.html", nil)
	})

	authorized.GET("/api/trusted-servers", func(c *gin.Context) {
		servers, err := localDAO.GetTrustedServers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, servers)
	})

	authorized.POST("/api/trusted-servers", func(c *gin.Context) {
		var req struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token, _ := auth.GenerateRandomToken()
		hash, _ := auth.HashSecret(token)

		if err := localDAO.AddTrustedServer(req.Name, hash); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	authorized.DELETE("/api/trusted-servers/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		if err := localDAO.DeleteTrustedServer(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	})

	authorized.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	authorized.GET("/api/movies", func(c *gin.Context) {
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

	authorized.GET("/api/episodes", func(c *gin.Context) {
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

	authorized.GET("/api/songs", func(c *gin.Context) {
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

	authorized.GET("/thumb/:hash", func(c *gin.Context) {
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

	r.GET("/video/:hash", func(c *gin.Context) {
		hash := c.Param("hash")
		if hash == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no hash provided"})
			return
		}

		movies, err := structs.GetMovies(plexDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var targetMovie *structs.Movie
		for _, m := range movies {
			if m.Hash == hash {
				targetMovie = m
				break
			}
		}

		if targetMovie == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
			return
		}

		videoPath := targetMovie.File
		if _, err := os.Stat(videoPath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "video file not found on disk",
				"path":  videoPath,
			})
			return
		}

		// Ensure we support range requests for streaming
		c.Header("Accept-Ranges", "bytes")
		c.File(videoPath)
	})

	r.GET("/remote-thumb/:id/:hash", func(c *gin.Context) {
		idStr := c.Param("id")
		hash := c.Param("hash")
		id, _ := strconv.Atoi(idStr)

		if hash == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no hash provided"})
			return
		}

		cachePath := filepath.Join("thumb_cache", hash+".jpg")
		if _, err := os.Stat(cachePath); err == nil {
			c.File(cachePath)
			return
		}

		// Not in cache, fetch from remote
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

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Get(target.Address + "/thumb/" + hash)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "could not reach remote server: " + err.Error()})
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			c.JSON(resp.StatusCode, gin.H{"error": "remote server returned error"})
			return
		}

		thumbData, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read remote thumbnail"})
			return
		}

		// Save to cache
		_ = os.WriteFile(cachePath, thumbData, 0644)

		c.Data(http.StatusOK, resp.Header.Get("Content-Type"), thumbData)
	})

	authorized.GET("/movies/gallery", func(c *gin.Context) {
		c.File("./static/html/index.html")
	})

	authorized.GET("/compare", func(c *gin.Context) {
		c.File("./static/html/compare.html")
	})

	authorized.GET("/add-server", func(c *gin.Context) {
		c.File("./static/html/add_server.html")
	})

	authorized.GET("/duplicates", func(c *gin.Context) {
		c.File("./static/html/duplicates.html")
	})

	// Local State API
	authorized.GET("/api/duplicates", func(c *gin.Context) {
		movies, err := structs.GetMovies(plexDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		duplicates := make(map[string][]*structs.Movie)
		for _, m := range movies {
			key := m.GetUniqueTitle()
			duplicates[key] = append(duplicates[key], m)
		}

		var result [][]*structs.Movie
		for _, group := range duplicates {
			if len(group) > 1 {
				result = append(result, group)
			}
		}

		c.JSON(http.StatusOK, result)
	})

	authorized.GET("/api/servers", func(c *gin.Context) {
		servers, err := localDAO.GetServers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, servers)
	})

	authorized.GET("/api/filters", func(c *gin.Context) {
		filters, err := localDAO.GetSavedFilters()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, filters)
	})

	authorized.POST("/api/filters", func(c *gin.Context) {
		var filter structs.SavedFilter
		if err := c.ShouldBindJSON(&filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		id, err := localDAO.AddSavedFilter(filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	})

	authorized.DELETE("/api/filters/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, _ := strconv.Atoi(idStr)
		if err := localDAO.DeleteSavedFilter(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	authorized.PUT("/api/filters/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, _ := strconv.Atoi(idStr)
		var filter structs.SavedFilter
		if err := c.ShouldBindJSON(&filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		filter.ID = id
		if err := localDAO.UpdateSavedFilter(filter); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	authorized.POST("/api/servers", func(c *gin.Context) {
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

	authorized.DELETE("/api/servers/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, _ := strconv.Atoi(idStr)
		if err := localDAO.DeleteServer(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.DELETE("/api/file", func(c *gin.Context) {
		var req struct {
			Hash string `json:"hash"`
			Path string `json:"path"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Hash == "" && req.Path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hash or path is required"})
			return
		}

		// Retrieve path from metadata
		var filePath string
		var err error
		if req.Path != "" {
			err = plexDB.QueryRow("SELECT file FROM media_parts WHERE file = ?", req.Path).Scan(&filePath)
		} else {
			err = plexDB.QueryRow("SELECT file FROM media_parts WHERE hash = ?", req.Hash).Scan(&filePath)
		}

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "file not found in metadata"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("database error: %v", err)})
			}
			return
		}

		// Security check: ensure the file exists before trying to delete
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found on disk"})
			return
		}

		if err := os.Remove(filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete file: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
	})

	authorized.PUT("/api/servers/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, _ := strconv.Atoi(idStr)
		var server structs.Server
		if err := c.ShouldBindJSON(&server); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		server.ID = id
		if err := localDAO.UpdateServer(server); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
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

		req, err := http.NewRequest("GET", target.Address+"/api/movies", nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create request: " + err.Error()})
			return
		}

		// Set auth headers if token is present
		if target.Token != "" {
			req.Header.Set("X-Server-Token", target.Token)
		}

		// Disable TLS verification for remote servers
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "could not reach remote server: " + err.Error()})
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode == http.StatusUnauthorized {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "remote server rejected authentication. Ensure tokens match."})
			return
		}

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
