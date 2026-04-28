package auth

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateJWT(t *testing.T) {
	testcases := []struct {
		name        string
		tokenSecret string
		usedId      uuid.UUID
		expiresIn   time.Duration
		err         error
	}{
		{
			name:        "Valid Signature",
			tokenSecret: "t0ken$ec12et",
			usedId:      uuid.New(),
			expiresIn:   10 * time.Minute,
			err:         nil,
		}, {
			name:        "Expired Signature",
			tokenSecret: "p@$$w0rd",
			usedId:      uuid.New(),
			expiresIn:   1 * time.Nanosecond,
			err:         errors.New("token has invalid claims: token is expired"),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			sign, err := MakeJwt(tc.usedId, tc.tokenSecret, tc.expiresIn)
			if err != nil {
				t.Fatalf("expected nil got %v", err)
			}
			actual_id, err := ValidateJwt(sign, tc.tokenSecret)
			if tc.err != nil && err == nil {
				t.Fatalf(("expected %v got %v"), tc.err, err)
			}

			if tc.err == nil && err != nil {
				t.Fatalf("expeceted %v got %v", tc.err, err)
			}

			if tc.err != nil && err != nil && tc.err.Error() != err.Error() {
				t.Fatalf("expected %v got %v", tc.err, err)
			}

			if tc.usedId.String() != actual_id.String() && tc.err == nil && err == nil {
				t.Fatalf("expected %v got %v", tc.usedId.String(), actual_id.String())
			}
		})
	}
}

func TestGetBearerToken(t *testing.T) {
	testcases := []struct {
		name  string
		token string
	}{
		{
			name:  "Valid Token",
			token: "290ndiownview",
		},
		{
			name:  "No Auth Token",
			token: "",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			header := make(http.Header)
			header.Add("Authorization", fmt.Sprintf("Bearer %v", tc.token))
			token, _ := GetBearerToken(header)
			if token != tc.token {
				t.Fatalf("want %v got %v", tc.token, token)
			}
		})
	}
}
