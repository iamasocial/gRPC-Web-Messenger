package utils

import (
	"errors"
	"messenger/backend/entities"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var jwtSecret = []byte("secure-key")

type Claims struct {
	UserId uint64 `json:"used_id"`
	jwt.StandardClaims
}

func GenerateToken(userID uint64) (*entities.Token, error) {
	expirationTime := time.Now().Add(1 * time.Hour)

	claims := &Claims{
		UserId: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			Issuer:    "messenger",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	return &entities.Token{
		UserID:    userID,
		Token:     tokenString,
		ExpiresAt: expirationTime,
	}, nil
}

func ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
