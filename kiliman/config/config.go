package config

import (
	"os"
)

var CNF = map[string]string{
	"IP_HOST": "172.16.21.69",
}

func init() {
	for key := range CNF {
		if val := os.Getenv(key); val != "" {
			CNF[key] = val
		}
	}
}
