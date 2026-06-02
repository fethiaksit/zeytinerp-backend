package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Exp      int64  `json:"exp"`
}

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}

func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func GenerateJWT(secret string, claims JWTClaims) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("JWT_SECRET is required")
	}

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	unsignedToken := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := signJWT(unsignedToken, secret)
	return unsignedToken + "." + signature, nil
}

func VerifyJWT(secret, token string) (JWTClaims, error) {
	if strings.TrimSpace(secret) == "" {
		return JWTClaims{}, errors.New("JWT_SECRET is required")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return JWTClaims{}, errors.New("invalid token")
	}

	unsignedToken := parts[0] + "." + parts[1]
	expectedSignature := signJWT(unsignedToken, secret)
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return JWTClaims{}, errors.New("invalid token")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return JWTClaims{}, errors.New("invalid token")
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return JWTClaims{}, errors.New("invalid token")
	}
	if claims.UserID == 0 || claims.Exp == 0 {
		return JWTClaims{}, errors.New("invalid token")
	}
	if time.Now().Unix() > claims.Exp {
		return JWTClaims{}, errors.New("token expired")
	}

	return claims, nil
}

func NewAuthClaims(userID uint, username, role string) JWTClaims {
	return JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		Exp:      time.Now().Add(24 * time.Hour).Unix(),
	}
}

func signJWT(unsignedToken, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsignedToken))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func UserIDFromContextValue(value interface{}) (uint, error) {
	switch v := value.(type) {
	case uint:
		return v, nil
	case int:
		if v <= 0 {
			return 0, errors.New("invalid user id")
		}
		return uint(v), nil
	case int64:
		if v <= 0 {
			return 0, errors.New("invalid user id")
		}
		return uint(v), nil
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil || parsed == 0 {
			return 0, errors.New("invalid user id")
		}
		return uint(parsed), nil
	default:
		return 0, fmt.Errorf("invalid user id")
	}
}
