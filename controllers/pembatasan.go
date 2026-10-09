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

type PembatasanController struct {
	db *gorm.DB
}

func NewPembatasanController(db *gorm.DB) *PembatasanController {
	return &PembatasanController{db: db}
}

func (c *PembatasanController) List(ctx echo.Context) error {
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

	db := c.db.Model(&models.Pembatasan{}).Preload("Akun")

	if query.Search != "" {
		search := "%" + query.Search + "%"
		db = db.Where("tipe LIKE ?", search)
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
			"tipe": "tipe", "tipe asc": "tipe ASC", "tipe desc": "tipe DESC",
			"akun_id": "akun_id", "akun_id asc": "akun_id ASC", "akun_id desc": "akun_id DESC",
			"created_at": "created_at", "created_at asc": "created_at ASC", "created_at desc": "created_at DESC",
			"updated_at": "updated_at", "updated_at asc": "updated_at ASC", "updated_at desc": "updated_at DESC",
		}
		if val, ok := allowedSorts[strings.ToLower(query.Sort)]; ok {
			orderBy = val
		}
	}

	var pembatasans []models.Pembatasan
	if err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&pembatasans).Error; err != nil {
		return err
	}

	responses := make([]dto.PembatasanResponse, len(pembatasans))
	for i, p := range pembatasans {
		responses[i] = dto.PembatasanResponse{
			ID:        p.ID,
			Tipe:      p.Tipe,
			Nilai:     p.Nilai,
			AkunID:    p.AkunID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		}
		if p.Akun != nil && p.Akun.ID != 0 {
			responses[i].Akun = &dto.AkunResponse{
				ID:        p.Akun.ID,
				Kode:      p.Akun.Kode,
				Nama:      p.Akun.Nama,
				JenisID:   p.Akun.JenisID,
				CreatedAt: p.Akun.CreatedAt,
				UpdatedAt: p.Akun.UpdatedAt,
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

func (c *PembatasanController) Get(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var pembatasan models.Pembatasan
	if err := c.db.Preload("Akun").First(&pembatasan, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Pembatasan tidak ditemukan")
		}
		return err
	}

	resp := dto.PembatasanResponse{
		ID:        pembatasan.ID,
		Tipe:      pembatasan.Tipe,
		Nilai:     pembatasan.Nilai,
		AkunID:    pembatasan.AkunID,
		CreatedAt: pembatasan.CreatedAt,
		UpdatedAt: pembatasan.UpdatedAt,
	}
	if pembatasan.Akun != nil && pembatasan.Akun.ID != 0 {
		resp.Akun = &dto.AkunResponse{
			ID:        pembatasan.Akun.ID,
			Kode:      pembatasan.Akun.Kode,
			Nama:      pembatasan.Akun.Nama,
			JenisID:   pembatasan.Akun.JenisID,
			CreatedAt: pembatasan.Akun.CreatedAt,
			UpdatedAt: pembatasan.Akun.UpdatedAt,
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (c *PembatasanController) Create(ctx echo.Context) error {
	var req dto.PembatasanCreateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	pembatasan := models.Pembatasan{
		Tipe:   req.Tipe,
		Nilai:  req.Nilai,
		AkunID: req.AkunID,
	}

	if err := c.db.Create(&pembatasan).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, dto.CreateResponse{
		Message: "Data Berhasil di Buat",
		Data: dto.PembatasanResponse{
			ID:        pembatasan.ID,
			Tipe:      pembatasan.Tipe,
			Nilai:     pembatasan.Nilai,
			AkunID:    pembatasan.AkunID,
			CreatedAt: pembatasan.CreatedAt,
			UpdatedAt: pembatasan.UpdatedAt,
		},
	})
}

func (c *PembatasanController) Update(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.PembatasanUpdateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	var pembatasan models.Pembatasan
	if err := c.db.First(&pembatasan, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Pembatasan tidak ditemukan")
		}
		return err
	}

	if req.Tipe != nil {
		pembatasan.Tipe = *req.Tipe
	}
	if req.Nilai != nil {
		pembatasan.Nilai = req.Nilai
	}
	if req.AkunID != nil {
		pembatasan.AkunID = req.AkunID
	}

	if err := c.db.Save(&pembatasan).Error; err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Data Berhasil di Perbarui",
	})
}

func (c *PembatasanController) Delete(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	result := c.db.Delete(&models.Pembatasan{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Pembatasan tidak ditemukan")
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Data Berhasil di Hapus",
	})
}


