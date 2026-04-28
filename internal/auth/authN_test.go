package auth

import (
	"testing"
)

func TestCheckAndCompareHashPassword(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		match bool
		err   error
	}{
		{
			name:  "Empty string",
			input: "",
			match: true,
			err:   nil,
		},
		{
			name:  "normal",
			input: "s!mp1ep@ssw0rd",
			match: true,
			err:   nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := HashPassword(tc.input)
			if err != nil {
				t.Fatalf("expected nil actual %v", err)
			}
			actual, err := CheckPasswordHash(tc.input, hash)
			if err != nil {
				t.Fatalf("expected nil actual %v", err)
			}
			if actual != tc.match {
				t.Fatalf("expected %v actual %v", tc.match, actual)
			}
		})
	}
}
