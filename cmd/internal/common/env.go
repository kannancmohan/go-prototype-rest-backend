package app_common

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func GetEnvNameFromCommandLine() string {
	var env string
	flag.StringVar(&env, "env", "", "Environment Name")
	flag.Parse()
	return env
}

type ErrorEnvVarNotFound struct {
	Key string
}

func (e *ErrorEnvVarNotFound) Error() string {
	return fmt.Sprintf("environment variable %q not found", e.Key)
}

type SecretManager interface {
	GetSecret(key string) (string, error)
}

// EnvFetcher fetches environment variables considering `_SECURE` suffix.
type envVarFetcher struct {
	secretManager SecretManager
}

func NewEnvVarFetcher(envFile string, secretManager SecretManager) *envVarFetcher {
	err := godotenv.Load(envFile) // loaf envvar from file (eg .env)
	if err != nil {
		slog.Info("No envvar file found..")
	}
	return &envVarFetcher{secretManager: secretManager}
}

func (e *envVarFetcher) GetEnv(key string) (string, error) {
	// If the key ends with `_SECURE` and a secret manager is provided, fetch from it.
	if strings.HasSuffix(key, "_SECURE") && e.secretManager != nil {
		return e.secretManager.GetSecret(key)
	}
	// Fetch from environment variables.
	value, exists := os.LookupEnv(key)
	if !exists {
		return "", &ErrorEnvVarNotFound{Key: key}
	}
	return value, nil
}

func (e *envVarFetcher) GetEnvString(key, fallback string) string {
	v, err := e.GetEnv(key)
	if err == nil {
		return v
	}
	return fallback
}

func (e *envVarFetcher) GetEnvDuration(key, fallback string) time.Duration {
	strValue := e.GetEnvString(key, fallback)
	duration, err := time.ParseDuration(strValue)
	if err != nil {
		log.Fatalf("failed to parse environment variable[key:%q, value:%q]: %v", key, strValue, err) //TODO check other option to handle error
	}
	return duration
}

// func getString(key, fallback string) string {
// 	v, ok := os.LookupEnv(key)
// 	if !ok {
// 		return fallback
// 	}
// 	return v
// }

// func getInt(key string, fallback int) int {
// 	v, ok := os.LookupEnv(key)
// 	if !ok {
// 		return fallback
// 	}
// 	intVal, err := strconv.Atoi(v)
// 	if err != nil {
// 		return fallback
// 	}
// 	return intVal
// }

// func getBool(key string, fallback bool) bool {
// 	v, ok := os.LookupEnv(key)
// 	if !ok {
// 		return fallback
// 	}
// 	boolVal, err := strconv.ParseBool(v)
// 	if err != nil {
// 		return fallback
// 	}
// 	return boolVal
// }
