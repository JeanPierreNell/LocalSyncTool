package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/kardianos/service"
)

const serviceName = "Local Sync service"
const serviceDescription = "Simple Service to sync local linked directories."
const configLocation = "config.json"
const chunkSize = 64000

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

	// var filesForOne []string
	// var filesForTwo []string

	// for i := 0; i < len(entriesOne); i++ {
	// 	if !contains(entriesTwo, entriesOne[i]) {
	// 		filesForOne = append(filesForOne, entriesOne[i].Name())
	// 	}
	// }

	// for i := 0; i < len(entriesTwo); i++ {
	// 	if !contains(entriesOne, entriesTwo[i]) {
	// 		filesForTwo = append(filesForTwo, entriesTwo[i].Name())
	// 	}
	// }

	// compareDirectories(entriesOne, entriesTwo)
	missingFiles1, nonUniqueFiles := missingFiles(entriesTwo, entriesOne)
	fmt.Println("Missing Files:")
	fmt.Println(nonUniqueFiles)
	outdatedFiles := nonUniqueFileCompare(nonUniqueFiles, locationOne, LocationTwo)

	filesForLocationOne := append(missingFiles1, outdatedFiles...)

	copyFiles(filesForLocationOne, locationOne, LocationTwo)

	missingFiles2, nonUniqueFiles := missingFiles(entriesOne, entriesTwo)
	fmt.Println("Missing Files:")
	fmt.Println(nonUniqueFiles)
	outdatedFiles2 := nonUniqueFileCompare(nonUniqueFiles, LocationTwo, locationOne)

	filesForLocationTwo := append(missingFiles2, outdatedFiles2...)

	copyFiles(filesForLocationTwo, LocationTwo, locationOne)
}

func copyFiles(files []string, source string, destination string) {
	for i := 0; i < len(files); i++ {
		copy(source+files[i], destination+files[i])
	}
}

func compareDirectories(sourceDirectory []fs.DirEntry, comparisonDirectory []fs.DirEntry) {

	var outdatedFiles []string
	var nonUniqueFiles []fs.DirEntry

	for i := 0; i < len(comparisonDirectory); i++ {
		if !contains(sourceDirectory, comparisonDirectory[i]) {
			outdatedFiles = append(outdatedFiles, comparisonDirectory[i].Name())
		} else {
			nonUniqueFiles = append(nonUniqueFiles, comparisonDirectory[i])
		}
	}

	for i := 0; i < len(nonUniqueFiles); i++ {
		//The file has to be opened first
		file, fileError := os.Open("" + nonUniqueFiles[i].Name())

		if fileError != nil {
			fmt.Println(fileError)
			return
		}

		// The file descriptor (File*) has to be used to get metadata
		fileInfo, err := file.Stat()

		// The file can be closed
		file.Close()

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(fileInfo.Name())
		fmt.Println(fileInfo.ModTime())
	}

}

func missingFiles(sourceDirectory []fs.DirEntry, comparisonDirectory []fs.DirEntry) ([]string, []fs.DirEntry) {
	var outdatedFiles []string
	var nonUniqueFiles []fs.DirEntry

	for i := 0; i < len(comparisonDirectory); i++ {
		if !contains(sourceDirectory, comparisonDirectory[i]) {
			outdatedFiles = append(outdatedFiles, comparisonDirectory[i].Name())
		} else {
			nonUniqueFiles = append(nonUniqueFiles, comparisonDirectory[i])
		}
	}

	return outdatedFiles, nonUniqueFiles
}

func nonUniqueFileCompare(nonUniqueFiles []fs.DirEntry, locationOne string, locationTwo string) []string {
	var outdatedFiles []string

	for i := 0; i < len(nonUniqueFiles); i++ {

		fileOneModTime, fileOneError := getFileModTime(locationOne + nonUniqueFiles[i].Name())
		if fileOneError != nil {
			fmt.Println(fileOneError)
		}

		fileTwoModTime, fileTwoError := getFileModTime(locationTwo + nonUniqueFiles[i].Name())
		if fileTwoError != nil {
			fmt.Println(fileTwoError)
		}

		if fileOneModTime.After(fileTwoModTime) && !(deepCompare(locationOne+nonUniqueFiles[i].Name(), locationTwo+nonUniqueFiles[i].Name())) {
			outdatedFiles = append(outdatedFiles, nonUniqueFiles[i].Name())
		}

	}

	return outdatedFiles
}

func getFileModTime(filepath string) (time.Time, error) {
	//The file has to be opened first
	file, fileError := os.Open(filepath)

	if fileError != nil {
		fmt.Println(fileError)
		return time.Time{}, fileError
	}

	// The file descriptor (File*) has to be used to get metadata
	fileInfo, err := file.Stat()

	// The file can be closed
	file.Close()

	if err != nil {
		fmt.Println(err)
		return time.Time{}, err
	}

	return fileInfo.ModTime(), nil
}

func deepCompare(file1, file2 string) bool {
	// Check file size ...

	f1, err := os.Open(file1)
	if err != nil {
		log.Fatal(err)
	}
	defer f1.Close()

	f2, err := os.Open(file2)
	if err != nil {
		log.Fatal(err)
	}
	defer f2.Close()

	for {
		b1 := make([]byte, chunkSize)
		_, err1 := f1.Read(b1)

		b2 := make([]byte, chunkSize)
		_, err2 := f2.Read(b2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true
			} else if err1 == io.EOF || err2 == io.EOF {
				return false
			} else {
				log.Fatal(err1, err2)
			}
		}

		if !bytes.Equal(b1, b2) {
			return false
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

func RemoveIndex(s []fs.DirEntry, index int) []fs.DirEntry {
	ret := make([]fs.DirEntry, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
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
