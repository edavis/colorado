package main

import "fmt"

func main() {
	config, err := loadConfig("config.toml")
	if err != nil {
		panic(err)
	}

	fmt.Println("starting up...")

	rc := NewRiverContainer(config)
	rc.Run()
}
