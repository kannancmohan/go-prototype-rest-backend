package env

import (
	"os"
	"strconv"
)

func GetString(key, fallback string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return v
}

func GetInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	intVal, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return intVal
}

func GetBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	boolVal, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return boolVal
}
