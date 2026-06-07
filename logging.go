package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func configureLoggingFromEnv() (func(), error) {
	return configureLogging(os.Getenv("LOG_FILE"))
}

func configureLogging(logFilePath string) (func(), error) {
	if logFilePath == "" {
		log.SetOutput(os.Stdout)
		gin.DefaultWriter = os.Stdout
		gin.DefaultErrorWriter = os.Stderr
		return func() {}, nil
	}

	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	writer := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(writer)
	gin.DefaultWriter = writer
	gin.DefaultErrorWriter = writer

	return func() {
		_ = logFile.Close()
	}, nil
}
