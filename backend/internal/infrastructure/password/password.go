// Package password enforces bcrypt's byte limit before hashing or comparison.
package password

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"unicode/utf8"
)

func Validate(value string) error {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 8 || len(value) > 72 {
		return errors.New("密码至少 8 个字符，且最多 72 字节")
	}
	return nil
}
func Hash(value string) (string, error) {
	if err := Validate(value); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(value), bcrypt.DefaultCost)
	return string(hash), err
}
func Matches(hash, value string) bool {
	if len(value) > 72 {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(value)) == nil
}
