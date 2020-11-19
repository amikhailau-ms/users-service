package auth

import (
	"github.com/dgrijalva/jwt-go"
)

type GameClaims struct {
	UserId    string `json:"user_id,omitempty"`
	UserName  string `json:"username,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
	IsAdmin   bool   `json:"is_admin,omitempty"`
	jwt.StandardClaims
}

func (c GameClaims) Valid() error {
	return c.StandardClaims.Valid()
}
