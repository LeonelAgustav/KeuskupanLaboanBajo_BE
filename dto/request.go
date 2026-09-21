package dto

import (
	"strings"
	"time"
)

type LocalDate struct {
	time.Time
}

func (ld *LocalDate) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		return nil
	}
	formats := []string{
		"2006-01-02",
		"02-01-2006",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			ld.Time = t
			return nil
		}
	}
	return nil
}

func (ld LocalDate) MarshalJSON() ([]byte, error) {
	if ld.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + ld.Format("2006-01-02") + `"`), nil
}

type AkunCreateRequest struct {
	Kode    string `json:"kode" validate:"required,min=3,max=20"`
	Nama    string `json:"nama" validate:"required,min=2,max=100"`
	JenisID uint     `json:"jenis_id" validate:"required"`
}

type AkunUpdateRequest struct {
	Kode    *string `json:"kode,omitempty" validate:"omitempty,min=3,max=20"`
	Nama    *string `json:"nama,omitempty" validate:"omitempty,min=2,max=100"`
	JenisID *uint     `json:"jenis_id,omitempty"`
}

type DetilJurnalCreateReq struct {
	AkunID     uint       `json:"akun_id" validate:"required"`
	ParokiID   uint       `json:"paroki_id" validate:"required"`
	Debit      *float64   `json:"debit,omitempty"`
	Kredit     *float64   `json:"kredit,omitempty"`
	Keterangan *string    `json:"keterangan,omitempty"`
}

type JenisCreateRequest struct {
	Nama string `json:"nama" validate:"required,min=2,max=50"`
}

type JenisUpdateRequest struct {
	Nama *string `json:"nama,omitempty" validate:"omitempty,min=2,max=50"`
}

type JurnalCreateRequest struct {
	KeuskupanID uint                     `json:"keuskupan_id" validate:"required"`
	Tanggal     LocalDate                `json:"tanggal" validate:"required"`
	Deskripsi   *string                  `json:"deskripsi,omitempty"`
	NoBukti     *string                  `json:"no_bukti,omitempty"`
	DetilJurnal []DetilJurnalCreateReq   `json:"detil_jurnal" validate:"required,min=1,dive"`
}

type JurnalUpdateRequest struct {
	KeuskupanID *uint                   `json:"keuskupan_id,omitempty"`
	Tanggal     *LocalDate              `json:"tanggal,omitempty"`
	Deskripsi   *string                 `json:"deskripsi,omitempty"`
	NoBukti     *string                 `json:"no_bukti,omitempty"`
	DetilJurnal []DetilJurnalCreateReq  `json:"detil_jurnal,omitempty" validate:"omitempty,dive"`
}

type KeuskupanCreateRequest struct {
	Nama   string `json:"nama" validate:"required,min=2,max=100"`
	Alamat string `json:"alamat" validate:"required,max=255"`
}

type KeuskupanUpdateRequest struct {
	Nama   *string `json:"nama,omitempty" validate:"omitempty,min=2,max=100"`
	Alamat *string `json:"alamat,omitempty" validate:"omitempty,max=255"`
}

type ParokiCreateRequest struct {
	Nama        string `json:"nama" validate:"required,min=2,max=100"`
	Alamat      string `json:"alamat" validate:"required,max=255"`
	KeuskupanID uint     `json:"keuskupan_id" validate:"required"`
}

type ParokiUpdateRequest struct {
	Nama        *string `json:"nama,omitempty" validate:"omitempty,min=2,max=100"`
	Alamat      *string `json:"alamat,omitempty" validate:"omitempty,max=255"`
	KeuskupanID *uint     `json:"keuskupan_id,omitempty"`
}

type PembatasanCreateRequest struct {
	Tipe   string   `json:"tipe" validate:"required,min=2,max=50"`
	Nilai  *float64 `json:"nilai,omitempty"`
	AkunID *uint      `json:"akun_id,omitempty"`
}

type PembatasanUpdateRequest struct {
	Tipe   *string  `json:"tipe,omitempty" validate:"omitempty,min=2,max=50"`
	Nilai  *float64 `json:"nilai,omitempty"`
	AkunID *uint      `json:"akun_id,omitempty"`
}

type UserCreateRequest struct {
	Nama     string `json:"nama" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email,max=100"`
	Password string `json:"password" validate:"required,min=6,max=100"`
	RoleID   *uint  `json:"role_id,omitempty"`
	ParokiID *uint  `json:"paroki_id,omitempty"`
}

type UserUpdateRequest struct {
	Nama     *string `json:"nama,omitempty" validate:"omitempty,min=2,max=100"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email,max=100"`
	Password *string `json:"password,omitempty" validate:"omitempty,min=6,max=100"`
	RoleID   *uint   `json:"role_id,omitempty"`
	ParokiID *uint   `json:"paroki_id,omitempty"`
}

type ListQuery struct {
	Page   int    `query:"page" validate:"omitempty,min=1"`
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Sort   string `query:"sort"`
	Search string `query:"search"`
}
