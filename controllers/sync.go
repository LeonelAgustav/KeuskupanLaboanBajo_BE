package controllers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"KeuskupanLaboanBajo_BE/models"
	"golang.org/x/crypto/bcrypt"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

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
	for _, it := range req.Items {
		log.Printf("   📦 %s %s id=%s", it.Operation, it.TargetTable, it.RowID)
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, it := range req.Items {
			switch it.TargetTable {
			case "jurnals":
				if it.Operation == "delete" {
					tx.Where("id = ?", it.RowID).Delete(&models.Jurnal{})
					continue
				}
				keuskupanID := strToUint(it.Payload["keuskupan_id"])
				if keuskupanID == 0 {
					log.Printf("⚠️ Skip jurnal %s: keuskupan_id kosong", it.RowID)
					continue
				}
				var j models.Jurnal
				if err := tx.Where("id = ?", it.RowID).First(&j).Error; err != nil {
					j.ID = it.RowID
					j.KeuskupanID = keuskupanID
					if v, ok := it.Payload["tanggal"].(string); ok {
						for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.999999", "2006-01-02T15:04:05", "2006-01-02"} {
							if t, err := time.Parse(layout, v); err == nil {
								j.Tanggal = t
								break
							}
						}
					} else {
						j.Tanggal = time.Now()
					}
					if v, ok := it.Payload["deskripsi"].(string); ok {
						j.Deskripsi = &v
					}
					if v, ok := it.Payload["no_bukti"].(string); ok && v != "" {
						j.NoBukti = &v
					}
					if uidStr, _ := c.Get("user_id").(string); uidStr != "" {
						if uid, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
							uid32 := uint(uid)
							j.CreatedBy = &uid32
						}
					}
					if err := tx.Create(&j).Error; err != nil {
						if j.NoBukti != nil && strings.Contains(err.Error(), "Duplicate") {
							newNb := *j.NoBukti + "-" + it.RowID[:4]
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
					if v, ok := it.Payload["deskripsi"].(string); ok {
						updates["deskripsi"] = v
					}
					if v, ok := it.Payload["no_bukti"].(string); ok {
						updates["no_bukti"] = v
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
					if id := strToUint(it.RowID); id != 0 {
						tx.Where("id = ?", id).Delete(&models.Keuskupan{})
					}
					continue
				}
				nama, _ := it.Payload["nama"].(string)
				alamat, _ := it.Payload["alamat"].(string)
				id := strToUint(it.RowID)
				if nama == "" {
					continue
				}
				var k models.Keuskupan
				if id != 0 {
					if err := tx.Where("id = ?", id).First(&k).Error; err == nil {
						tx.Model(&models.Keuskupan{}).Where("id = ?", id).Updates(map[string]interface{}{"nama": nama, "alamat": alamat})
						continue
					}
				}
				k.Nama = nama
				k.Alamat = alamat
				tx.Create(&k)
			case "parokis":
				if it.Operation == "delete" {
					if id := strToUint(it.RowID); id != 0 {
						tx.Where("id = ?", id).Delete(&models.Paroki{})
					}
					continue
				}
				nama, _ := it.Payload["nama"].(string)
				alamat, _ := it.Payload["alamat"].(string)
				keuskupanID := strToUint(it.Payload["keuskupan_id"])
				id := strToUint(it.RowID)
				if nama == "" || keuskupanID == 0 {
					continue
				}
				var p models.Paroki
				if id != 0 {
					if err := tx.Where("id = ?", id).First(&p).Error; err == nil {
						tx.Model(&models.Paroki{}).Where("id = ?", id).Updates(map[string]interface{}{"nama": nama, "alamat": alamat, "keuskupan_id": keuskupanID})
						continue
					}
				}
				p.Nama = nama
				p.Alamat = alamat
				p.KeuskupanID = keuskupanID
				tx.Create(&p)
			case "jenis":
				if it.Operation == "delete" {
					if id := strToUint(it.RowID); id != 0 {
						tx.Where("id = ?", id).Delete(&models.Jenis{})
					}
					continue
				}
				nama, _ := it.Payload["nama"].(string)
				id := strToUint(it.RowID)
				if nama == "" {
					continue
				}
				var j models.Jenis
				if id != 0 {
					if err := tx.Where("id = ?", id).First(&j).Error; err == nil {
						tx.Model(&models.Jenis{}).Where("id = ?", id).Updates(map[string]interface{}{"nama": nama})
						continue
					}
				}
				j.Nama = nama
				tx.Create(&j)
			case "akuns":
				if it.Operation == "delete" {
					if id := strToUint(it.RowID); id != 0 {
						tx.Where("id = ?", id).Delete(&models.Akun{})
					}
					continue
				}
				kode, _ := it.Payload["kode"].(string)
				nama, _ := it.Payload["nama"].(string)
				jenisID := strToUint(it.Payload["jenis_id"])
				id := strToUint(it.RowID)
				if kode == "" || jenisID == 0 {
					continue
				}
				var a models.Akun
				if id != 0 {
					if err := tx.Where("id = ?", id).First(&a).Error; err == nil {
						tx.Model(&models.Akun{}).Where("id = ?", id).Updates(map[string]interface{}{"kode": kode, "nama": nama, "jenis_id": jenisID})
						continue
					}
				}
				a.Kode = kode
				a.Nama = nama
				a.JenisID = jenisID
				tx.Create(&a)
			case "roles":
				if it.Operation == "delete" {
					if id := strToUint(it.RowID); id != 0 {
						tx.Where("id = ?", id).Delete(&models.Role{})
					}
					continue
				}
				nama, _ := it.Payload["nama"].(string)
				id := strToUint(it.RowID)
				if nama == "" {
					continue
				}
				var r models.Role
				if id != 0 {
					if err := tx.Where("id = ?", id).First(&r).Error; err == nil {
						tx.Model(&models.Role{}).Where("id = ?", id).Updates(map[string]interface{}{"nama": nama})
						continue
					}
				}
				r.Nama = nama
				tx.Create(&r)
			case "users":
				if it.Operation == "delete" {
					if id := strToUint(it.RowID); id != 0 {
						tx.Where("id = ?", id).Delete(&models.User{})
					}
					continue
				}
				nama, _ := it.Payload["nama"].(string)
				email, _ := it.Payload["email"].(string)
				password, _ := it.Payload["password"].(string)
				roleID := strToUint(it.Payload["role_id"])
				if nama == "" || email == "" {
					continue
				}
				var u models.User
				if err := tx.Where("email = ?", email).First(&u).Error; err == nil {
					updates := map[string]interface{}{"nama": nama, "email": email, "updated_at": time.Now()}
					if roleID != 0 {
						updates["role_id"] = roleID
					}
					if password != "" {
						hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
						updates["password_hash"] = string(hash)
					}
					tx.Model(&models.User{}).Where("id = ?", u.ID).Updates(updates)
					continue
				}
				hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				u.Nama = nama
				u.Email = email
				u.PasswordHash = string(hash)
				if roleID != 0 {
					uid := roleID
					u.RoleID = &uid
				}
				tx.Create(&u)
			case "detil_jurnals":
				if it.Operation == "delete" {
					if id := strToUint(it.RowID); id != 0 {
						tx.Where("id = ?", id).Delete(&models.DetilJurnal{})
					}
					continue
				}
				jurnalID, _ := it.Payload["jurnal_id"].(string)
				akunID := strToUint(it.Payload["akun_id"])
				parokiID := strToUint(it.Payload["paroki_id"])
				if jurnalID == "" || akunID == 0 || parokiID == 0 {
					continue
				}
				var d models.DetilJurnal
				if err := tx.Where("jurnal_id = ?", jurnalID).Order("id ASC").First(&d).Error; err != nil {
					d.JurnalID = jurnalID
					d.AkunID = akunID
					d.ParokiID = parokiID
					if v, ok := it.Payload["debit"].(float64); ok {
						d.Debit = &v
					}
					if v, ok := it.Payload["kredit"].(float64); ok {
						d.Kredit = &v
					}
					if uidStr, _ := c.Get("user_id").(string); uidStr != "" {
						if uid, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
							uid32 := uint(uid)
							d.CreatedBy = &uid32
						}
					}
					if v, ok := it.Payload["keterangan"].(string); ok {
						d.Keterangan = &v
					}
					tx.Create(&d)
				} else {
					updates := map[string]interface{}{"updated_at": time.Now()}
					if v, ok := it.Payload["debit"].(float64); ok {
						updates["debit"] = v
					}
					if v, ok := it.Payload["kredit"].(float64); ok {
						updates["kredit"] = v
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
	q := s.db.Preload("DetilJurnal").Where("updated_at > ?", since).Order("updated_at ASC").Limit(100)
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
	s.db.Unscoped().Where("deleted_at IS NOT NULL AND deleted_at > ?", since).Find(&deleted)

	// master juga biar web yang kosong dapat paroki/jenis/akun/role
	var keuskupans []models.Keuskupan
	s.db.Where("updated_at > ?", since).Find(&keuskupans)
	var parokis []models.Paroki
	s.db.Where("updated_at > ?", since).Find(&parokis)
	var jenisList []models.Jenis
	s.db.Where("updated_at > ?", since).Find(&jenisList)
	var akuns []models.Akun
	s.db.Where("updated_at > ?", since).Find(&akuns)
	var roles []models.Role
	s.db.Where("updated_at > ?", since).Find(&roles)
	var users []models.User
	userQuery := s.db.Preload("Role").Where("updated_at > ?", since)
	if role, _ := c.Get("role").(string); role == "ADMIN" {
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