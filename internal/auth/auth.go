package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/KirillZiborov/go-loyalty-program/internal/logging"
	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int `json:"user_id"`
}

const TokenExp = time.Hour * 3
const SecretKey = "supersecretkey"

func GenerateToken(userID int) (string, error) {
	if userID == 0 {
		logging.Sugar.Warnw("userID is 0 before token generation")
	}

	tokenString, err := BuildJWTString(userID)
	if err != nil {
		logging.Sugar.Fatalw("Error while generating token", err)
	}
	// log.Println("Generated token:", tokenString)

	return tokenString, nil
}

func BuildJWTString(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},

		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GetUserID(tokenString string) (int, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	logging.Sugar.Infow("Token is valid")
	return claims.UserID, nil
}

func AuthPost(w http.ResponseWriter, r *http.Request, userID int) error {
	token, err := GenerateToken(userID)
	if err != nil {
		http.Error(w, "Error while generating token", http.StatusInternalServerError)
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "cookie",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(TokenExp),
		HttpOnly: true,
	})

	return nil
}

func AuthGet(r *http.Request) (int, error) {
	cookie, err := r.Cookie("cookie")
	if err != nil {
		// log.Println("Cookie not found", err)
		return 0, err
	}

	userID, err := GetUserID(cookie.Value)
	if err != nil {
		// log.Println("Error extracting userID", err)
		return 0, err
	}

	logging.Sugar.Infof("Auth userID: %d", userID)
	return userID, err
}
