package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	apiKey := headers.Get("Authorization")
	if len(apiKey) == 0 {
		return "", fmt.Errorf("No API Key found")
	}
	authValues := strings.Split(apiKey, " ")
	if len(authValues) < 2 {
		return "", fmt.Errorf("No API Key found")
	}
	return authValues[1], nil
}
