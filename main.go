package main

import "fmt"

func main() {
	config, err := loadConfig("config.toml")
	if err != nil {
		panic(err)
	}

	fmt.Println("starting up...")

	rc := NewRiverContainer()
	for _, river := range config.River {
		rc.Add(river.Name, river.Feeds)
	}
	rc.Run()
}
