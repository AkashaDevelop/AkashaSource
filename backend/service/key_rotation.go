package service

import (
	"math/rand"
	"strings"
)

// GetNextKey selects a random key from a newline-separated list of keys
func GetNextKey(keys string) string {
	parts := strings.Split(keys, "\n")
	var validKeys []string
	for _, k := range parts {
		k = strings.TrimSpace(k)
		if k != "" {
			validKeys = append(validKeys, k)
		}
	}
	if len(validKeys) == 0 {
		return keys // fallback to original
	}
	return validKeys[rand.Intn(len(validKeys))]
}
