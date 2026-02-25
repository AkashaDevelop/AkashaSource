package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"net/url"

	"golang.org/x/crypto/bcrypt"
)

// Turnstile Verification
type TurnstileResponse struct {
	Success bool `json:"success"`
}

func VerifyTurnstile(token string) bool {
	if !TurnstileCheckEnabled {
		return true
	}
	if TurnstileSecretKey == "" {
		return true // Skip check if not configured
	}

	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
		"secret":   {TurnstileSecretKey},
		"response": {token},
	})
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result TurnstileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	return result.Success
}

// Password hashing
func Password2Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func ValidatePassword(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Token generation
func GenerateKey() string {
	return fmt.Sprintf("sk-%s", GetUUID())
}

func GetUUID() string {
	// Simple UUID generation for demo purposes.
	// In production, consider using "github.com/google/uuid"
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// HMAC-SHA256 signature
func HmacSha256(data string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
