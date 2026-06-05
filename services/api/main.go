package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/gradiliste/api/handlers"
	"github.com/gradiliste/api/repositories"
	"github.com/gradiliste/api/routes"
	"github.com/gradiliste/api/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	LoadConfig()

	db, err := InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// ── Dependency injection ──────────────────────────────────────────────────
	auditRepo := repositories.NewAuditRepository(db)
	userRepo := repositories.NewUserRepository(db)
	empRepo := repositories.NewEmployeeRepository(db)
	assetRepo := repositories.NewEmployeeAssetRepository(db)
	empSvc := services.NewEmployeeService(db, empRepo, assetRepo, auditRepo, userRepo, HashPassword)
	empHandler := handlers.NewEmployeeHandler(empSvc)

	projRepo := repositories.NewProjectRepository(db)
	projSvc := services.NewProjectService(db, projRepo, auditRepo)
	projHandler := handlers.NewProjectHandler(projSvc)

	matRepo := repositories.NewProjectMaterialRepository(db)
	importRepo := repositories.NewImportJobRepository(db)
	matSvc := services.NewProjectMaterialService(db, matRepo, importRepo, auditRepo, projRepo)
	matHandler := handlers.NewProjectMaterialHandler(matSvc)

	drRepo := repositories.NewDailyReportRepository(db)
	drSvc := services.NewDailyReportService(db, drRepo, auditRepo)
	drHandler := handlers.NewDailyReportHandler(drSvc)

	reportsRepo := repositories.NewReportsRepository(db)
	reportsSvc := services.NewReportsService(reportsRepo)
	reportsHandler := handlers.NewReportsHandler(reportsSvc)

	storageSvc := services.NewLocalStorageService(Config.UploadsDir)
	mpRepo := repositories.NewMaterialPurchasesRepository(db)
	mpSvc := services.NewMaterialPurchasesService(mpRepo, auditRepo, storageSvc)
	mpHandler := handlers.NewMaterialPurchasesHandler(mpSvc)

	invRepo := repositories.NewInventoryRepository(db)
	invSvc := services.NewInventoryService(db, invRepo)
	invHandler := handlers.NewInventoryHandler(invSvc)

	// ── Router ────────────────────────────────────────────────────────────────
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	api := router.Group("/api")

	// ── Public ────────────────────────────────────────────────────────────────
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Gradiliste API is running",
		})
	})

	api.GET("/db-health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := GetDB().Ping(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Database connection is working"})
	})

	// ── Auth ──────────────────────────────────────────────────────────────────
	auth := api.Group("/auth")
	{
		auth.POST("/login", LoginHandler)
		auth.POST("/register", RegisterHandler)            // always 403 — public registration disabled
		auth.GET("/me", AuthRequired(), MeHandler)
		auth.POST("/logout", AuthRequired(), LogoutHandler)
		auth.PATCH("/change-password", AuthRequired(), ChangePasswordHandler)
	}

	// ── Protected test routes (verify middleware wiring) ──────────────────────
	protected := api.Group("/protected", AuthRequired())
	{
		protected.GET("/me", ProtectedMeHandler)
		protected.GET("/director-engineer",
			RequireRoles("direktor", "inzenjer"),
			ProtectedDirectorEngineerHandler,
		)
		protected.GET("/admin",
			RequireRoles("direktor", "inzenjer", "administracija"),
			ProtectedAdminHandler,
		)
		protected.GET("/poslovoda",
			RequireRoles("direktor", "inzenjer", "poslovoda"),
			ProtectedPoslovodaHandler,
		)
	}

	// ── Business modules ──────────────────────────────────────────────────────
	routes.RegisterEmployeeRoutes(api, empHandler, AuthRequired(), RequireRoles)
	projectsGroup := routes.RegisterProjectRoutes(api, projHandler, AuthRequired(), RequireRoles)
	routes.RegisterProjectMaterialRoutes(projectsGroup, matHandler, RequireRoles)
	routes.RegisterDailyReportRoutes(api, drHandler, AuthRequired(), RequireRoles)
	routes.RegisterReportsRoutes(api, reportsHandler, AuthRequired(), RequireRoles)
	routes.RegisterMaterialPurchasesRoutes(api, mpHandler, AuthRequired(), RequireRoles)
	routes.RegisterInventoryRoutes(api, invHandler, AuthRequired(), RequireRoles)

	// ── Debug (development only) ──────────────────────────────────────────────
	if os.Getenv("ENV") == "development" {
		debug := api.Group("/debug")
		debug.GET("/db-summary", GetDBSummary)
		debug.GET("/reports", ReportsDebug)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Gradilište API on port %s (env: %s)\n", port, Config.Env)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
