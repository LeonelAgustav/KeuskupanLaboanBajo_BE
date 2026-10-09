package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"KeuskupanLaboanBajo_BE/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

var DB *gorm.DB

func JWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("❌ JWT_SECRET environment variable is not set")
	}
	return []byte(secret)
}

type Claims struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role,omitempty"`
	ParokiID     string `json:"paroki_id,omitempty"`
	TokenVersion uint   `json:"token_version"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, email, role string, tokenVersion uint, parokiID ...uint) (access, refresh string, err error) {
	uid := strconv.FormatUint(uint64(userID), 10)
	pid := ""
	if len(parokiID) > 0 && parokiID[0] != 0 {
		pid = strconv.FormatUint(uint64(parokiID[0]), 10)
	}
	
	// ponytail: 2y token for offline-first, kill via token_version bump in DB
	accessClaims := Claims{
		UserID:       uid,
		Email:        email,
		Role:         role,
		ParokiID:     pid,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * 365 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "keuskupan-be",
		},
	}
	refreshClaims := Claims{
		UserID:       uid,
		Email:        email,
		Role:         role,
		ParokiID:     pid,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * 365 * 24 * time.Hour)),
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
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return JWTSecret(), nil
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Token tidak valid")
		}
		if token == nil || !token.Valid {
			return echo.NewHTTPError(http.StatusUnauthorized, "Token tidak valid")
		}
		// ponytail: kill-switch, tolak token versi lama setelah admin bump token_version
		if DB != nil {
			var u models.User
			if err := DB.First(&u, claims.UserID).Error; err != nil || u.TokenVersion != claims.TokenVersion {
				return echo.NewHTTPError(http.StatusUnauthorized, "Sesi tidak valid, silakan login ulang")
			}
		}
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("paroki_id", claims.ParokiID)
		return next(c)
	}
}

func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole, _ := c.Get("role").(string)
			for _, r := range roles {
				if userRole == r {
					return next(c)
				}
			}
			return echo.NewHTTPError(http.StatusForbidden, "Anda tidak memiliki akses ke fitur ini")
		}
	}
}
