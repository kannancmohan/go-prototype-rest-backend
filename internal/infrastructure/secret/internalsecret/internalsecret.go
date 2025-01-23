package internalsecret

import (
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
)

type secretFetchStore struct {
	extSecretManager store.SecretManager
}

func NewSecretFetchStore(envFileName string) *secretFetchStore {
	loadEnvVar(envFileName)
	return &secretFetchStore{}
}

func (s *secretFetchStore) GetEnvString(key, fallback string) string {
	v, exits := s.getEnv(key)
	if exits == true {
		return v
	}
	return fallback
}

func (s *secretFetchStore) GetEnvDuration(key, fallback string) time.Duration {
	strValue := s.GetEnvString(key, fallback)
	duration, err := time.ParseDuration(strValue)
	if err != nil {
		log.Fatalf("failed to parse environment variable[key:%q, value:%q]: %v", key, strValue, err) //TODO check other option to handle error
	}
	return duration
}

func (s *secretFetchStore) getEnv(key string) (string, bool) {
	// If the key ends with `_SECURE` and external SecretManager is provided, fetch from it.
	if strings.HasSuffix(key, "_SECURE") && s.extSecretManager != nil {
		return s.extSecretManager.GetSecret(key)
	}
	// Fetch from environment variables.
	return os.LookupEnv(key)
}

func loadEnvVar(envFileName string) {
	if err := godotenv.Load(envFileName); err != nil { // load envvar from file (eg .env)
		slog.Info("No envvar file found..")
	}
}
