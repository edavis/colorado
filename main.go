package main

import (
	"log"
	"os"
)

var (
	logger, errorLog *log.Logger
)

// Set up two loggers: logger for os.Stdout, and errorLog for error.log
func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

	fp, err := os.Create("error.log")
	if err != nil {
		logger.Println(err)
	}
	errorLog = log.New(fp, "", log.LstdFlags|log.Lmicroseconds)
}

func main() {
	config, err := loadConfig("config.toml")
	if err != nil {
		logger.Fatalln(err)
	}

	rc := NewRiverContainer(config)
	rc.Run()
}
