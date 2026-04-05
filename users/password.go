package users

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", errors.New("password should not be empty")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(passwordHash), nil
}

func checkPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
