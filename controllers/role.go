package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"KeuskupanLaboanBajo_BE/dto"
	"KeuskupanLaboanBajo_BE/models"
)

type RoleController struct {
	db *gorm.DB
}

func NewRoleController(db *gorm.DB) *RoleController {
	return &RoleController{db: db}
}

func (c *RoleController) List(ctx echo.Context) error {
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

	db := c.db.Model(&models.Role{})

	if query.Search != "" {
		search := "%" + query.Search + "%"
		db = db.Where("nama LIKE ?", search)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return err
	}

	offset := (query.Page - 1) * query.Limit
	orderBy := "id DESC"
	if query.Sort != "" {
		allowedSorts := map[string]string{
			"id": "id", "id asc": "id ASC", "id desc": "id DESC",
			"nama": "nama", "nama asc": "nama ASC", "nama desc": "nama DESC",
			"created_at": "created_at", "created_at asc": "created_at ASC", "created_at desc": "created_at DESC",
			"updated_at": "updated_at", "updated_at asc": "updated_at ASC", "updated_at desc": "updated_at DESC",
		}
		if val, ok := allowedSorts[strings.ToLower(query.Sort)]; ok {
			orderBy = val
		}
	}

	var roles []models.Role
	if err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&roles).Error; err != nil {
		return err
	}

	responses := make([]dto.RoleResponse, len(roles))
	for i, r := range roles {
		responses[i] = dto.RoleResponse{
			ID:        r.ID,
			Nama:      r.Nama,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
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

func (c *RoleController) Get(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var role models.Role
	if err := c.db.First(&role, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Role tidak ditemukan")
		}
		return err
	}

	return ctx.JSON(http.StatusOK, dto.RoleResponse{
		ID:        role.ID,
		Nama:      role.Nama,
		CreatedAt: role.CreatedAt,
		UpdatedAt: role.UpdatedAt,
	})
}

func (c *RoleController) Create(ctx echo.Context) error {
	var req struct {
		Nama string `json:"nama" validate:"required,min=2,max=100"`
	}
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := validate.Struct(&req); err != nil {
		return err
	}

	role := models.Role{Nama: req.Nama}
	if err := c.db.Create(&role).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, dto.CreateResponse{
		Message: "Role berhasil dibuat",
		Data: dto.RoleResponse{
			ID:        role.ID,
			Nama:      role.Nama,
			CreatedAt: role.CreatedAt,
			UpdatedAt: role.UpdatedAt,
		},
	})
}

func (c *RoleController) Update(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var req struct {
		Nama *string `json:"nama,omitempty" validate:"omitempty,min=2,max=100"`
	}
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := validate.Struct(&req); err != nil {
		return err
	}

	var role models.Role
	if err := c.db.First(&role, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Role tidak ditemukan")
		}
		return err
	}

	if req.Nama != nil {
		role.Nama = *req.Nama
	}

	if err := c.db.Save(&role).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, dto.CreateResponse{
		Message: "Role berhasil diperbarui",
		Data: dto.RoleResponse{
			ID:        role.ID,
			Nama:      role.Nama,
			CreatedAt: role.CreatedAt,
			UpdatedAt: role.UpdatedAt,
		},
	})
}

func (c *RoleController) Delete(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	result := c.db.Delete(&models.Role{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Role tidak ditemukan")
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Role berhasil dihapus",
	})
}