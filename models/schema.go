package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SKEMA ID (disamakan dgn target Prisma):
// - Master (Keuskupan, Paroki, Jenis, Akun, DetilJurnal, Pembatasan, User): int auto-increment
// - Jurnal: string cuid (dibuat offline di client, id tetap string)
// - DetilJurnal.JurnalID: string (FK ke Jurnal)

type User struct {
	ID           uint   `gorm:"primaryKey"`
	UUID         string `gorm:"type:varchar(36);uniqueIndex;not null"`
	Nama         string `gorm:"not null;size:255"`
	Email        string `gorm:"uniqueIndex;not null;size:255"`
	PasswordHash string `gorm:"not null;size:255" json:"-"`
	TokenVersion uint   `gorm:"default:1;not null"`
	RoleID       *uint  `gorm:"index"`
	Role         *Role  `gorm:"foreignKey:RoleID;references:ID"`
	ParokiID     *uint  `gorm:"index"`
	Paroki       *Paroki `gorm:"foreignKey:ParokiID;references:ID"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.UUID == "" {
		u.UUID = uuid.New().String()
	}
	return nil
}

type Keuskupan struct {
	ID        uint     `gorm:"primaryKey"`
	UUID      string   `gorm:"type:varchar(36);uniqueIndex;not null"`
	Nama      string   `gorm:"not null"`
	Alamat    string   `gorm:"not null"`
	Paroki    []Paroki `gorm:"foreignKey:KeuskupanID"`
	Jurnal    []Jurnal `gorm:"foreignKey:KeuskupanID"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (k *Keuskupan) BeforeCreate(tx *gorm.DB) error {
	if k.UUID == "" {
		k.UUID = uuid.New().String()
	}
	return nil
}

type Paroki struct {
	ID          uint          `gorm:"primaryKey"`
	UUID        string        `gorm:"type:varchar(36);uniqueIndex;not null"`
	Nama        string        `gorm:"not null"`
	Alamat      string        `gorm:"not null"`
	KeuskupanID uint          `gorm:"index;not null"`
	Keuskupan   Keuskupan     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;foreignKey:KeuskupanID;references:ID"`
	DetilJurnal []DetilJurnal `gorm:"foreignKey:ParokiID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (p *Paroki) BeforeCreate(tx *gorm.DB) error {
	if p.UUID == "" {
		p.UUID = uuid.New().String()
	}
	return nil
}

type Jenis struct {
	ID        uint   `gorm:"primaryKey"`
	UUID      string `gorm:"type:varchar(36);uniqueIndex;not null"`
	Nama      string `gorm:"uniqueIndex;not null;size:100"`
	Akun      []Akun `gorm:"foreignKey:JenisID"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (j *Jenis) BeforeCreate(tx *gorm.DB) error {
	if j.UUID == "" {
		j.UUID = uuid.New().String()
	}
	return nil
}

type Akun struct {
	ID          uint          `gorm:"primaryKey"`
	UUID        string        `gorm:"type:varchar(36);uniqueIndex;not null"`
	Kode        string        `gorm:"uniqueIndex;not null;size:20"`
	Nama        string        `gorm:"not null;size:255"`
	JenisID     uint          `gorm:"index;not null"`
	Jenis       Jenis         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;foreignKey:JenisID;references:ID"`
	DetilJurnal []DetilJurnal `gorm:"foreignKey:AkunID"`
	Pembatasan  []Pembatasan  `gorm:"foreignKey:AkunID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (a *Akun) BeforeCreate(tx *gorm.DB) error {
	if a.UUID == "" {
		a.UUID = uuid.New().String()
	}
	return nil
}

type Jurnal struct {
	ID            string        `gorm:"primaryKey;size:50"`
	KeuskupanID   uint          `gorm:"index;not null"`
	Keuskupan     Keuskupan     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;foreignKey:KeuskupanID;references:ID"`
	DetilJurnal   []DetilJurnal `gorm:"foreignKey:JurnalID"`
	Tanggal       time.Time     `gorm:"index;not null"`
	Deskripsi     *string       `gorm:"size:500"`
	NoBukti       *string       `gorm:"uniqueIndex;size:50"`
	CreatedBy     *uint         `gorm:"index"`
	CreatedByUser *User         `gorm:"foreignKey:CreatedBy;references:ID"`
	UpdatedBy     *uint         `gorm:"index"`
	UpdatedByUser *User         `gorm:"foreignKey:UpdatedBy;references:ID"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (j *Jurnal) BeforeCreate(tx *gorm.DB) error {
	if j.ID == "" {
		j.ID = "c_" + uuid.New().String()
	}
	if j.Tanggal.IsZero() {
		j.Tanggal = time.Now()
	}
	return nil
}

type DetilJurnal struct {
	ID            uint     `gorm:"primaryKey"`
	JurnalID      string   `gorm:"index;not null;size:50"`
	Jurnal        Jurnal   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;foreignKey:JurnalID;references:ID"`
	AkunID        uint     `gorm:"index;not null"`
	Akun          Akun     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;foreignKey:AkunID;references:ID"`
	ParokiID      uint     `gorm:"index;not null"`
	Paroki        Paroki   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;foreignKey:ParokiID;references:ID"`
	Debit         *float64 `gorm:"type:decimal(15,2);default:0"`
	Kredit        *float64 `gorm:"type:decimal(15,2);default:0"`
	Keterangan    *string
	CreatedBy     *uint    `gorm:"index"`
	CreatedByUser *User    `gorm:"foreignKey:CreatedBy;references:ID"`
	UpdatedBy     *uint    `gorm:"index"`
	UpdatedByUser *User    `gorm:"foreignKey:UpdatedBy;references:ID"`
	CreatedAt     time.Time `gorm:"index"`
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

type Pembatasan struct {
	ID        uint     `gorm:"primaryKey"`
	Tipe      string   `gorm:"not null"`
	Nilai     *float64 `gorm:"type:decimal(15,2)"`
	AkunID    *uint
	Akun      *Akun `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:AkunID;references:ID"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Role struct {
	ID        uint   `gorm:"primaryKey"`
	Nama      string `gorm:"not null"`
	User      []User `gorm:"foreignKey:RoleID"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type AuditLog struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    *uint     `gorm:"index"`
	User      *User     `gorm:"foreignKey:UserID;references:ID"`
	Action    string    `gorm:"size:20;not null;index"` // INSERT, UPDATE, DELETE
	TableName string    `gorm:"size:50;not null;index"`
	RowID     string    `gorm:"size:64;not null;index"`
	Payload   string    `gorm:"type:text"`
	IPAddress string    `gorm:"size:45"`
	CreatedAt time.Time `gorm:"index"`
}
