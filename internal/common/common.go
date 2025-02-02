package common

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)

func GetRootDir() (string, error) {
	//TODO replace this logic
	_, currentFile, _, ok := runtime.Caller(1) // 1 for the calling function
	if !ok {
		return "", fmt.Errorf("unable to determine caller info")
	}
	rootDir := path.Join(path.Dir(currentFile), "../..")

	return rootDir, nil
}

func GetMigrationSourcePath() string {
	path, exists := os.LookupEnv("MIGRATIONS_PATH")
	if exists {
		if strings.HasPrefix(path, "file://") {
			return path
		}
		return "file://" + path
	}
	rootDir, _ := GetRootDir()
	return "file://" + rootDir + "/cmd/migrate/migrations"
}
