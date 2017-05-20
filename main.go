package main

import (
	"github.com/boltdb/bolt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	logger, errorLog *log.Logger
	db               *bolt.DB
)

// Set up two loggers: logger for os.Stdout, and errorLog for error.log
func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

	fp, err := os.Create("error.log")
	if err != nil {
		logger.Println(err)
	}
	errorLog = log.New(fp, "", log.LstdFlags|log.Lmicroseconds)

	db, err = bolt.Open("feeds.boltdb", 0644, nil)
	if err != nil {
		logger.Fatalln(err)
	}
}

// cleanup closes the bolt database.
func cleanup() {
	db.Close()
}

func main() {
	config, err := loadConfig("config.toml")
	if err != nil {
		logger.Fatalln(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Println("\ncleaning up...")
		cleanup()
		logger.Println("shutting down")
		os.Exit(1)
	}()

	rc := NewRiverContainer(config)
	rc.Run()
}
