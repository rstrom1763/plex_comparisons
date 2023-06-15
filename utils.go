package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
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
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jrudio/go-plex-client"
)

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

func createTarArchive(files []File) ([]byte, error) {
	// Create a buffer to hold the tar archive
	var buf bytes.Buffer

	// Create a tar writer
	tarWriter := tar.NewWriter(&buf)
	defer tarWriter.Close()

	for _, file := range files {
		fileHeader := &tar.Header{
			Name: file.Name,
			Mode: 0644, // Set appropriate file permissions
			Size: int64(len(file.Data)),
		}
		if err := tarWriter.WriteHeader(fileHeader); err != nil {
			return nil, fmt.Errorf("failed to write tar header for file %v: %w", file.Name, err)
		}
		if _, err := tarWriter.Write(file.Data); err != nil {
			return nil, fmt.Errorf("failed to write %v data to tar archive: %w", file.Name, err)
		}
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

// Clears the screen
func clear() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
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
func postData(serverUrl string, data []byte, username string, validate_ssl bool) {
	if !validate_ssl {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	r := bytes.NewReader(data)
	req, err := http.NewRequest("POST", serverUrl, r)
	if err != nil {
		fmt.Printf("Could not create request: %v\n", err)
		return
	}

	// Set the desired header
	req.Header.Set("username", username)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Could not post to the server: %v\n", err)
		return
	}
	defer resp.Body.Close()

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

func initConf() map[string]string {

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
	return conf

}

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
