package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"KeuskupanLaboanBajo_BE/dto"
	"KeuskupanLaboanBajo_BE/models"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

var syncValidate = validator.New()

type SyncController struct {
	db *gorm.DB
}

func NewSyncController(db *gorm.DB) *SyncController {
	return &SyncController{db: db}
}

type PushItem struct {
	TargetTable string                 `json:"target_table" validate:"required"`
	RowID       string                 `json:"row_id" validate:"required"`
	Operation   string                 `json:"operation" validate:"required,oneof=insert update delete"`
	Payload     map[string]interface{} `json:"payload"`
}

type PushRequest struct {
	Items []PushItem `json:"items" validate:"required,min=1,dive"`
}

func sanitizePayload(p map[string]interface{}) {
	// ponytail: strip meta fields that must never be mass-assigned
	for _, k := range []string{"created_at", "updated_at", "deleted_at", "is_synced", "created_by", "updated_by", "deleted_by", "id"} {
		delete(p, k)
	}
}

func parseDTO[T any](payload map[string]interface{}) (*T, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func writeAudit(tx *gorm.DB, c echo.Context, op, table, rowID string, payload map[string]interface{}) {
	var uid *uint
	if uidStr, _ := c.Get("user_id").(string); uidStr != "" {
		if n, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
			v := uint(n)
			uid = &v
		}
	}
	action := strings.ToUpper(op)
	if action != "INSERT" && action != "UPDATE" && action != "DELETE" {
		action = strings.ToUpper(op)
	}
	pb, _ := json.Marshal(payload)
	tx.Create(&models.AuditLog{
		UserID:    uid,
		Action:    action,
		TableName: table,
		RowID:     rowID,
		Payload:   string(pb),
		IPAddress: c.RealIP(),
	})
}

func strToUint(v interface{}) uint {
	switch t := v.(type) {
	case float64:
		return uint(t)
	case string:
		var n uint64
		for _, r := range t {
			if r < '0' || r > '9' {
				return 0
			}
			n = n*10 + uint64(r-'0')
		}
		return uint(n)
	}
	return 0
}

// syncUUID: FE kirim UUID di row_id; payload.uuid dipakai kalau row_id kosong.
func syncUUID(rowID string, dto *string) string {
	if dto != nil && *dto != "" {
		return *dto
	}
	return rowID
}

func (s *SyncController) Push(c echo.Context) error {
	var req PushRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("❌ Push: bind error: %v", err)
		return err
	}
	if err := c.Validate(&req); err != nil {
		log.Printf("❌ Push: validate error: %v", err)
		return err
	}

	log.Printf("📥 Push received: %d items", len(req.Items))
	role, _ := c.Get("role").(string)
	var pJwt uint
	if pidStr, _ := c.Get("paroki_id").(string); pidStr != "" {
		if pid, err := strconv.ParseUint(pidStr, 10, 32); err == nil {
			pJwt = uint(pid)
		}
	}
	for _, it := range req.Items {
		log.Printf("   📦 %s %s id=%s", it.Operation, it.TargetTable, it.RowID)
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, it := range req.Items {
			if len(it.RowID) == 0 {
				continue
			}
			if role == "MEMBER" {
				switch it.TargetTable {
				case "jurnals", "detil_jurnals":
				default:
					continue
				}
			}
			sanitizePayload(it.Payload)
			writeAudit(tx, c, it.Operation, it.TargetTable, it.RowID, it.Payload)
			switch it.TargetTable {
		case "jurnals":
			if it.Operation == "delete" {
				if role == "MEMBER" && pJwt != 0 {
					var n int64
					tx.Model(&models.DetilJurnal{}).Where("jurnal_id = ? AND paroki_id = ?", it.RowID, pJwt).Count(&n)
					if n == 0 {
						continue
					}
				}
				tx.Where("id = ?", it.RowID).Delete(&models.Jurnal{})
				continue
			}
			jDTO, err := parseDTO[dto.JurnalSyncDTO](it.Payload)
			if err != nil || syncValidate.Struct(jDTO) != nil || jDTO.KeuskupanID == 0 {
				log.Printf("⚠️ Skip jurnal %s: DTO invalid", it.RowID)
				continue
			}
			var j models.Jurnal
			if err := tx.Where("id = ?", it.RowID).First(&j).Error; err != nil {
				j.ID = it.RowID
				j.KeuskupanID = jDTO.KeuskupanID
				if v := jDTO.Tanggal; v != "" {
					for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.999999", "2006-01-02T15:04:05", "2006-01-02"} {
						if t, err := time.Parse(layout, v); err == nil {
							j.Tanggal = t
							break
						}
					}
				} else {
					j.Tanggal = time.Now()
				}
				j.Deskripsi = jDTO.Deskripsi
				j.NoBukti = jDTO.NoBukti
				if uidStr, _ := c.Get("user_id").(string); uidStr != "" {
					if uid, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
						uid32 := uint(uid)
						j.CreatedBy = &uid32
					}
				}
				if err := tx.Create(&j).Error; err != nil {
					if j.NoBukti != nil && strings.Contains(err.Error(), "Duplicate") {
						rowSuffix := it.RowID
						if len(rowSuffix) > 4 {
							rowSuffix = rowSuffix[:4]
						}
						newNb := *j.NoBukti + "-" + rowSuffix
						j.NoBukti = &newNb
						if err2 := tx.Create(&j).Error; err2 != nil {
							log.Printf("⚠️ Create jurnal %s error (retry): %v", it.RowID, err2)
							continue
						}
					} else {
						log.Printf("⚠️ Create jurnal %s error: %v", it.RowID, err)
						continue
					}
				}
			} else {
				updates := map[string]interface{}{"updated_at": time.Now()}
				if jDTO.Deskripsi != nil {
					updates["deskripsi"] = *jDTO.Deskripsi
				}
				if jDTO.NoBukti != nil && *jDTO.NoBukti != "" {
					updates["no_bukti"] = *jDTO.NoBukti
				}
				if uidStr, _ := c.Get("user_id").(string); uidStr != "" {
					if uid, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
						updates["updated_by"] = uint(uid)
					}
				}
				tx.Model(&models.Jurnal{}).Where("id = ?", it.RowID).Updates(updates)
			}
		case "keuskupans":
			if it.Operation == "delete" {
				if it.RowID != "" {
					tx.Where("uuid = ?", it.RowID).Delete(&models.Keuskupan{})
				}
				continue
			}
			kDTO, err := parseDTO[dto.MasterSyncDTO](it.Payload)
			if err != nil || kDTO.Nama == nil || *kDTO.Nama == "" {
				log.Printf("⚠️ Skip keuskupan %s: DTO invalid", it.RowID)
				continue
			}
			uuid := syncUUID(it.RowID, kDTO.UUID)
			if uuid != "" {
				var k models.Keuskupan
				if err := tx.Where("uuid = ?", uuid).First(&k).Error; err == nil {
					updates := map[string]interface{}{"nama": *kDTO.Nama, "updated_at": time.Now()}
					if kDTO.Alamat != nil {
						updates["alamat"] = *kDTO.Alamat
					}
					tx.Model(&models.Keuskupan{}).Where("uuid = ?", uuid).Updates(updates)
					continue
				}
			}
			k := models.Keuskupan{Nama: *kDTO.Nama}
			if uuid != "" {
				k.UUID = uuid
			}
			if kDTO.Alamat != nil {
				k.Alamat = *kDTO.Alamat
			}
			tx.Create(&k)
		case "parokis":
			if it.Operation == "delete" {
				if it.RowID != "" {
					tx.Where("uuid = ?", it.RowID).Delete(&models.Paroki{})
				}
				continue
			}
			pDTO, err := parseDTO[dto.MasterSyncDTO](it.Payload)
			if err != nil || pDTO.Nama == nil || *pDTO.Nama == "" || pDTO.KeuskupanID == nil || *pDTO.KeuskupanID == 0 {
				log.Printf("⚠️ Skip paroki %s: DTO invalid", it.RowID)
				continue
			}
			uuid := syncUUID(it.RowID, pDTO.UUID)
			if uuid != "" {
				var p models.Paroki
				if err := tx.Where("uuid = ?", uuid).First(&p).Error; err == nil {
					updates := map[string]interface{}{"nama": *pDTO.Nama, "keuskupan_id": *pDTO.KeuskupanID, "updated_at": time.Now()}
					if pDTO.Alamat != nil {
						updates["alamat"] = *pDTO.Alamat
					}
					tx.Model(&models.Paroki{}).Where("uuid = ?", uuid).Updates(updates)
					continue
				}
			}
			p := models.Paroki{Nama: *pDTO.Nama, KeuskupanID: *pDTO.KeuskupanID}
			if uuid != "" {
				p.UUID = uuid
			}
			if pDTO.Alamat != nil {
				p.Alamat = *pDTO.Alamat
			}
			tx.Create(&p)
		case "jenis":
			if it.Operation == "delete" {
				if it.RowID != "" {
					tx.Where("uuid = ?", it.RowID).Delete(&models.Jenis{})
				}
				continue
			}
			jDTO, err := parseDTO[dto.MasterSyncDTO](it.Payload)
			if err != nil || jDTO.Nama == nil || *jDTO.Nama == "" {
				log.Printf("⚠️ Skip jenis %s: DTO invalid", it.RowID)
				continue
			}
			uuid := syncUUID(it.RowID, jDTO.UUID)
			if uuid != "" {
				var j models.Jenis
				if err := tx.Where("uuid = ?", uuid).First(&j).Error; err == nil {
					tx.Model(&models.Jenis{}).Where("uuid = ?", uuid).Updates(map[string]interface{}{"nama": *jDTO.Nama, "updated_at": time.Now()})
					continue
				}
			}
			j := models.Jenis{Nama: *jDTO.Nama}
			if uuid != "" {
				j.UUID = uuid
			}
			tx.Create(&j)
		case "akuns":
			if it.Operation == "delete" {
				if it.RowID != "" {
					tx.Where("uuid = ?", it.RowID).Delete(&models.Akun{})
				}
				continue
			}
			aDTO, err := parseDTO[dto.MasterSyncDTO](it.Payload)
			if err != nil || aDTO.Kode == nil || *aDTO.Kode == "" || aDTO.JenisID == nil || *aDTO.JenisID == 0 {
				log.Printf("⚠️ Skip akun %s: DTO invalid", it.RowID)
				continue
			}
			uuid := syncUUID(it.RowID, aDTO.UUID)
			if uuid != "" {
				var a models.Akun
				if err := tx.Where("uuid = ?", uuid).First(&a).Error; err == nil {
					updates := map[string]interface{}{"kode": *aDTO.Kode, "jenis_id": *aDTO.JenisID, "updated_at": time.Now()}
					if aDTO.Nama != nil {
						updates["nama"] = *aDTO.Nama
					}
					tx.Model(&models.Akun{}).Where("uuid = ?", uuid).Updates(updates)
					continue
				}
			}
			a := models.Akun{Kode: *aDTO.Kode, JenisID: *aDTO.JenisID}
			if uuid != "" {
				a.UUID = uuid
			}
			if aDTO.Nama != nil {
				a.Nama = *aDTO.Nama
			}
			tx.Create(&a)
		case "roles":
			if r, _ := c.Get("role").(string); r != "SUPERADMIN" {
				continue
			}
			if it.Operation == "delete" {
				if id := strToUint(it.RowID); id != 0 {
					tx.Where("id = ?", id).Delete(&models.Role{})
				}
				continue
			}
			rDTO, err := parseDTO[dto.MasterSyncDTO](it.Payload)
			if err != nil || rDTO.Nama == nil || *rDTO.Nama == "" {
				log.Printf("⚠️ Skip role %s: DTO invalid", it.RowID)
				continue
			}
			id := strToUint(it.RowID)
			var ro models.Role
			if id != 0 {
				if err := tx.Where("id = ?", id).First(&ro).Error; err == nil {
					tx.Model(&models.Role{}).Where("id = ?", id).Updates(map[string]interface{}{"nama": *rDTO.Nama, "updated_at": time.Now()})
					continue
				}
			}
			ro.Nama = *rDTO.Nama
			tx.Create(&ro)
		case "users":
			if r, _ := c.Get("role").(string); r != "SUPERADMIN" {
				continue
			}
			if it.Operation == "delete" {
				if id := strToUint(it.RowID); id != 0 {
					tx.Where("id = ?", id).Delete(&models.User{})
				}
				continue
			}
			uDTO, err := parseDTO[dto.MasterSyncDTO](it.Payload)
			if err != nil || uDTO.Nama == nil || *uDTO.Nama == "" || uDTO.Email == nil || *uDTO.Email == "" {
				log.Printf("⚠️ Skip user %s: DTO invalid", it.RowID)
				continue
			}
			id := strToUint(it.RowID)
			var u models.User
			if id != 0 {
				if err := tx.Where("id = ?", id).First(&u).Error; err == nil {
					updates := map[string]interface{}{"nama": *uDTO.Nama, "email": *uDTO.Email, "updated_at": time.Now()}
					if uDTO.RoleID != nil && *uDTO.RoleID != 0 {
						updates["role_id"] = *uDTO.RoleID
					}
					if uDTO.ParokiID != nil && *uDTO.ParokiID != 0 {
						updates["paroki_id"] = *uDTO.ParokiID
					}
					if uDTO.Password != nil && *uDTO.Password != "" {
						hash, errBcrypt := bcrypt.GenerateFromPassword([]byte(*uDTO.Password), bcrypt.DefaultCost)
						if errBcrypt != nil {
							continue
						}
						updates["password_hash"] = string(hash)
					}
					tx.Model(&models.User{}).Where("id = ?", u.ID).Updates(updates)
					continue
				}
			}
			u.Nama = *uDTO.Nama
			u.Email = *uDTO.Email
			if uDTO.RoleID != nil && *uDTO.RoleID != 0 {
				u.RoleID = uDTO.RoleID
			}
			if uDTO.ParokiID != nil && *uDTO.ParokiID != 0 {
				u.ParokiID = uDTO.ParokiID
			}
			if uDTO.Password != nil && *uDTO.Password != "" {
				hash, errBcrypt := bcrypt.GenerateFromPassword([]byte(*uDTO.Password), bcrypt.DefaultCost)
				if errBcrypt != nil {
					continue
				}
				u.PasswordHash = string(hash)
			} else {
				log.Printf("⚠️ Skip user %s: password kosong", it.RowID)
				continue
			}
			if err := tx.Create(&u).Error; err != nil {
				log.Printf("⚠️ Create user %s error: %v", it.RowID, err)
				continue
			}
			case "detil_jurnals":
				if len(it.RowID) == 0 {
					continue
				}
				if it.Operation == "delete" {
					if id := strToUint(it.RowID); id != 0 {
						if role == "MEMBER" && pJwt != 0 {
							var owned int64
							tx.Model(&models.DetilJurnal{}).Where("id = ? AND paroki_id = ?", id, pJwt).Count(&owned)
							if owned == 0 {
								continue
							}
						}
						tx.Where("id = ?", id).Delete(&models.DetilJurnal{})
					}
					continue
				}
				dDTO, err := parseDTO[dto.DetilJurnalSyncDTO](it.Payload)
				if err != nil || syncValidate.Struct(dDTO) != nil || dDTO.AkunID == 0 {
					log.Printf("⚠️ Skip detil_jurnal %s: DTO invalid", it.RowID)
					continue
				}
				jurnalID := dDTO.JurnalID
				akunID := dDTO.AkunID
				parokiID := pJwt
				if parokiID == 0 {
					if role == "MEMBER" {
						continue
					}
					parokiID = dDTO.ParokiID
				}
				if jurnalID == "" || akunID == 0 || parokiID == 0 {
					continue
				}
				var d models.DetilJurnal
				if err := tx.Where("jurnal_id = ?", jurnalID).Order("id ASC").First(&d).Error; err != nil {
					d.JurnalID = jurnalID
					d.AkunID = akunID
					d.ParokiID = parokiID
					d.Debit = dDTO.Debit
					d.Kredit = dDTO.Kredit
					d.AkunDebitID = dDTO.AkunDebitID
					d.AkunKreditID = dDTO.AkunKreditID
					d.Jumlah = dDTO.Jumlah
					if uidStr, _ := c.Get("user_id").(string); uidStr != "" {
						if uid, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
							uid32 := uint(uid)
							d.CreatedBy = &uid32
						}
					}
					d.Keterangan = dDTO.Keterangan
					tx.Create(&d)
				} else {
					updates := map[string]interface{}{"updated_at": time.Now()}
					if parokiID != 0 {
						updates["paroki_id"] = parokiID
					}
					if dDTO.Debit != nil {
						updates["debit"] = *dDTO.Debit
					}
					if dDTO.Kredit != nil {
						updates["kredit"] = *dDTO.Kredit
					}
					if dDTO.AkunDebitID != nil {
						updates["akun_debit_id"] = *dDTO.AkunDebitID
					}
					if dDTO.AkunKreditID != nil {
						updates["akun_kredit_id"] = *dDTO.AkunKreditID
					}
					if dDTO.Jumlah != nil {
						updates["jumlah"] = *dDTO.Jumlah
					}
					if uidStr, _ := c.Get("user_id").(string); uidStr != "" {
						if uid, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
							updates["updated_by"] = uint(uid)
						}
					}
					tx.Model(&models.DetilJurnal{}).Where("id = ?", d.ID).Updates(updates)
				}
			default:
				log.Printf("⚠️ Push: skip unknown table %s", it.TargetTable)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("❌ Push transaction error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Gagal sync push")
	}
	log.Printf("✅ Push done: %d items", len(req.Items))
	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Sync push berhasil", "count": len(req.Items)})
}

func (s *SyncController) Pull(c echo.Context) error {
	sinceStr := c.QueryParam("since")
	var since time.Time
	if sinceStr != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.999999", "2006-01-02T15:04:05.999", "2006-01-02T15:04:05.000", "2006-01-02T15:04:05", "2006-01-02"} {
			if t, err := time.Parse(layout, sinceStr); err == nil {
				since = t
				break
			}
		}
	}
	log.Printf("📤 Pull request: since=%s", since.Format(time.RFC3339))

	var jurnals []models.Jurnal
	q := s.db.Preload("DetilJurnal").Order("updated_at ASC").Limit(100)
	// MEMBER hanya bisa pull jurnal milik paroki mereka
	if role, _ := c.Get("role").(string); role == "MEMBER" {
		if pidStr, _ := c.Get("paroki_id").(string); pidStr != "" {
			if pid, err := strconv.ParseUint(pidStr, 10, 32); err == nil && pid != 0 {
				q = q.Where("id IN (SELECT DISTINCT jurnal_id FROM detil_jurnal WHERE paroki_id = ? AND deleted_at IS NULL)", pid)
			}
		}
	}
	if err := q.Find(&jurnals).Error; err != nil {
		return err
	}
	var deleted []models.Jurnal
	s.db.Unscoped().Where("deleted_at IS NOT NULL").Find(&deleted)

	// master juga biar web yang kosong dapat paroki/jenis/akun/role
	var keuskupans []models.Keuskupan
	s.db.Find(&keuskupans)
	var parokis []models.Paroki
	s.db.Find(&parokis)
	var jenisList []models.Jenis
	s.db.Find(&jenisList)
	var akuns []models.Akun
	s.db.Find(&akuns)
	var roles []models.Role
	s.db.Find(&roles)
	var users []models.User
	userQuery := s.db.Preload("Role")
	role, _ := c.Get("role").(string)
	if role == "MEMBER" {
		// MEMBER hanya lihat user di paroki yang sama
		if pidStr, _ := c.Get("paroki_id").(string); pidStr != "" {
			if pid, err := strconv.ParseUint(pidStr, 10, 32); err == nil && pid != 0 {
				userQuery = userQuery.Where("paroki_id = ?", pid)
			} else {
				userQuery = userQuery.Where("1 = 0") // Block if no paroki
			}
		} else {
			userQuery = userQuery.Where("1 = 0")
		}
	} else if role == "ADMIN" {
		var superRole models.Role
		if err := s.db.Where("nama = ?", "SUPERADMIN").First(&superRole).Error; err == nil {
			userQuery = userQuery.Where("role_id != ? OR role_id IS NULL", superRole.ID)
		}
	}
	userQuery.Find(&users)

	log.Printf("📤 Pull result: jurnals=%d, deleted=%d, keuskupans=%d, parokis=%d, jenis=%d, akuns=%d, roles=%d, users=%d",
		len(jurnals), len(deleted), len(keuskupans), len(parokis), len(jenisList), len(akuns), len(roles), len(users))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"since":      since,
		"jurnals":    jurnals,
		"deleted":    deleted,
		"count":      len(jurnals),
		"keuskupans": keuskupans,
		"parokis":    parokis,
		"jenis":      jenisList,
		"akuns":      akuns,
		"roles":      roles,
		"users":      users,
	})
}