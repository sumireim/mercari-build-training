package main

import (
	"mercari-build-training/app"
	"os"
	"path/filepath"
)

const (
	port = "9000"
	//imageDirPath = "images"
)

func main() {
	// Get the absolute path to the images directory
	currentDir, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	imgDirPath := filepath.Join(currentDir, "images")
	//追加ここまで

	// This is the entry point of the application.
	// You don't need to modify this function.
	os.Exit(app.Server{
		Port:         port,
		ImageDirPath: imgDirPath,
	}.Run())
}
