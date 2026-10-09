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

type AkunController struct {
	db *gorm.DB
}

func NewAkunController(db *gorm.DB) *AkunController {
	return &AkunController{db: db}
}

func (c *AkunController) List(ctx echo.Context) error {
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

	db := c.db.Model(&models.Akun{}).Preload("Jenis")

	if query.Search != "" {
		search := "%" + query.Search + "%"
		db = db.Where("kode LIKE ? OR nama LIKE ?", search, search)
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
			"kode": "kode", "kode asc": "kode ASC", "kode desc": "kode DESC",
			"nama": "nama", "nama asc": "nama ASC", "nama desc": "nama DESC",
			"jenis_id": "jenis_id", "jenis_id asc": "jenis_id ASC", "jenis_id desc": "jenis_id DESC",
			"created_at": "created_at", "created_at asc": "created_at ASC", "created_at desc": "created_at DESC",
			"updated_at": "updated_at", "updated_at asc": "updated_at ASC", "updated_at desc": "updated_at DESC",
		}
		if val, ok := allowedSorts[strings.ToLower(s)]; ok {
			orderBy = val
		}
	}

	var akuns []models.Akun
	if err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&akuns).Error; err != nil {
		return err
	}

	responses := make([]dto.AkunResponse, len(akuns))
	for i, a := range akuns {
		responses[i] = dto.AkunResponse{
			ID:        a.ID,
			Kode:      a.Kode,
			Nama:      a.Nama,
			JenisID:   a.JenisID,
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
		}
		if a.Jenis.ID != 0 {
			responses[i].Jenis = &dto.JenisResponse{
				ID:        a.Jenis.ID,
				Nama:      a.Jenis.Nama,
				CreatedAt: a.Jenis.CreatedAt,
				UpdatedAt: a.Jenis.UpdatedAt,
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

func (c *AkunController) Get(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var akun models.Akun
	if err := c.db.Preload("Jenis").First(&akun, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Akun tidak ditemukan")
		}
		return err
	}

	resp := dto.AkunResponse{
		ID:        akun.ID,
		Kode:      akun.Kode,
		Nama:      akun.Nama,
		JenisID:   akun.JenisID,
		CreatedAt: akun.CreatedAt,
		UpdatedAt: akun.UpdatedAt,
	}
	if akun.Jenis.ID != 0 {
		resp.Jenis = &dto.JenisResponse{
			ID:        akun.Jenis.ID,
			Nama:      akun.Jenis.Nama,
			CreatedAt: akun.Jenis.CreatedAt,
			UpdatedAt: akun.Jenis.UpdatedAt,
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (c *AkunController) Create(ctx echo.Context) error {
	var req dto.AkunCreateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	akun := models.Akun{
		Kode:    req.Kode,
		Nama:    req.Nama,
		JenisID: req.JenisID,
	}

	if err := c.db.Create(&akun).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, dto.CreateResponse{
		Message: "Data Berhasil di Buat",
		Data: dto.AkunResponse{
			ID:        akun.ID,
			Kode:      akun.Kode,
			Nama:      akun.Nama,
			JenisID:   akun.JenisID,
			CreatedAt: akun.CreatedAt,
			UpdatedAt: akun.UpdatedAt,
		},
	})
}

func (c *AkunController) Update(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.AkunUpdateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	var akun models.Akun
	if err := c.db.First(&akun, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Akun tidak ditemukan")
		}
		return err
	}

	if req.Kode != nil {
		akun.Kode = *req.Kode
	}
	if req.Nama != nil {
		akun.Nama = *req.Nama
	}
	if req.JenisID != nil {
		akun.JenisID = *req.JenisID
	}

	if err := c.db.Save(&akun).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, dto.CreateResponse{
		Message: "Data Berhasil di Perbarui",
		Data: dto.AkunResponse{
			ID:        akun.ID,
			Kode:      akun.Kode,
			Nama:      akun.Nama,
			JenisID:   akun.JenisID,
			CreatedAt: akun.CreatedAt,
			UpdatedAt: akun.UpdatedAt,
		},
	})
}

func (c *AkunController) Delete(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	result := c.db.Delete(&models.Akun{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Akun tidak ditemukan")
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Data Berhasil di Hapus",
	})
}


