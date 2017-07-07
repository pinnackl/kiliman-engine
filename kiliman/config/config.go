package config

import (
	"os"
)

var CNF = map[string]string{
	"IP_HOST": "163.172.182.154",
}

func init() {
	for key := range CNF {
		if val := os.Getenv(key); val != "" {
			CNF[key] = val
		}
	}
}
