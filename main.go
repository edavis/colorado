package main

import (
	"flag"
	"github.com/boltdb/bolt"
	"github.com/fsnotify/fsnotify"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	logger, errorLog   *log.Logger
	db                 *bolt.DB
	dbPath, configPath string
	watcher            *fsnotify.Watcher
)

// Set up two loggers: logger for os.Stdout, and errorLog for error.log
func init() {
	flag.StringVar(&dbPath, "database", "feeds.db", "path to BoltDB database")
	flag.StringVar(&configPath, "config", "config.toml", "path to TOML config")
	flag.Parse()

	logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

	fp, err := os.Create("error.log")
	if err != nil {
		logger.Println(err)
	}
	errorLog = log.New(fp, "", log.LstdFlags|log.Lmicroseconds)

	db, err = bolt.Open(dbPath, 0644, nil)
	if err != nil {
		logger.Fatalln(err)
	}

	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		logger.Fatalln(err)
	}
}

// cleanup closes the bolt database.
func cleanup() {
	if err := watcher.Close(); err != nil {
		logger.Printf("problem closing watcher: %v", err)
	}
	db.Close()
}

func main() {
	logger.Println("starting up")

	config, err := loadConfig(configPath)
	if err != nil {
		logger.Fatalln(err)
	}

	if err = watcher.Add(configPath); err != nil {
		logger.Fatalln(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Println("cleaning up")
		cleanup()
		logger.Println("shutting down")
		os.Exit(1)
	}()

	rc := NewRiverContainer(config)
	rc.Run()
}
