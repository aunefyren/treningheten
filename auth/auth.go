package auth

import (
	"errors"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaim struct {
	UserID uuid.UUID `json:"id"`
	Admin  bool      `json:"admin"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID uuid.UUID) (tokenString string, err error) {
	expirationTime := time.Now().Add(time.Hour * 24 * 7)
	claims := &JWTClaim{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "Treningheten",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtKey := files.GetPrivateKey(1)
	tokenString, err = token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return
}

func GenerateJWTFromClaims(claims *JWTClaim) (tokenString string, err error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtKey := files.GetPrivateKey(1)
	tokenString, err = token.SignedString(jwtKey)
	return
}

func ValidateToken(signedToken string, admin bool) (err error) {
	jwtKey := files.GetPrivateKey(1)
	token, err := jwt.ParseWithClaims(
		signedToken,
		&JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtKey), nil
		},
	)
	if err != nil {
		return
	}
	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		err = errors.New("Couldn't parse claims.")
		return
	} else if claims.ExpiresAt == nil || claims.NotBefore == nil {
		err = errors.New("Claims not present.")
		return
	}
	now := time.Now()
	if claims.ExpiresAt.Time.Before(now) {
		err = errors.New("Token has expired.")
		return
	}
	if claims.NotBefore.Time.After(now) {
		err = errors.New("Token has not begun.")
		return
	}

	if admin {

		userObject, userErr := database.GetUserInformation(claims.UserID)
		if userErr != nil {
			err = errors.New("Failed to check admin status.")
			return
		} else if *userObject.Admin != true {
			err = errors.New("Token is not an admin session.")
			return
		}
	}

	return
}

func ParseToken(signedToken string) (*JWTClaim, error) {
	jwtKey := files.GetPrivateKey(1)
	token, err := jwt.ParseWithClaims(
		signedToken,
		&JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtKey), nil
		},
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		err = errors.New("Couldn't parse claims")
		return nil, err
	} else if claims.ExpiresAt == nil || claims.NotBefore == nil {
		err = errors.New("Claims not present.")
		return nil, err
	}
	now := time.Now()
	if claims.ExpiresAt.Time.Before(now) {
		err = errors.New("Token has expired.")
		return nil, err
	}
	if claims.NotBefore.Time.After(now) {
		err = errors.New("Token has not begun.")
		return nil, err
	}
	return claims, nil

}
