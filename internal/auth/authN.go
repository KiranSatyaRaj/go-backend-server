package auth

import (
	a2id "github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	hash, err := a2id.CreateHash(password, a2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := a2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return match, nil
}
