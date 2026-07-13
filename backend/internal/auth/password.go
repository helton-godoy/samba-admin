//go:build cgo

package auth

/*
#cgo linux LDFLAGS: -l:libargon2.so.1
#cgo freebsd CFLAGS: -I/usr/local/include
#cgo freebsd LDFLAGS: -L/usr/local/lib -largon2
#include <stdint.h>
#include <stddef.h>
int argon2id_hash_raw(uint32_t t_cost, uint32_t m_cost, uint32_t parallelism,
                     const void *pwd, size_t pwdlen, const void *salt, size_t saltlen,
                     void *hash, size_t hashlen);
*/
import "C"

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

const (
	memory      = 64 * 1024
	iterations  = 3
	parallelism = 2
	saltLength  = 16
	keyLength   = 32
)

func derive(password string, salt []byte, mem, it uint32, par uint32, length int) ([]byte, error) {
	out := make([]byte, length)
	pwd := []byte(password)
	var pp, sp, hp unsafe.Pointer
	if len(pwd) > 0 {
		pp = unsafe.Pointer(&pwd[0])
	}
	if len(salt) > 0 {
		sp = unsafe.Pointer(&salt[0])
	}
	if len(out) > 0 {
		hp = unsafe.Pointer(&out[0])
	}
	rc := C.argon2id_hash_raw(C.uint32_t(it), C.uint32_t(mem), C.uint32_t(par), pp, C.size_t(len(pwd)), sp, C.size_t(len(salt)), hp, C.size_t(len(out)))
	if rc != 0 {
		return nil, fmt.Errorf("argon2id falhou: código %d", int(rc))
	}
	return out, nil
}
func HashPassword(password string) (string, error) {
	if len(password) < 14 {
		return "", errors.New("a senha inicial deve possuir pelo menos 14 caracteres")
	}
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash, err := derive(password, salt, memory, iterations, parallelism, keyLength)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, iterations, parallelism, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}
func VerifyPassword(encoded, password string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false, errors.New("hash Argon2id inválido")
	}
	var mem, it, par uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &it, &par); err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	actual, err := derive(password, salt, mem, it, par, len(expected))
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
