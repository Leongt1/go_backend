package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTManager struct {
	secretKey string // for sigining of jwt token
	issuer    string // who issued the token
}

func NewJWTManager(secretKey string, issuer string) *JWTManager {
	return &JWTManager{
		secretKey: secretKey,
		issuer:    issuer,
	}
}

type UserClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (jm *JWTManager) GenerateToken(userID uuid.UUID, role string, ttl time.Duration) (string, error) {
	// getting current time
	now := time.Now().UTC()

	// creating claims (payload)
	claims := &UserClaims{
		UserID: userID.String(),
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jm.issuer,
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	// creating new token with claims and a signing method
	// created HEADER + PAYLOAD
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// signing the token with secret key and returning it
	// created HEADER + PAYLOAD + SIGNATURE
	return token.SignedString([]byte(jm.secretKey))
}

func (jm *JWTManager) VerifyToken(tokenString string) (*UserClaims, error) {
	// parsing the token with claims and a signing method
	token, err := jwt.ParseWithClaims(
		tokenString,
		&UserClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jm.secretKey), nil
		},
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, err
	}

	return claims, nil
}
