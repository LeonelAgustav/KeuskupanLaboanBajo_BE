package dto

type JurnalSyncDTO struct {
	KeuskupanID uint    `json:"keuskupan_id" validate:"required"`
	Tanggal     string  `json:"tanggal" validate:"required"`
	Deskripsi   *string `json:"deskripsi,omitempty"`
	NoBukti     *string `json:"no_bukti,omitempty"`
}

type DetilJurnalSyncDTO struct {
	JurnalID     string   `json:"jurnal_id" validate:"required"`
	AkunID       uint     `json:"akun_id" validate:"required"`
	ParokiID     uint     `json:"paroki_id" validate:"required"`
	AkunDebitID  *uint    `json:"akun_debit_id,omitempty"`
	AkunKreditID *uint    `json:"akun_kredit_id,omitempty"`
	Jumlah       *float64 `json:"jumlah,omitempty"`
	Debit        *float64 `json:"debit,omitempty"`
	Kredit       *float64 `json:"kredit,omitempty"`
	Keterangan   *string  `json:"keterangan,omitempty"`
}

type MasterSyncDTO struct {
	UUID        *string `json:"uuid,omitempty"`
	Nama        *string `json:"nama,omitempty"`
	Alamat      *string `json:"alamat,omitempty"`
	KeuskupanID *uint   `json:"keuskupan_id,omitempty"`
	Kode        *string `json:"kode,omitempty"`
	JenisID     *uint   `json:"jenis_id,omitempty"`
	Email       *string `json:"email,omitempty"`
	Password    *string `json:"password,omitempty"`
	RoleID      *uint   `json:"role_id,omitempty"`
	ParokiID    *uint   `json:"paroki_id,omitempty"`
}
