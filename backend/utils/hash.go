package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

func HashPassword(password string) (string, error) {
	salt, err := GenerateSalt(16)
	if err != nil {
		return "", err
	}

	timeCost := uint32(1)
	memoryCost := uint32(64 * 1024)
	parallelism := uint8(1)
	hashLength := uint32(32)

	hash := argon2.IDKey([]byte(password), []byte(salt), timeCost, memoryCost, parallelism, hashLength)

	encodedHash := base64.StdEncoding.EncodeToString(hash)

	hashString := fmt.Sprintf("%s:%s", salt, encodedHash)

	return hashString, nil
}

func GenerateSalt(length int) (string, error) {
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %v", err)
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}
