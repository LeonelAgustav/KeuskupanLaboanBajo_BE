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

type ParokiController struct {
	db *gorm.DB
}

func NewParokiController(db *gorm.DB) *ParokiController {
	return &ParokiController{db: db}
}

func (c *ParokiController) List(ctx echo.Context) error {
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

	db := c.db.Model(&models.Paroki{}).Preload("Keuskupan")

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
			"keuskupan_id": "keuskupan_id", "keuskupan_id asc": "keuskupan_id ASC", "keuskupan_id desc": "keuskupan_id DESC",
			"created_at": "created_at", "created_at asc": "created_at ASC", "created_at desc": "created_at DESC",
			"updated_at": "updated_at", "updated_at asc": "updated_at ASC", "updated_at desc": "updated_at DESC",
		}
		if val, ok := allowedSorts[strings.ToLower(query.Sort)]; ok {
			orderBy = val
		}
	}

	var parokis []models.Paroki
	if err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&parokis).Error; err != nil {
		return err
	}

	responses := make([]dto.ParokiResponse, len(parokis))
	for i, p := range parokis {
		responses[i] = dto.ParokiResponse{
			ID:          p.ID,
			Nama:        p.Nama,
			Alamat:      p.Alamat,
			KeuskupanID: p.KeuskupanID,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		}
		if p.Keuskupan.ID != 0 {
			responses[i].Keuskupan = &dto.KeuskupanResponse{
				ID:        p.Keuskupan.ID,
				Nama:      p.Keuskupan.Nama,
				Alamat:    p.Keuskupan.Alamat,
				CreatedAt: p.Keuskupan.CreatedAt,
				UpdatedAt: p.Keuskupan.UpdatedAt,
			}
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

func (c *ParokiController) Get(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var paroki models.Paroki
	if err := c.db.Preload("Keuskupan").First(&paroki, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Paroki tidak ditemukan")
		}
		return err
	}

	resp := dto.ParokiResponse{
		ID:          paroki.ID,
		Nama:        paroki.Nama,
		Alamat:      paroki.Alamat,
		KeuskupanID: paroki.KeuskupanID,
		CreatedAt:   paroki.CreatedAt,
		UpdatedAt:   paroki.UpdatedAt,
	}
	if paroki.Keuskupan.ID != 0 {
		resp.Keuskupan = &dto.KeuskupanResponse{
			ID:        paroki.Keuskupan.ID,
			Nama:      paroki.Keuskupan.Nama,
			Alamat:    paroki.Keuskupan.Alamat,
			CreatedAt: paroki.Keuskupan.CreatedAt,
			UpdatedAt: paroki.Keuskupan.UpdatedAt,
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (c *ParokiController) Create(ctx echo.Context) error {
	var req dto.ParokiCreateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	paroki := models.Paroki{
		Nama:        req.Nama,
		Alamat:      req.Alamat,
		KeuskupanID: req.KeuskupanID,
	}

	if err := c.db.Create(&paroki).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, dto.CreateResponse{
		Message: "Data Berhasil di Buat",
		Data: dto.ParokiResponse{
			ID:          paroki.ID,
			Nama:        paroki.Nama,
			Alamat:      paroki.Alamat,
			KeuskupanID: paroki.KeuskupanID,
			CreatedAt:   paroki.CreatedAt,
			UpdatedAt:   paroki.UpdatedAt,
		},
	})
}

func (c *ParokiController) Update(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.ParokiUpdateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	var paroki models.Paroki
	if err := c.db.First(&paroki, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Paroki tidak ditemukan")
		}
		return err
	}

	if req.Nama != nil {
		paroki.Nama = *req.Nama
	}
	if req.Alamat != nil {
		paroki.Alamat = *req.Alamat
	}
	if req.KeuskupanID != nil {
		paroki.KeuskupanID = *req.KeuskupanID
	}

	if err := c.db.Save(&paroki).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, dto.CreateResponse{
		Message: "Data Berhasil di Perbarui",
		Data: dto.ParokiResponse{
			ID:          paroki.ID,
			Nama:        paroki.Nama,
			Alamat:      paroki.Alamat,
			KeuskupanID: paroki.KeuskupanID,
			CreatedAt:   paroki.CreatedAt,
			UpdatedAt:   paroki.UpdatedAt,
		},
	})
}

func (c *ParokiController) Delete(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	result := c.db.Delete(&models.Paroki{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Paroki tidak ditemukan")
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Data Berhasil di Hapus",
	})
}


