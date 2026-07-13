//go:build !cgo

package auth

import "errors"

func HashPassword(string) (string, error) { return "", errors.New("Argon2id requer CGO e libargon2") }
func VerifyPassword(string, string) (bool, error) {
	return false, errors.New("Argon2id requer CGO e libargon2")
}
