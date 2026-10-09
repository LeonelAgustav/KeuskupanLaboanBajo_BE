package controllers

import (
	"net/http"
	"strconv"

	"KeuskupanLaboanBajo_BE/middleware"
	"KeuskupanLaboanBajo_BE/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct {
	db *gorm.DB
}

func NewAuthController(db *gorm.DB) *AuthController {
	return &AuthController{db: db}
}

type RegisterRequest struct {
	Nama     string `json:"nama" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=100"`
	RoleID   *uint  `json:"role_id,omitempty"`
	ParokiID *uint  `json:"paroki_id,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func (a *AuthController) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	// Public registration is always MEMBER: never trust req.RoleID (no privilege escalation)
	var memberRole models.Role
	var roleID *uint
	if err := a.db.Where("nama = ?", "MEMBER").First(&memberRole).Error; err == nil {
		roleID = &memberRole.ID
	}
	if req.ParokiID == nil || *req.ParokiID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Registrasi wajib memilih paroki")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Gagal memproses password")
	}

	user := models.User{
		Nama:         req.Nama,
		Email:        req.Email,
		PasswordHash: string(hash),
		RoleID:       roleID,
		ParokiID:     req.ParokiID,
	}
	if err := a.db.Create(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusConflict, "Email sudah terdaftar")
	}
	
	// Load role name if RoleID is set
	roleName := ""
	if user.RoleID != nil {
		var role models.Role
		if err := a.db.First(&role, *user.RoleID).Error; err == nil {
			roleName = role.Nama
		}
	}
	
	var parokiID uint
	if user.ParokiID != nil {
		parokiID = *user.ParokiID
	}
	access, refresh, _ := middleware.GenerateToken(user.ID, user.Email, roleName, user.TokenVersion, parokiID)
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Registrasi berhasil",
		"data": map[string]interface{}{
			"id": user.ID, "email": user.Email, "role": roleName, "paroki_id": parokiID, "access_token": access, "refresh_token": refresh,
		},
	})
}

func (a *AuthController) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	var user models.User
	if err := a.db.Preload("Role").Where("email = ? OR nama = ?", req.Email, req.Email).First(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Email atau password salah")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Email atau password salah")
	}
	
	roleName := ""
	if user.Role != nil {
		roleName = user.Role.Nama
	}
	
	var parokiID uint
	if user.ParokiID != nil {
		parokiID = *user.ParokiID
	}
	access, refresh, _ := middleware.GenerateToken(user.ID, user.Email, roleName, user.TokenVersion, parokiID)
	return c.JSON(http.StatusOK, map[string]string{
		"access_token": access, "refresh_token": refresh,
	})
}

func (a *AuthController) Refresh(c echo.Context) error {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := c.Bind(&body); err != nil {
		return err
	}
	if err := c.Validate(&body); err != nil {
		return err
	}
	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(body.RefreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return middleware.JWTSecret(), nil
	})
	if err != nil || !token.Valid {
		return echo.NewHTTPError(http.StatusUnauthorized, "Refresh token tidak valid")
	}
	var uid uint64
	uid, err = strconv.ParseUint(claims.UserID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Refresh token tidak valid")
	}
	// Get role from claims
	roleName := claims.Role
	var parokiID uint
	if claims.ParokiID != "" {
		if pid, err := strconv.ParseUint(claims.ParokiID, 10, 64); err == nil {
			parokiID = uint(pid)
		}
	}
	// ponytail: kill-switch validation on refresh
	var user models.User
	if err := a.db.First(&user, uid).Error; err != nil || user.TokenVersion != claims.TokenVersion {
		return echo.NewHTTPError(http.StatusUnauthorized, "Sesi telah dicabut, silakan login ulang")
	}
	access, refresh, _ := middleware.GenerateToken(user.ID, user.Email, roleName, user.TokenVersion, parokiID)
	return c.JSON(http.StatusOK, map[string]string{"access_token": access, "refresh_token": refresh})
}
