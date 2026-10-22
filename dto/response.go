package dto

import (
	"time"
)

type AkunResponse struct {
	ID        uint           `json:"id"`
	UUID      string         `json:"uuid"`
	Kode      string         `json:"kode"`
	Nama      string         `json:"nama"`
	JenisID   uint           `json:"jenis_id"`
	Jenis     *JenisResponse `json:"jenis,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type BukuBesarMutasi struct {
	Tanggal string  `json:"tanggal"`
	NoBukti *string `json:"no_bukti,omitempty"`
	Uraian  *string `json:"uraian,omitempty"`
	Debit   float64 `json:"debit"`
	Kredit  float64 `json:"kredit"`
	Saldo   float64 `json:"saldo"`
}

type BukuBesarResponse struct {
	AkunID     uint              `json:"akun_id"`
	AkunKode   string            `json:"akun_kode"`
	AkunNama   string            `json:"akun_nama"`
	SaldoAwal  float64           `json:"saldo_awal"`
	Mutasi     []BukuBesarMutasi `json:"mutasi"`
	SaldoAkhir float64           `json:"saldo_akhir"`
}

type DetilJurnalResp struct {
	ID           uint            `json:"id"`
	JurnalID     string          `json:"jurnal_id"`
	ParokiID     uint            `json:"paroki_id"`
	Paroki       *ParokiResponse `json:"paroki,omitempty"`
	AkunDebitID  *uint           `json:"akun_debit_id,omitempty"`
	AkunDebit    *AkunResponse   `json:"akun_debit,omitempty"`
	AkunKreditID *uint           `json:"akun_kredit_id,omitempty"`
	AkunKredit   *AkunResponse   `json:"akun_kredit,omitempty"`
	Jumlah       *float64        `json:"jumlah,omitempty"`
	AkunID       uint            `json:"akun_id"` // shortcut: backward compat, drop setelah migrasi
	Akun         *AkunResponse   `json:"akun,omitempty"` // shortcut: backward compat
	Debit        *float64        `json:"debit,omitempty"` // shortcut: backward compat
	Kredit       *float64        `json:"kredit,omitempty"` // shortcut: backward compat
	Keterangan   *string         `json:"keterangan,omitempty"`
	CreatedBy    *uint           `json:"created_by,omitempty"`
	UpdatedBy    *uint           `json:"updated_by,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type JenisResponse struct {
	ID        uint      `json:"id"`
	UUID      string    `json:"uuid"`
	Nama      string    `json:"nama"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type JurnalResponse struct {
	ID          string             `json:"id"`
	KeuskupanID uint               `json:"keuskupan_id"`
	Keuskupan   *KeuskupanResponse `json:"keuskupan,omitempty"`
	Tanggal     time.Time          `json:"tanggal"`
	Deskripsi   *string            `json:"deskripsi,omitempty"`
	NoBukti     *string            `json:"no_bukti,omitempty"`
	DetilJurnal []DetilJurnalResp  `json:"detil_jurnal,omitempty"`
	CreatedBy   *uint              `json:"created_by,omitempty"`
	UpdatedBy   *uint              `json:"updated_by,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type KeuskupanResponse struct {
	ID        uint      `json:"id"`
	UUID      string    `json:"uuid"`
	Nama      string    `json:"nama"`
	Alamat    string    `json:"alamat"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ParokiResponse struct {
	ID          uint               `json:"id"`
	UUID        string             `json:"uuid"`
	Nama        string             `json:"nama"`
	Alamat      string             `json:"alamat"`
	KeuskupanID uint               `json:"keuskupan_id"`
	Keuskupan   *KeuskupanResponse `json:"keuskupan,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type PembatasanResponse struct {
	ID        uint          `json:"id"`
	Tipe      string        `json:"tipe"`
	Nilai     *float64      `json:"nilai,omitempty"`
	AkunID    *uint         `json:"akun_id,omitempty"`
	Akun      *AkunResponse `json:"akun,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type RoleResponse struct {
	ID        uint      `json:"id"`
	UUID      string    `json:"uuid"`
	Nama      string    `json:"nama"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserResponse struct {
	ID        uint            `json:"id"`
	UUID      string          `json:"uuid"`
	Nama      string          `json:"nama"`
	Email     string          `json:"email"`
	Role      *RoleResponse   `json:"role,omitempty"`
	ParokiID  *uint           `json:"paroki_id,omitempty"`
	Paroki    *ParokiResponse `json:"paroki,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type ListResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"total_pages"`
}

type ErrorResponse struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type CreateResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
