package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	nocache "github.com/alexander-melentyev/gin-nocache"
	"github.com/gin-gonic/gin"
	"github.com/jrudio/go-plex-client"
)

func returnErr(c *gin.Context, statusCode int, err error) {
	c.Data(statusCode, "text/plain", []byte(err.Error()))
}

func initMap(userObjects *[]plex.Metadata, userMap map[string]Movie) {

	for _, movie := range *userObjects {
		newMovie := Movie{movie}
		_, exists := userMap[newMovie.getTitle()]

		if exists {
			fmt.Println("Movie with Duplicate name: " + movie.Title)
		} else {
			userMap[newMovie.getTitle()] = newMovie
		}

	}
}

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

type User struct {
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Username  string `json:"username"`
}

type Movie struct {
	MetaDataObject plex.Metadata
}

func (m Movie) getTitle() string {
	return m.MetaDataObject.Title
}

func (m Movie) getYear() int {
	return m.MetaDataObject.Year
}

func (m Movie) getMetadata() plex.Metadata {
	return m.MetaDataObject
}

func compressData(data []byte) []byte {
	var compressedData bytes.Buffer

	// Create a new Gzip Writer, providing the compressedData buffer
	gzipWriter := gzip.NewWriter(&compressedData)

	// Write the data to the Gzip Writer
	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil
	}

	// Close the Gzip Writer to flush any remaining data
	err = gzipWriter.Close()
	if err != nil {
		return nil
	}

	// Return the compressed data as a byte slice
	return compressedData.Bytes()
}

func decompressData(data []byte) []byte {
	// Create a buffer with the data bytes
	buf := bytes.NewReader(data)

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(buf)
	if err != nil {
		return nil
	}
	defer gzipReader.Close()

	// Read the decompressed data from the gzip reader
	decompressedData, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil
	}

	return decompressedData
}

func generateSSL() {

	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Error generating private key:", err)
		return
	}

	// Generate a self-signed certificate
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		fmt.Println("Error creating certificate:", err)
		return
	}

	// Write the private key and certificate to files
	keyOut, err := os.Create("./private.key")
	if err != nil {
		fmt.Println("Error creating private key file:", err)
		return
	}
	defer keyOut.Close()

	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	certOut, err := os.Create("./cert.pem")
	if err != nil {
		fmt.Println("Error creating certificate file:", err)
		return
	}
	defer certOut.Close()

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	fmt.Println("TLS certificate and private key generated successfully.")
}

func ensureFolderExists(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the folder if it doesn't exist
			err = os.MkdirAll(path, 0755)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func ensureFileExists(path string) {

	// Check if the file exists
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			log.Fatalf("Could not create file %v: %v", path, err)
		}
		file.Close()
	}
}

func createTarArchive(file1Data []byte, file1Name string, file2Data []byte, file2Name string) ([]byte, error) {
	// Create a buffer to hold the tar archive
	var buf bytes.Buffer

	// Create a tar writer
	tarWriter := tar.NewWriter(&buf)
	defer tarWriter.Close()

	// Create file 1 in the tar archive
	file1Header := &tar.Header{
		Name: file1Name,
		Mode: 0644, // Set appropriate file permissions
		Size: int64(len(file1Data)),
	}
	if err := tarWriter.WriteHeader(file1Header); err != nil {
		return nil, fmt.Errorf("failed to write tar header for file 1: %w", err)
	}
	if _, err := tarWriter.Write(file1Data); err != nil {
		return nil, fmt.Errorf("failed to write file 1 data to tar archive: %w", err)
	}

	// Create file 2 in the tar archive
	file2Header := &tar.Header{
		Name: file2Name,
		Mode: 0644, // Set appropriate file permissions
		Size: int64(len(file2Data)),
	}
	if err := tarWriter.WriteHeader(file2Header); err != nil {
		return nil, fmt.Errorf("failed to write tar header for file 2: %w", err)
	}
	if _, err := tarWriter.Write(file2Data); err != nil {
		return nil, fmt.Errorf("failed to write file 2 data to tar archive: %w", err)
	}

	// Close the tar writer to flush any remaining data
	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Return the tar archive data as a []byte
	return buf.Bytes(), nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true // File exists
	}
	if os.IsNotExist(err) {
		return false // File does not exist
	}
	return false // Error occurred (e.g., permission denied)
}

func sendAsFile(c *gin.Context, data []byte, filename string) {

	err := os.WriteFile("./"+filename, data, 0644)
	if err != nil {
		log.Fatalf("Could not write file: %v", err)
	}
	defer os.Remove("./" + filename)

	// Set the appropriate headers for the download
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/plain")
	c.Header("Content-Length", string(len(data)))

	// Write the text content as the response body
	c.File("./" + filename)

}

func main() {

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

	//Read config json contents
	ensureFileExists("./config.json")
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

		diffArchive, err := createTarArchive([]byte(diff1), diff1Name, []byte(diff2), diff2Name)
		if err != nil {
			log.Fatalf("Could not create Tar archive: %v", err)
		}

		sendAsFile(c, compressData(diffArchive), "archive.tar.gz")

	})

	fmt.Printf("Listening on port %v...\n", conf["port"])     //Notifies that server is running on X port
	r.RunTLS(":"+conf["port"], "./cert.pem", "./private.key") //Start running the Gin server

}
