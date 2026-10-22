package controllers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"KeuskupanLaboanBajo_BE/dto"
	"KeuskupanLaboanBajo_BE/models"
)

type JurnalController struct {
	db *gorm.DB
}

func NewJurnalController(db *gorm.DB) *JurnalController {
	return &JurnalController{db: db}
}

func (c *JurnalController) BukuBesar(ctx echo.Context) error {
	// Query params: keuskupanId, akunId, tahun, bulan, parokiId (optional)
	keuskupanID := ctx.QueryParam("keuskupanId")
	akunID := ctx.QueryParam("akunId")
	tahunStr := ctx.QueryParam("tahun")
	bulanStr := ctx.QueryParam("bulan")
	parokiID := ctx.QueryParam("parokiId")
	// Tetap terima nama lama agar klien lama tidak rusak.
	if keuskupanID == "" {
		keuskupanID = ctx.QueryParam("keuskupan_id")
	}
	if akunID == "" {
		akunID = ctx.QueryParam("akun_id")
	}
	if parokiID == "" {
		parokiID = ctx.QueryParam("paroki_id")
	}

	if keuskupanID == "" || akunID == "" || tahunStr == "" || bulanStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required params: keuskupan_id, akun_id, tahun, bulan")
	}

	keuskupanIDInt, err := strconv.Atoi(keuskupanID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid keuskupan_id")
	}
	akunIDInt, err := strconv.Atoi(akunID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid akun_id")
	}
	tahun, err := strconv.Atoi(tahunStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tahun")
	}
	bulan, err := strconv.Atoi(bulanStr)
	if err != nil || bulan < 1 || bulan > 12 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid bulan (1-12)")
	}

	// MEMBER dipaksa ke paroki sendiri, abaikan query param
	if role, _ := ctx.Get("role").(string); role == "MEMBER" {
		pidStr, _ := ctx.Get("paroki_id").(string)
		if pidStr == "" {
			return echo.NewHTTPError(http.StatusForbidden, "Paroki tidak ditemukan untuk user")
		}
		parokiID = pidStr
	}

	var parokiIDInt *int
	if parokiID != "" {
		parokiIDIntParsed, err := strconv.Atoi(parokiID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid paroki_id")
		}
		parokiIDInt = &parokiIDIntParsed
	}

	var akun models.Akun
	if err := c.db.Where("id = ? AND deleted_at IS NULL", akunIDInt).First(&akun).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Akun tidak ditemukan")
		}
		return err
	}

	// Sisi normal berdasarkan kode awalan
	kode := akun.Kode
	prefix := "1"
	if len(kode) > 0 {
		prefix = kode[:1]
	}
	debitNormal := prefix == "1" || prefix == "5"

	startDate := time.Date(tahun, time.Month(bulan), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	// Saldo awal
	parokiFilter := ""
	awalArgs := []interface{}{keuskupanIDInt, akunIDInt, startStr}
	mutArgs := []interface{}{keuskupanIDInt, akunIDInt, startStr, endStr}
	if parokiIDInt != nil {
		parokiFilter = "AND dj.paroki_id = ? "
		awalArgs = append(awalArgs, *parokiIDInt)
		mutArgs = append(mutArgs, *parokiIDInt)
	}
	var awal struct {
		TotalDebit  float64 `gorm:"column:total_debit"`
		TotalKredit float64 `gorm:"column:total_kredit"`
	}
	awalErr := c.db.Raw(`
		SELECT COALESCE(SUM(dj.debit), 0) AS total_debit,
			   COALESCE(SUM(dj.kredit), 0) AS total_kredit
		FROM (
			SELECT id, jurnal_id, paroki_id, keterangan, created_at, deleted_at,
			       COALESCE(akun_debit_id, akun_id) AS akun_id,
			       CASE WHEN akun_debit_id IS NOT NULL THEN COALESCE(jumlah, 0) ELSE COALESCE(debit, 0) END AS debit,
			       CASE WHEN akun_debit_id IS NOT NULL THEN 0 ELSE COALESCE(kredit, 0) END AS kredit
			FROM detil_jurnal
			UNION ALL
			SELECT id, jurnal_id, paroki_id, keterangan, created_at, deleted_at,
			       akun_kredit_id AS akun_id, 0 AS debit, COALESCE(jumlah, 0) AS kredit
			FROM detil_jurnal
			WHERE akun_kredit_id IS NOT NULL
		) dj
		INNER JOIN jurnals j ON dj.jurnal_id = j.id
		WHERE j.keuskupan_id = ?
		  AND dj.akun_id = ?
		  AND j.tanggal < ?
		  AND j.deleted_at IS NULL
		  AND dj.deleted_at IS NULL
		`+parokiFilter,
		awalArgs...,
	).Scan(&awal).Error
	if awalErr != nil {
		return awalErr
	}
	awalDebit, awalKredit := awal.TotalDebit, awal.TotalKredit

	saldoAwal := 0.0
	if debitNormal {
		saldoAwal = awalDebit - awalKredit
	} else {
		saldoAwal = awalKredit - awalDebit
	}

	// Mutasi dalam periode
	type MutasiRow struct {
		Tanggal    string
		NoBukti    *string
		Deskripsi  *string
		Debit      float64
		Kredit     float64
		Keterangan *string
	}
	var mutRows []MutasiRow
	query := `
		SELECT j.tanggal AS tanggal,
			   j.no_bukti AS no_bukti,
			   j.deskripsi AS deskripsi,
			   COALESCE(dj.debit, 0) AS debit,
			   COALESCE(dj.kredit, 0) AS kredit,
			   dj.keterangan AS keterangan
		FROM (
			SELECT id, jurnal_id, paroki_id, keterangan, created_at, deleted_at,
			       COALESCE(akun_debit_id, akun_id) AS akun_id,
			       CASE WHEN akun_debit_id IS NOT NULL THEN COALESCE(jumlah, 0) ELSE COALESCE(debit, 0) END AS debit,
			       CASE WHEN akun_debit_id IS NOT NULL THEN 0 ELSE COALESCE(kredit, 0) END AS kredit
			FROM detil_jurnal
			UNION ALL
			SELECT id, jurnal_id, paroki_id, keterangan, created_at, deleted_at,
			       akun_kredit_id AS akun_id, 0 AS debit, COALESCE(jumlah, 0) AS kredit
			FROM detil_jurnal
			WHERE akun_kredit_id IS NOT NULL
		) dj
		INNER JOIN jurnals j ON dj.jurnal_id = j.id
		WHERE j.keuskupan_id = ?
		  AND dj.akun_id = ?
		  AND j.tanggal >= ?
		  AND j.tanggal < ?
		  AND j.deleted_at IS NULL
		  AND dj.deleted_at IS NULL
		` + parokiFilter + `
		ORDER BY j.tanggal, dj.created_at, dj.id
	`
	queryErr := c.db.Raw(query, mutArgs...).Scan(&mutRows).Error
	if queryErr != nil {
		return queryErr
	}

	mutasi := []dto.BukuBesarMutasi{}
	saldo := saldoAwal
	for _, r := range mutRows {
		if debitNormal {
			saldo = saldo + r.Debit - r.Kredit
		} else {
			saldo = saldo + r.Kredit - r.Debit
		}
		uraian := r.Deskripsi
		if uraian == nil || *uraian == "" {
			uraian = r.Keterangan
		}
		mutasi = append(mutasi, dto.BukuBesarMutasi{
			Tanggal: r.Tanggal,
			NoBukti: r.NoBukti,
			Uraian:  uraian,
			Debit:   r.Debit,
			Kredit:  r.Kredit,
			Saldo:   saldo,
		})
	}

	return ctx.JSON(http.StatusOK, dto.BukuBesarResponse{
		AkunID:     akun.ID,
		AkunKode:   akun.Kode,
		AkunNama:   akun.Nama,
		SaldoAwal:  saldoAwal,
		Mutasi:     mutasi,
		SaldoAkhir: saldo,
	})
}

func (c *JurnalController) List(ctx echo.Context) error {
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

	db := c.db.Model(&models.Jurnal{}).Preload("Keuskupan")

	if query.Search != "" {
		search := "%" + query.Search + "%"
		db = db.Where("deskripsi LIKE ? OR no_bukti LIKE ?", search, search)
	}

	if keuskupanID := ctx.QueryParam("keuskupan_id"); keuskupanID != "" {
		db = db.Where("keuskupan_id = ?", keuskupanID)
	}

	// Filter untuk MEMBER: hanya bisa lihat jurnal milik paroki mereka sendiri
	if role, _ := ctx.Get("role").(string); role == "MEMBER" {
		if pidStr, _ := ctx.Get("paroki_id").(string); pidStr != "" {
			if pid, err := strconv.ParseUint(pidStr, 10, 32); err == nil && pid != 0 {
				db = db.Where("id IN (SELECT DISTINCT jurnal_id FROM detil_jurnal WHERE paroki_id = ? AND deleted_at IS NULL)", pid)
			}
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return err
	}

	offset := (query.Page - 1) * query.Limit
	orderBy := "tanggal DESC, created_at DESC"
	if query.Sort != "" {
		allowedSorts := map[string]string{
			"tanggal": "tanggal", "tanggal asc": "tanggal ASC", "tanggal desc": "tanggal DESC",
			"created_at": "created_at", "created_at asc": "created_at ASC", "created_at desc": "created_at DESC",
			"updated_at": "updated_at", "updated_at asc": "updated_at ASC", "updated_at desc": "updated_at DESC",
			"no_bukti": "no_bukti", "no_bukti asc": "no_bukti ASC", "no_bukti desc": "no_bukti DESC",
		}
		if val, ok := allowedSorts[strings.ToLower(query.Sort)]; ok {
			orderBy = val
		}
	}

	var jurals []models.Jurnal
	if err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&jurals).Error; err != nil {
		return err
	}

	jurnalIDs := make([]string, len(jurals))
	for i, j := range jurals {
		jurnalIDs[i] = j.ID
	}

	detilMap := c.loadDetilJurnal(jurnalIDs)

	responses := make([]dto.JurnalResponse, len(jurals))
	for i, j := range jurals {
		detils := detilMap[j.ID]
		responses[i] = dto.JurnalResponse{
			ID:          j.ID,
			KeuskupanID: j.KeuskupanID,
			Tanggal:     j.Tanggal,
			Deskripsi:   j.Deskripsi,
			NoBukti:     j.NoBukti,
			DetilJurnal: detils,
			CreatedBy:   j.CreatedBy,
			UpdatedBy:   j.UpdatedBy,
			CreatedAt:   j.CreatedAt,
			UpdatedAt:   j.UpdatedAt,
		}
		if j.Keuskupan.ID != 0 {
			responses[i].Keuskupan = &dto.KeuskupanResponse{
				ID:        j.Keuskupan.ID,
				Nama:      j.Keuskupan.Nama,
				Alamat:    j.Keuskupan.Alamat,
				CreatedAt: j.Keuskupan.CreatedAt,
				UpdatedAt: j.Keuskupan.UpdatedAt,
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

func (c *JurnalController) loadDetilJurnal(jurnalIDs []string) map[string][]dto.DetilJurnalResp {
	detilMap := make(map[string][]dto.DetilJurnalResp)
	if len(jurnalIDs) == 0 {
		return detilMap
	}

	placeholders := make([]string, len(jurnalIDs))
	args := make([]interface{}, len(jurnalIDs))
	for i, id := range jurnalIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	query := `
		SELECT dj.id, dj.jurnal_id, dj.akun_id, dj.paroki_id, dj.debit, dj.kredit, dj.keterangan, dj.created_by, dj.updated_by, dj.created_at, dj.updated_at,
		       dj.akun_debit_id, dj.akun_kredit_id, dj.jumlah,
		       a.id as akun_id2, a.kode, a.nama, a.jenis_id, a.created_at as akun_created_at, a.updated_at as akun_updated_at,
		       p.id as paroki_id2, p.nama as paroki_nama, p.alamat, p.keuskupan_id, p.created_at as paroki_created_at, p.updated_at as paroki_updated_at
		FROM detil_jurnal dj
		LEFT JOIN akun a ON dj.akun_id = a.id
		LEFT JOIN paroki p ON dj.paroki_id = p.id
		WHERE dj.jurnal_id IN (` + inClause + `)
	`

	rows, err := c.db.Raw(query, args...).Rows()
	if err != nil {
		return detilMap
	}
	defer rows.Close()

	for rows.Next() {
		var d models.DetilJurnal
		var akunID, parokiID sql.NullInt64
		var kode, nama, parokiNama, alamat sql.NullString
		var jenisID sql.NullInt64
		var akunCreatedAt, akunUpdatedAt, parokiCreatedAt, parokiUpdatedAt sql.NullTime
		var keuskupanID sql.NullInt64

		if err := rows.Scan(
			&d.ID, &d.JurnalID, &d.AkunID, &d.ParokiID, &d.Debit, &d.Kredit, &d.Keterangan, &d.CreatedBy, &d.UpdatedBy, &d.CreatedAt, &d.UpdatedAt,
			&d.AkunDebitID, &d.AkunKreditID, &d.Jumlah,
			&akunID, &kode, &nama, &jenisID, &akunCreatedAt, &akunUpdatedAt,
			&parokiID, &parokiNama, &alamat, &keuskupanID, &parokiCreatedAt, &parokiUpdatedAt,
		); err != nil {
			continue
		}

		resp := dto.DetilJurnalResp{
			ID:           d.ID,
			JurnalID:     d.JurnalID,
			AkunDebitID:  d.AkunDebitID,
			AkunKreditID: d.AkunKreditID,
			Jumlah:       d.Jumlah,
			AkunID:     d.AkunID,
			ParokiID:   d.ParokiID,
			Debit:      d.Debit,
			Kredit:     d.Kredit,
			Keterangan: d.Keterangan,
			CreatedBy:  d.CreatedBy,
			UpdatedBy:  d.UpdatedBy,
			CreatedAt:  d.CreatedAt,
			UpdatedAt:  d.UpdatedAt,
		}

		if akunID.Valid {
			resp.Akun = &dto.AkunResponse{
				ID:        uint(akunID.Int64),
				Kode:      kode.String,
				Nama:      nama.String,
				JenisID:   uint(jenisID.Int64),
				CreatedAt: akunCreatedAt.Time,
				UpdatedAt: akunUpdatedAt.Time,
			}
		}
		if parokiID.Valid {
			resp.Paroki = &dto.ParokiResponse{
				ID:          uint(parokiID.Int64),
				Nama:        parokiNama.String,
				Alamat:      alamat.String,
				KeuskupanID: uint(keuskupanID.Int64),
				CreatedAt:   parokiCreatedAt.Time,
				UpdatedAt:   parokiUpdatedAt.Time,
			}
		}
		detilMap[d.JurnalID] = append(detilMap[d.JurnalID], resp)
	}
	return detilMap
}

func (c *JurnalController) Get(ctx echo.Context) error {
	id := ctx.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var jurnal models.Jurnal
	// Use raw SQL to avoid UUID parsing issue
	query := `SELECT id, keuskupan_id, tanggal, deskripsi, no_bukti, created_by, updated_by, created_at, updated_at FROM jurnal WHERE id = ?`
	if err := c.db.Raw(query, id).Scan(&jurnal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Jurnal tidak ditemukan")
		}
		return err
	}

	if jurnal.ID == "" {
		return echo.NewHTTPError(http.StatusNotFound, "Jurnal tidak ditemukan")
	}

	// Load Keuskupan separately
	var keuskupan models.Keuskupan
	c.db.First(&keuskupan, jurnal.KeuskupanID)

	detilMap := c.loadDetilJurnal([]string{id})
	detils := detilMap[id]

	resp := dto.JurnalResponse{
		ID:          jurnal.ID,
		KeuskupanID: jurnal.KeuskupanID,
		Tanggal:     jurnal.Tanggal,
		Deskripsi:   jurnal.Deskripsi,
		NoBukti:     jurnal.NoBukti,
		DetilJurnal: detils,
		CreatedBy:   jurnal.CreatedBy,
		UpdatedBy:   jurnal.UpdatedBy,
		CreatedAt:   jurnal.CreatedAt,
		UpdatedAt:   jurnal.UpdatedAt,
	}
	if keuskupan.ID != 0 {
		resp.Keuskupan = &dto.KeuskupanResponse{
			ID:        keuskupan.ID,
			Nama:      keuskupan.Nama,
			Alamat:    keuskupan.Alamat,
			CreatedAt: keuskupan.CreatedAt,
			UpdatedAt: keuskupan.UpdatedAt,
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (c *JurnalController) Create(ctx echo.Context) error {
	var req dto.JurnalCreateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	jurnalID := uuid.New().String()

	// MEMBER validation: must use their own paroki_id
	var memberParokiID uint = 0
	if role, _ := ctx.Get("role").(string); role == "MEMBER" {
		if pidStr, _ := ctx.Get("paroki_id").(string); pidStr != "" {
			if pid, err := strconv.ParseUint(pidStr, 10, 32); err == nil {
				memberParokiID = uint(pid)
			}
		}
	}

	err := c.db.Transaction(func(tx *gorm.DB) error {
		jurnal := models.Jurnal{
			ID:          jurnalID,
			KeuskupanID: req.KeuskupanID,
			Tanggal:     req.Tanggal.Time,
			Deskripsi:   req.Deskripsi,
			NoBukti:     req.NoBukti,
		}
		if err := tx.Create(&jurnal).Error; err != nil {
			return err
		}

		for _, d := range req.DetilJurnal {
			if memberParokiID != 0 && d.ParokiID != memberParokiID {
				return echo.NewHTTPError(http.StatusForbidden, "MEMBER hanya bisa menambah jurnal untuk parokinya sendiri")
			}
			detil := models.DetilJurnal{
				JurnalID:   jurnalID,
				AkunID:     d.AkunID,
				ParokiID:   d.ParokiID,
				Debit:      d.Debit,
				Kredit:     d.Kredit,
				Keterangan: d.Keterangan,
				AkunDebitID:  d.AkunDebitID,
				AkunKreditID: d.AkunKreditID,
				Jumlah:       d.Jumlah,
			}
			if err := tx.Create(&detil).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	var created models.Jurnal
	// Use raw SQL to avoid UUID parsing issue
	query := `SELECT id, keuskupan_id, tanggal, deskripsi, no_bukti, created_at, updated_at FROM jurnal WHERE id = ?`
	if err := c.db.Raw(query, jurnalID).Scan(&created).Error; err != nil {
		return err
	}

	if created.ID == "" {
		return echo.NewHTTPError(http.StatusNotFound, "Jurnal tidak ditemukan")
	}

	// Load Keuskupan separately
	var keuskupan models.Keuskupan
	c.db.First(&keuskupan, created.KeuskupanID)

	detilMap := c.loadDetilJurnal([]string{jurnalID})
	detils := detilMap[jurnalID]

	return ctx.JSON(http.StatusCreated, dto.CreateResponse{
		Message: "Data Berhasil di Buat",
		Data: dto.JurnalResponse{
			ID:          created.ID,
			KeuskupanID: created.KeuskupanID,
			Tanggal:     created.Tanggal,
			Deskripsi:   created.Deskripsi,
			NoBukti:     created.NoBukti,
			DetilJurnal: detils,
			CreatedAt:   created.CreatedAt,
			UpdatedAt:   created.UpdatedAt,
		},
	})
}

func (c *JurnalController) Update(ctx echo.Context) error {
	id := ctx.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.JurnalUpdateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if err := ctx.Validate(&req); err != nil {
		return err
	}

	// Capture user_id from JWT for audit trail
	var updatedBy *uint
	var createdBy *uint
	var memberParokiID uint = 0
	if uidStr, ok := ctx.Get("user_id").(string); ok && uidStr != "" {
		if uid, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
			uid32 := uint(uid)
			updatedBy = &uid32
			createdBy = &uid32
		}
	}
	if role, _ := ctx.Get("role").(string); role == "MEMBER" {
		if pidStr, _ := ctx.Get("paroki_id").(string); pidStr != "" {
			if pid, err := strconv.ParseUint(pidStr, 10, 32); err == nil {
				memberParokiID = uint(pid)
			}
		}
	}

	err := c.db.Transaction(func(tx *gorm.DB) error {
		var jurnal models.Jurnal
		if err := tx.Raw("SELECT id, keuskupan_id FROM jurnal WHERE id = ? AND deleted_at IS NULL", id).Scan(&jurnal).Error; err != nil {
			return err
		}
		if jurnal.ID == "" {
			return echo.NewHTTPError(http.StatusNotFound, "Jurnal tidak ditemukan")
		}

		updates := map[string]interface{}{"updated_at": time.Now()}
		if req.KeuskupanID != nil {
			updates["keuskupan_id"] = *req.KeuskupanID
		}
		if req.Tanggal != nil {
			updates["tanggal"] = req.Tanggal.Time
		}
		if req.Deskripsi != nil {
			updates["deskripsi"] = *req.Deskripsi
		}
		if req.NoBukti != nil {
			updates["no_bukti"] = *req.NoBukti
		}
		if updatedBy != nil {
			updates["updated_by"] = *updatedBy
		}
		if len(updates) > 1 {
			if err := tx.Model(&models.Jurnal{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}

		if req.DetilJurnal != nil {
			// Delete old detil_jurnal
			if err := tx.Where("jurnal_id = ?", id).Delete(&models.DetilJurnal{}).Error; err != nil {
				return err
			}
			// Insert new detil_jurnal
			insertQuery := `INSERT INTO detil_jurnal (jurnal_id, akun_id, paroki_id, debit, kredit, keterangan, created_by, updated_by, akun_debit_id, akun_kredit_id, jumlah, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
			for _, d := range req.DetilJurnal {
				if memberParokiID != 0 && d.ParokiID != memberParokiID {
					return echo.NewHTTPError(http.StatusForbidden, "MEMBER hanya bisa menambah jurnal untuk parokinya sendiri")
				}
				if err := tx.Exec(insertQuery, id, d.AkunID, d.ParokiID, d.Debit, d.Kredit, d.Keterangan, createdBy, updatedBy, d.AkunDebitID, d.AkunKreditID, d.Jumlah).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	var updated models.Jurnal
	// Use raw SQL to avoid UUID parsing issue
	query := `SELECT id, keuskupan_id, tanggal, deskripsi, no_bukti, created_by, updated_by, created_at, updated_at FROM jurnal WHERE id = ?`
	if err := c.db.Raw(query, id).Scan(&updated).Error; err != nil {
		return err
	}

	if updated.ID == "" {
		return echo.NewHTTPError(http.StatusNotFound, "Jurnal tidak ditemukan")
	}

	// Load Keuskupan separately
	var keuskupan models.Keuskupan
	c.db.First(&keuskupan, updated.KeuskupanID)

	detilMap := c.loadDetilJurnal([]string{id})
	detils := detilMap[id]

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"message": "Data Berhasil di Perbarui",
		"data": dto.JurnalResponse{
			ID:          updated.ID,
			KeuskupanID: updated.KeuskupanID,
			Tanggal:     updated.Tanggal,
			Deskripsi:   updated.Deskripsi,
			NoBukti:     updated.NoBukti,
			DetilJurnal: detils,
			CreatedBy:   updated.CreatedBy,
			UpdatedBy:   updated.UpdatedBy,
			CreatedAt:   updated.CreatedAt,
			UpdatedAt:   updated.UpdatedAt,
		},
	})
}

func (c *JurnalController) Delete(ctx echo.Context) error {
	id := ctx.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

	result := c.db.Exec("DELETE FROM jurnal WHERE id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Jurnal tidak ditemukan")
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "Data Berhasil di Hapus",
	})
}

func (c *JurnalController) ListDetil(ctx echo.Context) error {
	id := ctx.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID tidak valid")
	}

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

	db := c.db.Model(&models.DetilJurnal{}).Where("jurnal_id = ?", id).Preload("Akun").Preload("Paroki")

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return err
	}

	offset := (query.Page - 1) * query.Limit
	orderBy := "id ASC"
	if query.Sort != "" {
		allowedSorts := map[string]string{
			"id": "id", "id asc": "id ASC", "id desc": "id DESC",
			"akun_id": "akun_id", "akun_id asc": "akun_id ASC", "akun_id desc": "akun_id DESC",
			"paroki_id": "paroki_id", "paroki_id asc": "paroki_id ASC", "paroki_id desc": "paroki_id DESC",
			"created_at": "created_at", "created_at asc": "created_at ASC", "created_at desc": "created_at DESC",
			"updated_at": "updated_at", "updated_at asc": "updated_at ASC", "updated_at desc": "updated_at DESC",
		}
		if val, ok := allowedSorts[strings.ToLower(query.Sort)]; ok {
			orderBy = val
		}
	}

	var detils []models.DetilJurnal
	if err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&detils).Error; err != nil {
		return err
	}

	responses := make([]dto.DetilJurnalResp, len(detils))
	for i, d := range detils {
		responses[i] = dto.DetilJurnalResp{
			ID:           d.ID,
			JurnalID:     d.JurnalID,
			AkunDebitID:  d.AkunDebitID,
			AkunKreditID: d.AkunKreditID,
			Jumlah:       d.Jumlah,
			AkunID:     d.AkunID,
			ParokiID:   d.ParokiID,
			Debit:      d.Debit,
			Kredit:     d.Kredit,
			Keterangan: d.Keterangan,
			CreatedBy:  d.CreatedBy,
			UpdatedBy:  d.UpdatedBy,
			CreatedAt:  d.CreatedAt,
			UpdatedAt:  d.UpdatedAt,
		}
		if d.Akun.ID != 0 {
			responses[i].Akun = &dto.AkunResponse{
				ID:        d.Akun.ID,
				Kode:      d.Akun.Kode,
				Nama:      d.Akun.Nama,
				JenisID:   d.Akun.JenisID,
				CreatedAt: d.Akun.CreatedAt,
				UpdatedAt: d.Akun.UpdatedAt,
			}
		}
		if d.Paroki.ID != 0 {
			responses[i].Paroki = &dto.ParokiResponse{
				ID:          d.Paroki.ID,
				Nama:        d.Paroki.Nama,
				Alamat:      d.Paroki.Alamat,
				KeuskupanID: d.Paroki.KeuskupanID,
				CreatedAt:   d.Paroki.CreatedAt,
				UpdatedAt:   d.Paroki.UpdatedAt,
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
