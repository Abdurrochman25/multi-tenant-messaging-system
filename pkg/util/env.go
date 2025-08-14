package util

import (
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

var once sync.Once

const defaultConfigFile = ".env"

func LoadEnv(file string) {
	once.Do(func() {
		if file == "" {
			file = defaultConfigFile
		}
		if err := godotenv.Load(file); err != nil {
			log.Fatalf("Error loading %s file", file)
		}
	})
}

func GetEnv(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return defaultVal
}

func GetEnvAsInt(key string, defaultVal int) int {
	strVal := GetEnv(key, "")

	if val, err := strconv.Atoi(strVal); err == nil {
		return val
	}

	return defaultVal
}
