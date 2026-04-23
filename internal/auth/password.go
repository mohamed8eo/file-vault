package auth

import (
	"errors"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password are require")
	}

	hashPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}

	return hashPassword, nil
}

func CheckHash(hashPassword, password string) (bool, error) {
	if hashPassword == "" || password == "" {
		return false, errors.New("hashPassword & password are required")
	}

	match, err := argon2id.ComparePasswordAndHash(password, hashPassword)
	if err != nil {
		return false, err
	}

	if !match {
		return false, errors.New("passwords are not match")
	}

	return true, nil
}
