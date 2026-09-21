package routes

import (
	"KeuskupanLaboanBajo_BE/controllers"
	localMw "KeuskupanLaboanBajo_BE/middleware"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"gorm.io/gorm"
)

func RegisterRoutes(e *echo.Echo, db *gorm.DB) {
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	// ponytail: CORS untuk app+web 1 codebase (flutter web di localhost:*)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:*","http://127.0.0.1:*","http://localhost:8080","http://127.0.0.1:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	api := e.Group("/api/v1")

	// User routes (proteksi JWT biar bisa cek role)
	userController := controllers.NewUserController(db)
	users := api.Group("/users", localMw.JWTMiddleware)
	users.POST("", userController.Create)
	users.GET("", userController.List)
	users.GET("/:id", userController.Get)
	users.PUT("/:id", userController.Update)
	users.DELETE("/:id", userController.Delete)

	// Role routes
	roleController := controllers.NewRoleController(db)
	roles := api.Group("/roles")
	roles.POST("", roleController.Create)
	roles.GET("", roleController.List)
	roles.GET("/:id", roleController.Get)
	roles.PUT("/:id", roleController.Update)
	roles.DELETE("/:id", roleController.Delete)

	// Paroki routes
	parokiController := controllers.NewParokiController(db)
	parokis := api.Group("/paroki")
	parokis.POST("", parokiController.Create)
	parokis.GET("", parokiController.List)
	parokis.GET("/:id", parokiController.Get)
	parokis.PUT("/:id", parokiController.Update)
	parokis.DELETE("/:id", parokiController.Delete)

	// Keuskupan routes
	keuskupanController := controllers.NewKeuskupanController(db)
	keuskupans := api.Group("/keuskupan")
	keuskupans.POST("", keuskupanController.Create)
	keuskupans.GET("", keuskupanController.List)
	keuskupans.GET("/:id", keuskupanController.Get)
	keuskupans.PUT("/:id", keuskupanController.Update)
	keuskupans.DELETE("/:id", keuskupanController.Delete)

	// Akun routes
	akunController := controllers.NewAkunController(db)
	akuns := api.Group("/akun")
	akuns.POST("", akunController.Create)
	akuns.GET("", akunController.List)
	akuns.GET("/:id", akunController.Get)
	akuns.PUT("/:id", akunController.Update)
	akuns.DELETE("/:id", akunController.Delete)

	// Jenis routes
	jenisController := controllers.NewJenisController(db)
	jeniss := api.Group("/jenis")
	jeniss.POST("", jenisController.Create)
	jeniss.GET("", jenisController.List)
	jeniss.GET("/:id", jenisController.Get)
	jeniss.PUT("/:id", jenisController.Update)
	jeniss.DELETE("/:id", jenisController.Delete)

	// Pembatasan routes
	pembatasanController := controllers.NewPembatasanController(db)
	pembatasans := api.Group("/pembatasan")
	pembatasans.POST("", pembatasanController.Create)
	pembatasans.GET("", pembatasanController.List)
	pembatasans.GET("/:id", pembatasanController.Get)
	pembatasans.PUT("/:id", pembatasanController.Update)
	pembatasans.DELETE("/:id", pembatasanController.Delete)

	// Jurnal routes
	jurnalController := controllers.NewJurnalController(db)
	jurnals := api.Group("/jurnal", localMw.JWTMiddleware)
	jurnals.POST("", jurnalController.Create)
	jurnals.GET("", jurnalController.List)
	jurnals.GET("/:id", jurnalController.Get)
	jurnals.PUT("/:id", jurnalController.Update)
	jurnals.DELETE("/:id", jurnalController.Delete)
	jurnals.GET("/:id/detil", jurnalController.ListDetil)

	// Auth routes (public)
	authController := controllers.NewAuthController(db)
	auth := api.Group("/auth")
	auth.POST("/register", authController.Register)
	auth.POST("/login", authController.Login)
	auth.POST("/refresh", authController.Refresh)

	// Sync routes (protected JWT)
	syncController := controllers.NewSyncController(db)
	sync := api.Group("/sync", localMw.JWTMiddleware)
	sync.POST("/push", syncController.Push)
	sync.GET("/pull", syncController.Pull)
}
