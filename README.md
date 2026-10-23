# Keuskupan Labuan Bajo - Backend (BE)

Sistem Jurnal Keuangan Keuskupan Labuan Bajo - Offline First. REST API untuk sinkronisasi data antara klien Flutter dan database pusat.

## Stack
- Go (module `KeuskupanLaboanBajo_BE`)
- Echo Framework v4
- GORM + driver MySQL
- JWT (`golang-jwt/jwt/v5`)
- Validator (`go-playground/validator`)

## Dependency (`go.mod`)

### Langsung
| Package | Versi | Fungsi |
|---------|-------|--------|
| `github.com/labstack/echo/v4` | v4.12.0 | Web framework HTTP, routing, middleware |
| `gorm.io/gorm` | v1.31.2 | ORM untuk akses database |
| `gorm.io/driver/mysql` | v1.6.0 | Driver MySQL untuk GORM |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | Generate & verifikasi token JWT (auth) |
| `github.com/go-playground/validator/v10` | v10.14.0 | Validasi struct body request |
| `github.com/google/uuid` | v1.6.0 | Generate ID unik (jurnal) |
| `github.com/joho/godotenv` | v1.5.1 | Baca konfigurasi dari file `.env` |
| `golang.org/x/crypto` | v0.55.0 | Hashing password (bcrypt) |

## Cara Menjalankan
```bash
go run main.go
```

Untuk dev di jaringan lokal (HP + laptop satu WiFi), pastikan `BIND_ADDR=0.0.0.0` dan izinkan port `8080` di Windows Firewall.

## Endpoint Utama
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/register`
- `POST /api/v1/sync/push` — kirim antrean perubahan dari klien
- `GET  /api/v1/sync/pull?since=...` — tarik data terbaru (dengan resolusi konflik)
