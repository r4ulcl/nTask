package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateToken Generate oauth
func GenerateToken(length int) (string, error) {
	if length%2 != 0 {
		return "", fmt.Errorf("token length must be even")
	}

	bytes := make([]byte, length/2)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
