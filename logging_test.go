package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestConfigureLoggingWritesStandardAndGinLogsToFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "server.log")

	cleanup, err := configureLogging(logPath)
	if err != nil {
		t.Fatalf("configureLogging() error = %v", err)
	}
	defer func() {
		cleanup()
		log.SetOutput(os.Stderr)
		gin.DefaultWriter = os.Stdout
		gin.DefaultErrorWriter = os.Stderr
	}()

	log.Println("standard logger message")
	if _, err := gin.DefaultWriter.Write([]byte("gin logger message\n")); err != nil {
		t.Fatalf("gin.DefaultWriter.Write() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(logPath) error = %v", err)
	}
	logs := string(data)
	if !strings.Contains(logs, "standard logger message") {
		t.Fatalf("log file = %q, want standard logger message", logs)
	}
	if !strings.Contains(logs, "gin logger message") {
		t.Fatalf("log file = %q, want gin logger message", logs)
	}
}
