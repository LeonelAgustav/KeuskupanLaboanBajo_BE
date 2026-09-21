package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"KeuskupanLaboanBajo_BE/models"
	"golang.org/x/crypto/bcrypt"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func parseDBURL(raw string) (user, pass, host, port, name string, ok bool) {
	if raw == "" {
		return "", "", "", "", "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", "", "", "", false
	}
	if u.Scheme != "mysql" {
		return "", "", "", "", "", false
	}
	user = u.User.Username()
	pass, _ = u.User.Password()
	host = u.Hostname()
	port = u.Port()
	if port == "" {
		port = "3306"
	}
	name = strings.TrimPrefix(u.Path, "/")
	if user == "" || host == "" || name == "" {
		return "", "", "", "", "", false
	}
	return user, pass, host, port, name, true
}

func ensureDatabaseExists(user, pass, host, port, name string) {
	dsnRoot := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, host, port)
	dbRoot, err := gorm.Open(mysql.Open(dsnRoot), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Gagal konek MySQL server untuk create DB: %v", err)
	}
	sqlDB, _ := dbRoot.DB()
	defer sqlDB.Close()
	createSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", name)
	if err := dbRoot.Exec(createSQL).Error; err != nil {
		log.Fatalf("❌ Gagal create database %s: %v", name, err)
	}
	log.Printf("✅ Database `%s` siap (created if not exists)", name)
}

func ConnectDB() *gorm.DB {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Warning: .env tidak ditemukan, pakai env sistem")
	}

	var dsn string
	var user, pass, host, port, name string

	if dbURL := os.Getenv("DB_URL"); dbURL != "" {
		var ok bool
		user, pass, host, port, name, ok = parseDBURL(dbURL)
		if !ok {
			log.Fatalf("❌ DB_URL format salah, harus mysql://user:pass@host:port/dbname, got: %s", dbURL)
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			user, pass, host, port, name)
		log.Printf("🔗 Pakai DB_URL -> %s:***@%s:%s/%s", user, host, port, name)
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{SingularTable: true},
			Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			}),
		})
		if err != nil {
			log.Fatalf("❌ Gagal koneksi database: %v", err)
		}
		if err := db.AutoMigrate(
			&models.Role{},
			&models.User{},
			&models.Keuskupan{},
			&models.Paroki{},
			&models.Jenis{},
			&models.Akun{},
			&models.Jurnal{},
			&models.DetilJurnal{},
			&models.Pembatasan{},
		); err != nil {
			log.Fatalf("❌ Gagal AutoMigrate: %v", err)
		}
		seedRoles(db)
		seedUsers(db)
		log.Println("✅ Koneksi database berhasil + AutoMigrate OK!")
		return db
	}

	user = os.Getenv("DB_USER")
	pass = os.Getenv("DB_PASS")
	host = os.Getenv("DB_HOST")
	port = os.Getenv("DB_PORT")
	name = os.Getenv("DB_NAME")

	if user == "" || host == "" || name == "" {
		log.Fatal("❌ DB_USER/DB_HOST/DB_NAME kosong, cek .env atau set DB_URL")
	}
	if port == "" {
		port = "3306"
	}
	ensureDatabaseExists(user, pass, host, port, name)

	dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, host, port, name)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		}),
	})
	if err != nil {
		log.Fatalf("❌ Gagal koneksi database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.Role{},
		&models.User{},
		&models.Keuskupan{},
		&models.Paroki{},
		&models.Jenis{},
		&models.Akun{},
		&models.Jurnal{},
		&models.DetilJurnal{},
		&models.Pembatasan{},
	); err != nil {
		log.Fatalf("❌ Gagal AutoMigrate: %v", err)
	}

	seedRoles(db)
	seedUsers(db)

	log.Println("✅ Koneksi database berhasil + AutoMigrate OK!")
	return db
}

func seedRoles(db *gorm.DB) {
	roles := []string{"SUPERADMIN", "ADMIN", "MEMBER"}
	for _, nama := range roles {
		var count int64
		db.Model(&models.Role{}).Where("nama = ?", nama).Count(&count)
		if count == 0 {
			if err := db.Create(&models.Role{Nama: nama}).Error; err != nil {
				log.Printf("⚠️ Gagal seed role %s: %v", nama, err)
			} else {
				log.Printf("✅ Role %s seeded", nama)
			}
		}
	}
}

func seedUsers(db *gorm.DB) {
	// Find SUPERADMIN role
	var role models.Role
	if err := db.Where("nama = ?", "SUPERADMIN").First(&role).Error; err != nil {
		log.Printf("⚠️ Gagal seed user: role SUPERADMIN tidak ditemukan")
		return
	}

	var count int64
	db.Model(&models.User{}).Where("email = ?", "superadmin@keuskupan.test").Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		user := models.User{
			Nama:         "Super Admin",
			Email:        "superadmin@keuskupan.test",
			PasswordHash: string(hash),
			RoleID:       &role.ID,
		}
		if err := db.Create(&user).Error; err != nil {
			log.Printf("⚠️ Gagal seed superadmin: %v", err)
		} else {
			log.Printf("✅ Superadmin seeded (email: superadmin@keuskupan.test, pass: password123)")
		}
	}
}