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
	GetSecret(key string) (string, bool)
}

// EnvFetcher fetches environment variables considering `_SECURE` suffix.
type envVarFetcher struct {
	secretManager SecretManager
}

func NewEnvVarFetcher(envFile string, secretManager SecretManager) *envVarFetcher {
	err := godotenv.Load(envFile) // load envvar from file (eg .env)
	if err != nil {
		slog.Info("No envvar file found..")
	}
	return &envVarFetcher{secretManager: secretManager}
}

func (e *envVarFetcher) GetEnv(key string) (string, bool) {
	// If the key ends with `_SECURE` and a secret manager is provided, fetch from it.
	if strings.HasSuffix(key, "_SECURE") && e.secretManager != nil {
		return e.secretManager.GetSecret(key)
	}
	// Fetch from environment variables.
	return os.LookupEnv(key)
}

func (e *envVarFetcher) GetEnvString(key, fallback string) string {
	v, exits := e.GetEnv(key)
	if exits == true {
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
