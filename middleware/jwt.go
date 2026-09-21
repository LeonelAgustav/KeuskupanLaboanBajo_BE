package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func JWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "keuskupan-dev-secret-ganti-di-prod"
	}
	return []byte(secret)
}

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Role     string `json:"role,omitempty"`
	ParokiID string `json:"paroki_id,omitempty"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, email, role string, parokiID ...uint) (access, refresh string, err error) {
	uid := strconv.FormatUint(uint64(userID), 10)
	pid := ""
	if len(parokiID) > 0 && parokiID[0] != 0 {
		pid = strconv.FormatUint(uint64(parokiID[0]), 10)
	}
	// ponytail: 1 tahun biar tidak sering login di lapangan
	oneYear := 365 * 24 * time.Hour
	accessClaims := Claims{
		UserID:   uid,
		Email:    email,
		Role:     role,
		ParokiID: pid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(oneYear)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "keuskupan-be",
		},
	}
	refreshClaims := Claims{
		UserID:   uid,
		Email:    email,
		Role:     role,
		ParokiID: pid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(oneYear)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "keuskupan-be",
		},
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	access, err = at.SignedString(JWTSecret())
	if err != nil {
		return
	}
	refresh, err = rt.SignedString(JWTSecret())
	return
}

func JWTMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// ponytail: preflight OPTIONS jangan check JWT, biar CORS lancar
		if c.Request().Method == "OPTIONS" {
			return next(c)
		}
		auth := c.Request().Header.Get("Authorization")
		if auth == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Token tidak ada")
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Format Authorization harus Bearer <token>")
		}
		tokenStr := parts[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return JWTSecret(), nil
		})
		if err != nil || !token.Valid {
			return echo.NewHTTPError(http.StatusUnauthorized, "Token tidak valid: "+err.Error())
		}
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("paroki_id", claims.ParokiID)
		return next(c)
	}
}
