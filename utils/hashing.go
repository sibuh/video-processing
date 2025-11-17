package utils

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const salt = 10

var ErrHashingFailed = errors.New("hashing failed")

func HashPassword(pass string) (string, error) {
	byt, err := bcrypt.GenerateFromPassword([]byte(pass), salt)
	if err != nil {
		return "", errors.Join(err, ErrHashingFailed)
	}
	return string(byt), nil
}
func CheckPassword(hash, pass string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)); err != nil {
		return false
	}
	return true
}
