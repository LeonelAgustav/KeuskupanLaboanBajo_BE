package routes

import (
	"os"
	"strings"
	"time"

	"KeuskupanLaboanBajo_BE/controllers"
	localMw "KeuskupanLaboanBajo_BE/middleware"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"gorm.io/gorm"
)

func RegisterRoutes(e *echo.Echo, db *gorm.DB) {
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            31536000,
		HSTSExcludeSubdomains: false,
		HSTSPreloadEnabled:    true,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}))
	e.Use(middleware.BodyLimit("2M"))
	// ponytail: CORS dynamic origins from env
	corsOrigins := strings.Split(os.Getenv("CORS_ORIGINS"), ",")
	if len(corsOrigins) == 1 && corsOrigins[0] == "" {
		corsOrigins = []string{"*"}
	}
	if len(corsOrigins) == 1 && corsOrigins[0] == "*" {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
			AllowCredentials: false,
		}))
	} else {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     corsOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
			AllowCredentials: true,
		}))
	}

	// ponytail: whitelist sort columns, prevent SQL injection in ORDER BY
	sortValidator := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sortRaw := c.QueryParam("sort")
			if sortRaw == "" {
				return next(c)
			}
			allowed := map[string]bool{
				"id": true, "nama": true, "kode": true, "email": true,
				"tanggal": true, "created_at": true, "updated_at": true,
				"created_at desc": true, "created_at asc": true,
				"updated_at desc": true, "updated_at asc": true,
				"id desc": true, "id asc": true,
				"nama desc": true, "nama asc": true,
				"kode desc": true, "kode asc": true,
				"email desc": true, "email asc": true,
				"tanggal desc": true, "tanggal asc": true,
			}
			// support comma-separated: "tanggal desc, created_at desc"
			parts := strings.Split(sortRaw, ",")
			for _, p := range parts {
				if !allowed[strings.TrimSpace(strings.ToLower(p))] {
					c.QueryParams().Del("sort")
					return next(c)
				}
			}
			return next(c)
		}
	}
	e.Use(sortValidator)

	// Auth routes (public)
	authController := controllers.NewAuthController(db)
	auth := e.Group("/api/v1/auth")
	// register: 3 per jam
	auth.POST("/register", authController.Register, middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{Rate: 3, Burst: 3, ExpiresIn: 1 * time.Hour}),
	}))
	// login: 5 per menit
	auth.POST("/login", authController.Login, middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{Rate: 5, Burst: 5, ExpiresIn: 1 * time.Minute}),
	}))
	// refresh: 60 per menit
	auth.POST("/refresh", authController.Refresh, middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{Rate: 60, Burst: 60, ExpiresIn: 1 * time.Minute}),
	}))

	// API (Protected)
	apiGroup := e.Group("/api/v1", localMw.JWTMiddleware)

	// User routes (ADMIN+ saja)
	userController := controllers.NewUserController(db)
	users := apiGroup.Group("/users", localMw.RequireRole("SUPERADMIN", "ADMIN"))
	users.POST("", userController.Create)
	users.GET("", userController.List)
	users.GET("/:id", userController.Get)
	users.PUT("/:id", userController.Update)
	users.DELETE("/:id", userController.Delete)

	// Role routes (SUPERADMIN saja)
	roleController := controllers.NewRoleController(db)
	roles := apiGroup.Group("/roles", localMw.RequireRole("SUPERADMIN"))
	roles.POST("", roleController.Create)
	roles.GET("", roleController.List)
	roles.GET("/:id", roleController.Get)
	roles.PUT("/:id", roleController.Update)
	roles.DELETE("/:id", roleController.Delete)

	// Paroki routes (ADMIN+)
	parokiController := controllers.NewParokiController(db)
	parokis := apiGroup.Group("/paroki", localMw.RequireRole("SUPERADMIN", "ADMIN"))
	parokis.POST("", parokiController.Create)
	parokis.GET("", parokiController.List)
	parokis.GET("/:id", parokiController.Get)
	parokis.PUT("/:id", parokiController.Update)
	parokis.DELETE("/:id", parokiController.Delete)

	// Keuskupan routes (SUPERADMIN only)
	keuskupanController := controllers.NewKeuskupanController(db)
	keuskupans := apiGroup.Group("/keuskupan", localMw.RequireRole("SUPERADMIN"))
	keuskupans.POST("", keuskupanController.Create)
	keuskupans.GET("", keuskupanController.List)
	keuskupans.GET("/:id", keuskupanController.Get)
	keuskupans.PUT("/:id", keuskupanController.Update)
	keuskupans.DELETE("/:id", keuskupanController.Delete)

	// Akun routes (ADMIN+)
	akunController := controllers.NewAkunController(db)
	akuns := apiGroup.Group("/akun", localMw.RequireRole("SUPERADMIN", "ADMIN"))
	akuns.POST("", akunController.Create)
	akuns.GET("", akunController.List)
	akuns.GET("/:id", akunController.Get)
	akuns.PUT("/:id", akunController.Update)
	akuns.DELETE("/:id", akunController.Delete)

	// Jenis routes (ADMIN+)
	jenisController := controllers.NewJenisController(db)
	jeniss := apiGroup.Group("/jenis", localMw.RequireRole("SUPERADMIN", "ADMIN"))
	jeniss.POST("", jenisController.Create)
	jeniss.GET("", jenisController.List)
	jeniss.GET("/:id", jenisController.Get)
	jeniss.PUT("/:id", jenisController.Update)
	jeniss.DELETE("/:id", jenisController.Delete)

	// Pembatasan routes (ADMIN+)
	pembatasanController := controllers.NewPembatasanController(db)
	pembatasans := apiGroup.Group("/pembatasan", localMw.RequireRole("SUPERADMIN", "ADMIN"))
	pembatasans.POST("", pembatasanController.Create)
	pembatasans.GET("", pembatasanController.List)
	pembatasans.GET("/:id", pembatasanController.Get)
	pembatasans.PUT("/:id", pembatasanController.Update)
	pembatasans.DELETE("/:id", pembatasanController.Delete)

	// Jurnal routes (all authenticated roles)
	jurnalController := controllers.NewJurnalController(db)
	jurnals := apiGroup.Group("/jurnal")
	jurnals.POST("", jurnalController.Create)
	jurnals.GET("", jurnalController.List)
	jurnals.GET("/:id", jurnalController.Get)
	jurnals.PUT("/:id", jurnalController.Update)
	jurnals.DELETE("/:id", jurnalController.Delete)
	jurnals.GET("/:id/detil", jurnalController.ListDetil)

	// Sync routes
	syncController := controllers.NewSyncController(db)
	sync := apiGroup.Group("/sync")
	// push: 30 per menit per user, cegah abuse sync flood
	sync.POST("/push", syncController.Push, middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store:   middleware.NewRateLimiterMemoryStore(30),
		IdentifierExtractor: func(c echo.Context) (string, error) {
			id, _ := c.Get("user_id").(string)
			return id, nil
		},
	}))
	sync.GET("/pull", syncController.Pull)
}
