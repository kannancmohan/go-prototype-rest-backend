package app_common

import (
	"flag"
)

func GetEnvNameFromCommandLine() string {
	var env string
	flag.StringVar(&env, "env", "", "Environment Name")
	flag.Parse()
	return env
}
