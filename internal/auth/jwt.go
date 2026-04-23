package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	AccessToken  TokenType = "access-token"
	RefreshToken TokenType = "refresh-token"
)

func MakeToken(issue, secret string, userID string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    issue,
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID,
	})

	return token.SignedString([]byte(secret))
}

func GetBearerToken(authorizationHeader string) (string, error) {
	token := strings.TrimPrefix(authorizationHeader, "Bearer ")

	if token == authorizationHeader {
		return "", errors.New("must be Bearer")
	}

	return token, nil
}

func ValidateJWT(tokenSecret, tokenString string) (uuid.UUID, error) {
	claimsStruct := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(t *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		},
	)
	if err != nil {
		return uuid.Nil, err
	}

	userIDString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	issue, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.Nil, err
	}

	if issue != string(RefreshToken) && issue != string(AccessToken) {
		return uuid.Nil, errors.New("invalid token issuer")
	}

	id, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}
