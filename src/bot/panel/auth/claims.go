package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	ManagerId uint `json:"mid"`
	FractionId uint `json:"fid"`
	SquadId *uint `json:"sid,omitempty"`
	AccountType *string `json:"role,omitempty"`
	jwt.RegisteredClaims
}
