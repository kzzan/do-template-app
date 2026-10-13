package jwt

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestGenAndParseToken(t *testing.T) {
	j := NewJWT(viper.New())

	tests := []struct {
		name    string
		userId  string
		wantErr bool
	}{
		{name: "roundtrip", userId: "user-123"},
		{name: "empty user id", userId: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := j.GenToken(tt.userId, time.Now().Add(time.Hour))
			if err != nil {
				t.Fatalf("GenToken: %v", err)
			}

			claims, err := j.ParseToken(token)
			if err != nil {
				t.Fatalf("ParseToken: %v", err)
			}
			if claims.UserID != tt.userId {
				t.Errorf("UserID = %q, want %q", claims.UserID, tt.userId)
			}
		})
	}
}

func TestParseTokenInvalid(t *testing.T) {
	j := NewJWT(viper.New())

	tests := []struct {
		name  string
		token string
	}{
		{name: "empty", token: ""},
		{name: "bearer only", token: "Bearer "},
		{name: "garbage", token: "not-a-jwt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := j.ParseToken(tt.token); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseTokenExpired(t *testing.T) {
	j := NewJWT(viper.New())
	token, err := j.GenToken("u1", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("GenToken: %v", err)
	}
	if _, err := j.ParseToken(token); err == nil {
		t.Error("expired token should fail")
	}
}
