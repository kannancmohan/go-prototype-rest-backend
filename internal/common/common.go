package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	projectRoot     string
	projectRootOnce sync.Once
)

func GetProjectRoot() (string, error) {
	var err error

	projectRootOnce.Do(func() {
		dir, e := os.Getwd() // Get the current working directory
		if e != nil {
			err = e
			return
		}

		for { // Traverse up the directory tree until we find go.mod
			if _, e := os.Stat(filepath.Join(dir, "go.mod")); e == nil { // Check for the existence of go.mod
				projectRoot = dir
				return
			}

			parentDir := filepath.Dir(dir) // Move up one directory
			if parentDir == dir {          // Reached the root of the filesystem
				err = fmt.Errorf("project root not found")
				return
			}

			dir = parentDir
		}
	})

	return projectRoot, err
}

func GetMigrationSourcePath() string {
	path, exists := os.LookupEnv("MIGRATIONS_PATH")
	if exists {
		if strings.HasPrefix(path, "file://") {
			return path
		}
		return "file://" + path
	}
	rootDir, _ := GetProjectRoot()
	return "file://" + rootDir + "/cmd/migrate/migrations"
}
