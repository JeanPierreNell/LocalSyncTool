package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"

	"github.com/kardianos/service"
)

const serviceName = "Local Sync service"
const serviceDescription = "Simple Service to sync local linked directories."
const configLocation = "config.json"

type program struct{}

type Config struct {
	LinkedLocation []Location `json:"linkedLocation"`
}

type Location struct {
	LocationOne string `json:"LocationOne"`
	LocationTwo string `json:"LocationTwo"`
}

func (p program) Start(s service.Service) error {
	fmt.Println(s.String() + " started")
	go p.run()
	return nil
}

func (p program) Stop(s service.Service) error {
	fmt.Println(s.String() + " stopped")
	return nil
}

func (p program) run() {

	jsonFile, err := os.Open(configLocation)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully Opened " + configLocation)
	defer jsonFile.Close()

	configBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("Cannot Open Config File at " + configLocation)
	}

	var config Config
	json.Unmarshal(configBytes, &config)

	fmt.Println("Linked Directories")
	for i := 0; i < len(config.LinkedLocation); i++ {
		fmt.Println(config.LinkedLocation[i].LocationOne + " - " + config.LinkedLocation[i].LocationTwo)
		SyncDirectory(config.LinkedLocation[i].LocationOne, config.LinkedLocation[i].LocationTwo)
	}

}

func SyncDirectory(locationOne string, LocationTwo string) {
	entriesOne, err := os.ReadDir(locationOne)
	if err != nil {
		log.Fatal(err)
	}

	entriesTwo, err := os.ReadDir(LocationTwo)
	if err != nil {
		log.Fatal(err)
	}

	var longEntries []fs.DirEntry
	var shortEntries []fs.DirEntry

	var shortFileName string
	var longFileName string

	if len(entriesOne) > len(entriesTwo) {
		longEntries = entriesOne
		shortEntries = entriesTwo
		shortFileName = LocationTwo
		longFileName = locationOne
	} else {
		longEntries = entriesTwo
		shortEntries = entriesOne
		shortFileName = locationOne
		longFileName = LocationTwo
	}

	fmt.Println("Long Entries : " + longFileName)
	fmt.Println("Short Entries : " + shortFileName)

	for i := 0; i < len(longEntries); i++ {
		if !contains(shortEntries, longEntries[i]) {
			copy(longFileName+longEntries[i].Name(), shortFileName+longEntries[i].Name())
		}
	}
}

func contains(s []fs.DirEntry, str fs.DirEntry) bool {
	for _, v := range s {
		if v.Name() == str.Name() {
			return true
		}
	}

	return false
}

func copy(sourceFilePath string, DestinationFilePath string) {
	fmt.Println("Copying File: " + sourceFilePath + " to : " + DestinationFilePath)
	input, err := ioutil.ReadFile(sourceFilePath)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = ioutil.WriteFile(DestinationFilePath, input, 0644)
	if err != nil {
		fmt.Println("Error creating", DestinationFilePath)
		fmt.Println(err)
		return
	}
}

func main() {
	serviceConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceName,
		Description: serviceDescription,
	}

	prg := &program{}
	s, err := service.New(prg, serviceConfig)

	if err != nil {
		fmt.Println("Cannot create the service: " + err.Error())
	}

	err = s.Run()
	if err != nil {
		fmt.Println("Cannot start the service: " + err.Error())
	}
}
