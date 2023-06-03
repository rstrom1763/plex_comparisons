package main

import (
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
)

func compare() {

}

type User struct {
	Firstname string
	Lastname  string
	Username  string
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

func main() {

	generateSSL()

	dataDir := "./data/"
	//Ensure that the dataDir actually exists
	err := os.Mkdir(dataDir, os.ModePerm)
	if err != nil {
		if err.Error() == "mkdir ./data/: Cannot create a file when that file already exists." || err.Error() == "mkdir ./data/: file exists" {
			fmt.Print()
		} else {
			fmt.Println("Error creating folder:", err)
		}
	}

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

	fmt.Printf("Listening on port %v...\n", conf["port"])     //Notifies that server is running on X port
	r.RunTLS(":"+conf["port"], "./cert.pem", "./private.key") //Start running the Gin server

}
