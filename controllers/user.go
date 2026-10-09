package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"KeuskupanLaboanBajo_BE/dto"
	"KeuskupanLaboanBajo_BE/models"
)

var validate = validator.New()

type UserController struct {
	db *gorm.DB
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{db: db}
}

func (c *UserController) List(ctx echo.Context) error {
	var query dto.ListQuery
	if err := ctx.Bind(&query); err != nil {
		return err
	}
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 {
		query.Limit = 10
	}
	if query.Limit > 100 {
		query.Limit = 100
	}

	db := c.db.Model(&models.User{}).Preload("Role").Preload("Paroki")

	if query.Search != "" {
		search := "%" + query.Search + "%"
		db = db.Where("nama LIKE ? OR email LIKE ?", search, search)
	}
	// ADMIN tidak boleh lihat SUPERADMIN
	if role, _ := ctx.Get("role").(string); role == "ADMIN" {
		var superRole models.Role
		if err := c.db.Where("nama = ?", "SUPERADMIN").First(&superRole).Error; err == nil {
			db = db.Where("role_id != ? OR role_id IS NULL", superRole.ID)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return err
	}

	offset := (query.Page - 1) * query.Limit
	orderBy := "id DESC"
	if query.Sort != "" {
		s := query.Sort
		allowedSorts := map[string]string{
			"id": "id", "id asc": "id ASC", "id desc": "id DESC",
			"nama": "nama", "nama asc": "nama ASC", "nama desc": "nama DESC",
			"email": "email", "email asc": "email ASC", "email desc": "email DESC",
			"created_at": "created_at", "created_at asc": "created_at ASC", "created_at desc": "created_at DESC",
			"updated_at": "updated_at", "updated_at asc": "updated_at ASC", "updated_at desc": "updated_at DESC",
		}
		if val, ok := allowedSorts[strings.ToLower(s)]; ok {
			orderBy = val
		}
	}

	var user []models.User
	if err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&user).Error; err != nil {
		return err
	}

	responses := make([]dto.UserResponse, len(user))
	for i, u := range user {
		responses[i] = dto.UserResponse{
			ID:        u.ID,
			Nama:      u.Nama,
			Email:     u.Email,
			Role:      toRoleResponse(u.Role),
			ParokiID:  u.ParokiID,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		}
	}

	return ctx.JSON(http.StatusOK, dto.ListResponse{
		Data:       responses,
		Page:       query.Page,
		Limit:      query.Limit,
		Total:      total,
		TotalPages: int((total + int64(query.Limit) - 1) / int64(query.Limit)),
	})
}

func (c *UserController) Get(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var user models.User
	if err := c.db.Preload("Role").Preload("Paroki").First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "User tidak ditemukan")
		}
		return err
	}

	return ctx.JSON(http.StatusOK, dto.UserResponse{
		ID:        user.ID,
		Nama:      user.Nama,
		Email:     user.Email,
		Role:      toRoleResponse(user.Role),
		ParokiID:  user.ParokiID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

func (c *UserController) Create(ctx echo.Context) error {
	var req dto.UserCreateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := validate.Struct(&req); err != nil {
		return err
	}

	// ADMIN tidak boleh buat SUPERADMIN
	if req.RoleID != nil {
		var r models.Role
		if err := c.db.First(&r, *req.RoleID).Error; err == nil && r.Nama == "SUPERADMIN" {
			if role, _ := ctx.Get("role").(string); role == "ADMIN" {
				return echo.NewHTTPError(http.StatusForbidden, "ADMIN tidak boleh membuat user SUPERADMIN")
			}
		}
	}
	user := models.User{
		Nama:     req.Nama,
		Email:    req.Email,
		RoleID:   req.RoleID,
		ParokiID: req.ParokiID,
	}

	if err := c.db.Create(&user).Error; err != nil {
		return err
	}

	// Load role if exists
	var role *models.Role
	if user.RoleID != nil {
		var roleModel models.Role
		if err := c.db.First(&roleModel, *user.RoleID).Error; err == nil {
			role = &roleModel
		}
	}

	return ctx.JSON(http.StatusCreated, dto.CreateResponse{
		Message: "Data Berhasil di Buat",
		Data: dto.UserResponse{
			ID:        user.ID,
			Nama:      user.Nama,
			Email:     user.Email,
			Role:      toRoleResponse(role),
			ParokiID:  user.ParokiID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	})
}

func (c *UserController) Update(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UserUpdateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := validate.Struct(&req); err != nil {
		return err
	}

	var user models.User
	if err := c.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "User tidak ditemukan")
		}
		return err
	}

	// Validate MEMBER must have paroki_id
	targetRoleID := user.RoleID
	if req.RoleID != nil {
		targetRoleID = req.RoleID
	}
	if targetRoleID != nil {
		var role models.Role
		if err := c.db.First(&role, *targetRoleID).Error; err == nil && role.Nama == "MEMBER" {
			targetParokiID := user.ParokiID
			if req.ParokiID != nil {
				targetParokiID = req.ParokiID
			}
			if targetParokiID == nil || *targetParokiID == 0 {
				return echo.NewHTTPError(http.StatusBadRequest, "MEMBER wajib memilih paroki")
			}
		}
	}

	if req.Nama != nil {
		user.Nama = *req.Nama
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Password != nil && *req.Password != "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		user.PasswordHash = string(hash)
		user.TokenVersion++ // invalidate old JWTs on password change
	}
	if req.RoleID != nil {
		user.RoleID = req.RoleID
		user.TokenVersion++ // invalidate old JWTs on role change
	}
	if req.ParokiID != nil {
		user.ParokiID = req.ParokiID
	}

	if err := c.db.Save(&user).Error; err != nil {
		return err
	}

	// Load role if exists
	var role *models.Role
	if user.RoleID != nil {
		var roleModel models.Role
		if err := c.db.First(&roleModel, *user.RoleID).Error; err == nil {
			role = &roleModel
		}
	}

	return ctx.JSON(http.StatusOK, dto.CreateResponse{
		Message: "Data Berhasil di Perbarui",
		Data: dto.UserResponse{
			ID:        user.ID,
			Nama:      user.Nama,
			Email:     user.Email,
			Role:      toRoleResponse(role),
			ParokiID:  user.ParokiID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	})
}

func (c *UserController) Delete(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	result := c.db.Delete(&models.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "User tidak ditemukan")
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Data Berhasil di Hapus",
	})
}

func toRoleResponse(role *models.Role) *dto.RoleResponse {
	if role == nil {
		return nil
	}
	return &dto.RoleResponse{
		ID:        role.ID,
		Nama:      role.Nama,
		CreatedAt: role.CreatedAt,
		UpdatedAt: role.UpdatedAt,
	}
}
