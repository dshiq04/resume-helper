// Package security holds the password primitives shared by the HTTP server and
// the standalone password recovery command.
package security

import (
	"crypto/rand"
	"math/big"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinPasswordRunes = 4
	MaxPasswordRunes = 64
	// RecoveryPasswordLength is long enough to be typed comfortably while
	// remaining far outside any brute-force budget.
	RecoveryPasswordLength = 16
)

// passwordAlphabet avoids visually ambiguous characters (0/O, 1/l/I) so the
// generated recovery password can be copied from the console reliably.
const passwordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

// HashPassword hashes a password with the same bcrypt cost used by the normal
// password endpoints.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword compares a plaintext password with a bcrypt hash.
func VerifyPassword(hash, password string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GeneratePassword returns a cryptographically random password from the
// unambiguous alphabet.
func GeneratePassword(length int) (string, error) {
	if length <= 0 {
		length = RecoveryPasswordLength
	}
	result := make([]byte, length)
	max := big.NewInt(int64(len(passwordAlphabet)))
	for index := range result {
		value, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[index] = passwordAlphabet[value.Int64()]
	}
	return string(result), nil
}

// ValidatePassword applies the same rules as the password setup endpoint.
func ValidatePassword(password string) (string, string) {
	if password == "" {
		return "password_required", "Password is required."
	}
	runes := utf8.RuneCountInString(password)
	if runes < MinPasswordRunes {
		return "password_too_short", "Password must be at least 4 characters."
	}
	if runes > MaxPasswordRunes || len(password) > 72 {
		return "password_too_long", "Password is too long."
	}
	return "", ""
}
