package store

import "time"

type SecretFetchStore interface {
	GetEnvString(key, fallback string) string
	GetEnvDuration(key, fallback string) time.Duration
	GetEnvBool(key, fallback string) bool
}

type SecretManager interface {
	GetSecret(key string) (string, bool)
}
