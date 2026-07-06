package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
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
		return true
	}

	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
		"secret":   {TurnstileSecretKey},
		"response": {token},
	})
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result TurnstileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	return result.Success
}

// GeeTest (极验) Verification
type GeeTestValidateRequest struct {
	LotNumber     string `json:"lot_number"`
	CaptchaOutput string `json:"captcha_output"`
	PassToken     string `json:"pass_token"`
	GenTime       string `json:"gen_time"`
}

type GeeTestResponse struct {
	Result string `json:"result"`
	Reason string `json:"reason"`
}

func VerifyGeeTest(req *GeeTestValidateRequest) bool {
	if !GeeTestEnabled || GeeTestId == "" || GeeTestKey == "" {
		return true
	}
	if req == nil || req.LotNumber == "" {
		return false
	}

	// Generate sign_token = hmac_sha256(lot_number, geetest_key)
	signToken := HmacSha256(req.LotNumber, GeeTestKey)

	formData := url.Values{
		"lot_number":     {req.LotNumber},
		"captcha_output": {req.CaptchaOutput},
		"pass_token":     {req.PassToken},
		"gen_time":       {req.GenTime},
		"sign_token":     {signToken},
		"captcha_id":     {GeeTestId},
	}

	resp, err := http.PostForm("https://gcaptcha4.geetest.com/validate", formData)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result GeeTestResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	return result.Result == "success"
}

// hCaptcha Verification
type HCaptchaResponse struct {
	Success bool `json:"success"`
}

func VerifyHCaptcha(token string) bool {
	if !HCaptchaEnabled {
		return true
	}
	if HCaptchaSecretKey == "" {
		return true
	}

	resp, err := http.PostForm("https://api.hcaptcha.com/siteverify", url.Values{
		"secret":   {HCaptchaSecretKey},
		"response": {token},
	})
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result HCaptchaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	return result.Success
}

// Google reCAPTCHA Verification
type ReCaptchaResponse struct {
	Success bool    `json:"success"`
	Score   float64 `json:"score"`
	Action  string  `json:"action"`
}

func VerifyReCaptcha(token string) bool {
	if !ReCaptchaEnabled {
		return true
	}
	if ReCaptchaSecretKey == "" {
		return true
	}

	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   {ReCaptchaSecretKey},
		"response": {token},
	})
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result ReCaptchaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	if !result.Success {
		return false
	}
	// v3: check score threshold (0.5)
	if ReCaptchaVersion == "v3" {
		if result.Score < 0.5 {
			return false
		}
	}
	return true
}

// VerifyCaptcha is a unified captcha verification function
// It checks the configured provider (turnstile, geetest, hcaptcha, or recaptcha)
func VerifyCaptcha(turnstileToken, hcaptchaToken, recaptchaToken string, geetest *GeeTestValidateRequest) bool {
	switch CaptchaProvider {
	case "turnstile":
		if !TurnstileCheckEnabled {
			return true
		}
		return VerifyTurnstile(turnstileToken)
	case "geetest":
		if !GeeTestEnabled {
			return true
		}
		return VerifyGeeTest(geetest)
	case "hcaptcha":
		if !HCaptchaEnabled {
			return true
		}
		return VerifyHCaptcha(hcaptchaToken)
	case "recaptcha":
		if !ReCaptchaEnabled {
			return true
		}
		return VerifyReCaptcha(recaptchaToken)
	default:
		// Fallback: check turnstile if enabled
		if TurnstileCheckEnabled {
			return VerifyTurnstile(turnstileToken)
		}
		return true
	}
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
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

// HMAC-SHA256 signature
func HmacSha256(data string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// MapToJSON converts a map to JSON string
func MapToJSON(m map[string]float64) (string, error) {
	b, err := json.Marshal(m)
	return string(b), err
}

// GetTimestamp 返回当前 Unix 时间戳
func GetTimestamp() int64 {
	return time.Now().Unix()
}
