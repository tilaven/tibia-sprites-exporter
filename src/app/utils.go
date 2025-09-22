package app

import "os"

func isEnvExist(key string) bool {
	data, ok := os.LookupEnv(key)
	return ok && data != ""
}
