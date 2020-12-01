package secretmanager

import (
	"os"

	"github.com/joho/godotenv"
)

// NewOfflineClient returns an "offline" implementation that uses a dot-env format input
func NewOfflineClient(file, project string) (KVClient, error) {
	fileReader, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fileReader.Close()
	env, err := godotenv.Parse(fileReader)
	if err != nil {
		return nil, err
	}
	var flatList []string
	for k, v := range env {
		flatList = append(flatList, k, v)
	}
	return NewInMemoryClient(project, flatList...), nil
}
