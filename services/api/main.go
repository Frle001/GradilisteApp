package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	// ── Storage service ───────────────────────────────────────────────────────
	var storageSvc services.StorageService
	switch Config.UploadStorageDriver {
	case "s3":
		s3Svc, err := services.NewS3StorageService(
			Config.S3Endpoint,
			Config.S3AccessKeyID,
			Config.S3SecretKey,
			Config.S3Region,
			Config.S3Bucket,
			Config.S3PublicBaseURL,
			Config.S3UsePathStyle,
		)
		if err != nil {
			log.Fatalf("Failed to initialize S3 storage: %v", err)
		}
		storageSvc = s3Svc
	default:
		if err := os.MkdirAll(Config.LocalUploadDir, 0755); err != nil {
			log.Fatalf("Failed to create local upload dir %q: %v", Config.LocalUploadDir, err)
		}
		storageSvc = services.NewLocalStorageService(Config.LocalUploadDir)
	}

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

	mpRepo := repositories.NewMaterialPurchasesRepository(db)
	mpSvc := services.NewMaterialPurchasesService(mpRepo, auditRepo, storageSvc)
	mpHandler := handlers.NewMaterialPurchasesHandler(mpSvc)

	invRepo := repositories.NewInventoryRepository(db)
	invSvc := services.NewInventoryService(db, invRepo)
	invHandler := handlers.NewInventoryHandler(invSvc)

	whRepo := repositories.NewWorkerHoursRepository(db)
	whSvc := services.NewWorkerHoursService(whRepo)
	whHandler := handlers.NewWorkerHoursHandler(whSvc)

	// ── Router ────────────────────────────────────────────────────────────────
	if Config.Env != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(RequestIDMiddleware())
	router.Use(StructuredLogMiddleware())
	router.Use(CORSMiddleware())
	router.Use(BodySizeLimitMiddleware(Config.MaxUploadSizeMB << 20))

	// Root-level health check — used by load balancers and cloud platforms.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api")

	// ── Liveness + readiness ──────────────────────────────────────────────────
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "env": Config.Env, "version": Config.AppVersion})
	})

	api.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		dbStatus := "ok"
		if err := GetDB().Ping(ctx); err != nil {
			slog.Error("readiness: db ping failed", "error", err)
			dbStatus = "error"
		}

		storageStatus := "ok"
		if err := storageSvc.CheckHealth(ctx); err != nil {
			slog.Warn("readiness: storage check failed", "error", err)
			storageStatus = "unavailable"
		}

		status := "ready"
		httpStatus := http.StatusOK
		if dbStatus != "ok" {
			status = "not_ready"
			httpStatus = http.StatusServiceUnavailable
		}

		c.JSON(httpStatus, gin.H{
			"status":   status,
			"database": dbStatus,
			"storage":  storageStatus,
		})
	})

	// ── Auth ──────────────────────────────────────────────────────────────────
	authRL := RateLimitMiddleware(1, 5) // 1 req/s sustained, burst 5
	auth := api.Group("/auth")
	{
		auth.POST("/login", authRL, LoginHandler)
		auth.POST("/refresh", authRL, RefreshHandler)
		auth.POST("/register", RegisterHandler) // always returns 403
		auth.GET("/me", AuthRequired(), MeHandler)
		auth.POST("/logout", AuthRequired(), LogoutHandler)
		auth.PATCH("/change-password", AuthRequired(), ChangePasswordHandler)
	}

	// ── Protected test routes ─────────────────────────────────────────────────
	protected := api.Group("/protected", AuthRequired())
	{
		protected.GET("/me", ProtectedMeHandler)
		protected.GET("/director-engineer", RequireRoles("direktor", "inzenjer"), ProtectedDirectorEngineerHandler)
		protected.GET("/admin", RequireRoles("direktor", "inzenjer", "administracija"), ProtectedAdminHandler)
		protected.GET("/poslovoda", RequireRoles("direktor", "inzenjer", "poslovoda"), ProtectedPoslovodaHandler)
	}

	// ── Business modules ──────────────────────────────────────────────────────
	routes.RegisterEmployeeRoutes(api, empHandler, AuthRequired(), RequireRoles)
	projectsGroup := routes.RegisterProjectRoutes(api, projHandler, AuthRequired(), RequireRoles)
	routes.RegisterProjectMaterialRoutes(projectsGroup, matHandler, RequireRoles)
	routes.RegisterDailyReportRoutes(api, drHandler, AuthRequired(), RequireRoles)
	routes.RegisterReportsRoutes(api, reportsHandler, AuthRequired(), RequireRoles)
	routes.RegisterMaterialPurchasesRoutes(api, mpHandler, AuthRequired(), RequireRoles)
	routes.RegisterInventoryRoutes(api, invHandler, AuthRequired(), RequireRoles)
	routes.RegisterWorkerHoursRoutes(api, whHandler, AuthRequired(), RequireRoles)

	// ── Debug (development only) ──────────────────────────────────────────────
	if Config.Env == "development" {
		debug := api.Group("/debug")
		debug.GET("/db-summary", GetDBSummary)
		debug.GET("/reports", ReportsDebug)
	}

	// ── HTTP server with graceful shutdown ────────────────────────────────────
	srv := &http.Server{
		Addr:    ":" + Config.ServerPort,
		Handler: router,
	}

	go func() {
		slog.Info("starting server", "port", Config.ServerPort, "env", Config.Env, "version", Config.AppVersion)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	slog.Info("server stopped")
}
