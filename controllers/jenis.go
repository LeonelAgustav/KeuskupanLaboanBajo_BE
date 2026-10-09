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

type JenisController struct {
	db *gorm.DB
}

func NewJenisController(db *gorm.DB) *JenisController {
	return &JenisController{db: db}
}

func (c *JenisController) List(ctx echo.Context) error {
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

	db := c.db.Model(&models.Jenis{})

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

	var jeniss []models.Jenis
	if err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&jeniss).Error; err != nil {
		return err
	}

	responses := make([]dto.JenisResponse, len(jeniss))
	for i, j := range jeniss {
		responses[i] = dto.JenisResponse{
			ID:        j.ID,
			Nama:      j.Nama,
			CreatedAt: j.CreatedAt,
			UpdatedAt: j.UpdatedAt,
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

func (c *JenisController) Get(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var jenis models.Jenis
	if err := c.db.First(&jenis, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Jenis tidak ditemukan")
		}
		return err
	}

	return ctx.JSON(http.StatusOK, dto.JenisResponse{
		ID:        jenis.ID,
		Nama:      jenis.Nama,
		CreatedAt: jenis.CreatedAt,
		UpdatedAt: jenis.UpdatedAt,
	})
}

func (c *JenisController) Create(ctx echo.Context) error {
	var req dto.JenisCreateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	jenis := models.Jenis{
		Nama: req.Nama,
	}

	if err := c.db.Create(&jenis).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, dto.CreateResponse{
		Message: "Data Berhasil di Buat",
		Data: dto.JenisResponse{
			ID:        jenis.ID,
			Nama:      jenis.Nama,
			CreatedAt: jenis.CreatedAt,
			UpdatedAt: jenis.UpdatedAt,
		},
	})
}

func (c *JenisController) Update(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.JenisUpdateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	var jenis models.Jenis
	if err := c.db.First(&jenis, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Jenis tidak ditemukan")
		}
		return err
	}

	if req.Nama != nil {
		jenis.Nama = *req.Nama
	}

	if err := c.db.Save(&jenis).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Data Berhasil di Perbarui",
	})
}

func (c *JenisController) Delete(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	result := c.db.Delete(&models.Jenis{}, id)
	if result.Error != nil {
		if result.Error.Error() != "" && (result.Error.Error() == "foreign key constraint" || result.Error.Error() == "FOREIGN KEY constraint failed") {
			return echo.NewHTTPError(http.StatusBadRequest, "Tidak bisa hapus: masih ada akun terkait")
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Jenis tidak ditemukan")
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Data Berhasil di Hapus",
	})
}
