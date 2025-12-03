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
	"database/sql"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
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
		log.Fatal("Error generating private key:", err)
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
		log.Fatal("Error creating certificate:", err)
		return
	}

	// Write the private key and certificate to files
	keyOut, err := os.Create("./private.key")
	if err != nil {
		log.Fatal("Error creating private key file:", err)
		return
	}
	defer keyOut.Close()

	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	certOut, err := os.Create("./cert.pem")
	if err != nil {
		log.Fatal("Error creating certificate file:", err)
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

func postData(serverUrl string, data []byte, username string, validateSsl bool) {
	if !validateSsl {
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

// Create the DB connection
func initDB(path string) (*sql.DB, error) {

	// Create the db connection
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("could not open DB: %s", err)
	}

	// Test if we can access the db properly
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("could not ping DB: %s", err)
	}

	return db, nil

}

// Get key from the env file
func env(key string) (string, error) {

	// load .env file
	err := godotenv.Load(DOTENV_PATH)
	if err != nil {
		return "", fmt.Errorf("error loading .env file: %s", err)
	}

	return os.Getenv(key), nil
}

func addNoHaveToPath(path string) string {
	prefix := path[:strings.LastIndex(path, ".")]
	fileExtension := path[strings.LastIndex(path, "."):]
	return prefix + "_no_have" + fileExtension
}

func humanReadableByteCountString(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div := float64(unit)
	exp := 0
	for n := float64(bytes) / div; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	value := float64(bytes) / div
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}

	if exp >= len(units) {
		exp = len(units) - 1
	}

	return fmt.Sprintf("%.2f %s", value, units[exp])
}

func getByteSumFromDumpFile(dumpPath string, mediaType string) (int64, error) {
	var byteSum int64

	if !fileExists(dumpPath) {
		log.Fatalf("%s does not exist", dumpPath)
	}

	items, err := getMediaItemsFromCSV(dumpPath, mediaType)
	if err != nil {
		return 0, fmt.Errorf("could not get media items from csv: %s", err.Error())
	}

	for _, item := range items {
		byteSum += item.GetSizeBytes()
	}

	return byteSum, nil

}
