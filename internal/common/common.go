package common

import (
	"fmt"
	"path"
	"runtime"
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
