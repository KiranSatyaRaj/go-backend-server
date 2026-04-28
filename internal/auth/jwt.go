package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJwt(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(tokenSecret))
	return ss, err
}

func ValidateJwt(tokenString, tokenSecret string) (uuid.UUID, error) {
	type MyClaims struct {
		jwt.RegisteredClaims
	}
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	claims, ok := token.Claims.(*MyClaims)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("claims not found")
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.UUID{}, err
	}
	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	bearer_token := headers.Get("Authorization")
	if len(bearer_token) == 0 {
		return "", fmt.Errorf("No Auth Token found")
	}
	bearer_token = strings.Split(bearer_token, " ")[1]
	return bearer_token, nil
}
