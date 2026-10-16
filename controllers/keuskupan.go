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

type KeuskupanController struct {
	db *gorm.DB
}

func NewKeuskupanController(db *gorm.DB) *KeuskupanController {
	return &KeuskupanController{db: db}
}

func (c *KeuskupanController) List(ctx echo.Context) error {
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

	db := c.db.Model(&models.Keuskupan{})

	if query.Search != "" {
		search := "%" + query.Search + "%"
		db = db.Where("nama LIKE ? OR alamat LIKE ?", search, search)
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

	var keuskupans []models.Keuskupan
	if err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&keuskupans).Error; err != nil {
		return err
	}

	responses := make([]dto.KeuskupanResponse, len(keuskupans))
	for i, k := range keuskupans {
	responses[i] = dto.KeuskupanResponse{
			ID:        k.ID,
			UUID:      k.UUID,
			Nama:      k.Nama,
			Alamat:    k.Alamat,
			CreatedAt: k.CreatedAt,
			UpdatedAt: k.UpdatedAt,
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

func (c *KeuskupanController) Get(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var keuskupan models.Keuskupan
	if err := c.db.First(&keuskupan, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Keuskupan tidak ditemukan")
		}
		return err
	}

	return ctx.JSON(http.StatusOK, dto.KeuskupanResponse{
		ID:        keuskupan.ID,
		UUID:      keuskupan.UUID,
		Nama:      keuskupan.Nama,
		Alamat:    keuskupan.Alamat,
		CreatedAt: keuskupan.CreatedAt,
		UpdatedAt: keuskupan.UpdatedAt,
	})
}

func (c *KeuskupanController) Create(ctx echo.Context) error {
	var req dto.KeuskupanCreateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	keuskupan := models.Keuskupan{
		Nama:   req.Nama,
		Alamat: req.Alamat,
	}

	if req.UUID != "" {
		var existing models.Keuskupan
		if err := c.db.Where("uuid = ?", req.UUID).First(&existing).Error; err == nil {
			return ctx.JSON(http.StatusOK, dto.CreateResponse{
				Message: "Data Sudah Ada",
				Data: dto.KeuskupanResponse{
					ID:        existing.ID,
					Nama:      existing.Nama,
					Alamat:    existing.Alamat,
					CreatedAt: existing.CreatedAt,
					UpdatedAt: existing.UpdatedAt,
				},
			})
		}
		keuskupan.UUID = req.UUID
	}

	if err := c.db.Create(&keuskupan).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, dto.CreateResponse{
		Message: "Data Berhasil di Buat",
		Data: dto.KeuskupanResponse{
			ID:        keuskupan.ID,
			UUID:      keuskupan.UUID,
			Nama:      keuskupan.Nama,
			Alamat:    keuskupan.Alamat,
			CreatedAt: keuskupan.CreatedAt,
			UpdatedAt: keuskupan.UpdatedAt,
		},
	})
}

func (c *KeuskupanController) Update(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.KeuskupanUpdateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	var keuskupan models.Keuskupan
	if err := c.db.First(&keuskupan, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Keuskupan tidak ditemukan")
		}
		return err
	}

	if req.Nama != nil {
		keuskupan.Nama = *req.Nama
	}
	if req.Alamat != nil {
		keuskupan.Alamat = *req.Alamat
	}

	if err := c.db.Save(&keuskupan).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, dto.CreateResponse{
		Message: "Data Berhasil di Perbarui",
		Data: dto.KeuskupanResponse{
			ID:        keuskupan.ID,
			UUID:      keuskupan.UUID,
			Nama:      keuskupan.Nama,
			Alamat:    keuskupan.Alamat,
			CreatedAt: keuskupan.CreatedAt,
			UpdatedAt: keuskupan.UpdatedAt,
		},
	})
}

func (c *KeuskupanController) Delete(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	result := c.db.Delete(&models.Keuskupan{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Keuskupan tidak ditemukan")
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Data Berhasil di Hapus",
	})
}
