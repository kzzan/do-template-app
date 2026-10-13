// Package jwt 提供 HS256 签发与解析。
package jwt

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

type JWT struct {
	key []byte
}

type MyCustomClaims struct {
	UserID string
	jwt.RegisteredClaims
}

func NewJWT(conf *viper.Viper) *JWT {
	return &JWT{key: []byte(conf.GetString("security.jwt.key"))}
}

// GenToken 签发指定过期时间的 token。
func (j *JWT) GenToken(userId string, expiresAt time.Time) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, MyCustomClaims{
		UserID: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	})
	return token.SignedString(j.key)
}

// ParseToken 解析 token，自动去除 Bearer 前缀。
func (j *JWT) ParseToken(tokenString string) (*MyCustomClaims, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("token is empty")
	}
	token, err := jwt.ParseWithClaims(tokenString, &MyCustomClaims{}, func(*jwt.Token) (any, error) {
		return j.key, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*MyCustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
