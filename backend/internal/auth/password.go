package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Memory      uint32 = 64 * 1024
	argon2Iterations  uint32 = 3
	argon2Parallelism uint8  = 4
	argon2SaltLength         = 16
	argon2KeyLength   uint32 = 32
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	return encodePassword(password, salt, argon2Memory, argon2Iterations, argon2Parallelism, argon2KeyLength), nil
}

func VerifyPassword(encoded, password string) (bool, error) {
	memory, iterations, parallelism, salt, expected, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}

	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func dummyPasswordHash() string {
	return encodePassword("switchonyourcode-invalid-password", make([]byte, argon2SaltLength), argon2Memory, argon2Iterations, argon2Parallelism, argon2KeyLength)
}

func encodePassword(password string, salt []byte, memory, iterations uint32, parallelism uint8, keyLength uint32) string {
	key := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

func parsePasswordHash(encoded string) (uint32, uint32, uint8, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return 0, 0, 0, nil, nil, errors.New("invalid Argon2id password hash format")
	}

	parameters := strings.Split(parts[3], ",")
	if len(parameters) != 3 {
		return 0, 0, 0, nil, nil, errors.New("invalid Argon2id parameters")
	}

	memory, err := parseUint32Parameter(parameters[0], "m=")
	if err != nil {
		return 0, 0, 0, nil, nil, err
	}
	iterations, err := parseUint32Parameter(parameters[1], "t=")
	if err != nil {
		return 0, 0, 0, nil, nil, err
	}
	parallelism64, err := parseUint32Parameter(parameters[2], "p=")
	if err != nil || parallelism64 == 0 || parallelism64 > 255 {
		return 0, 0, 0, nil, nil, errors.New("invalid Argon2id parallelism")
	}
	parallelism := uint8(parallelism64)

	if memory < 8*uint32(parallelism) || memory > 1024*1024 || iterations == 0 || iterations > 10 {
		return 0, 0, 0, nil, nil, errors.New("unsafe Argon2id parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return 0, 0, 0, nil, nil, errors.New("invalid Argon2id salt")
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) < 16 || len(expected) > 64 {
		return 0, 0, 0, nil, nil, errors.New("invalid Argon2id output")
	}

	return memory, iterations, parallelism, salt, expected, nil
}

func parseUint32Parameter(value, prefix string) (uint32, error) {
	if !strings.HasPrefix(value, prefix) {
		return 0, errors.New("invalid Argon2id parameter")
	}
	parsed, err := strconv.ParseUint(strings.TrimPrefix(value, prefix), 10, 32)
	if err != nil {
		return 0, errors.New("invalid Argon2id parameter")
	}
	return uint32(parsed), nil
}
