package main

import (
	"os"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/joho/godotenv"
)

func prepareOfflineClient(file string) secretmanager.KVClient {
	fileReader, err := os.Open(file)
	panicIfErr(err)
	defer fileReader.Close()
	env, err := godotenv.Parse(fileReader)
	panicIfErr(err)
	var flatList []string
	for k, v := range env {
		flatList = append(flatList, k, v)
	}
	return secretmanager.NewInMemoryClient(file, flatList...)
}
