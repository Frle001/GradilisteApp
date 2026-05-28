package main

import (
	"log"
	"os"
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	// Initialize database
	db, err := InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize router
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	// API v1 routes (future)
	api := router.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"message": "Gradiliste API is running",
			})
		})

		api.GET("/db-health", func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			database := GetDB()
			if database == nil {
				c.JSON(500, gin.H{
					"status": "error",
					"error":  "database pool is not initialized",
				})
				return
			}

			if err := database.Ping(ctx); err != nil {
				c.JSON(500, gin.H{
					"status": "error",
					"error":  err.Error(),
				})
				return
			}

			c.JSON(200, gin.H{
				"status":  "ok",
				"message": "Database connection is working",
			})
		})
		// Auth endpoints (future)
		// auth := api.Group("/auth")
		// {
		// 	auth.POST("/login", handlers.Login)
		// 	auth.POST("/register", handlers.Register)
		// }

		// Employee endpoints (future)
		// employees := api.Group("/employees")
		// {
		// 	employees.GET("", handlers.ListEmployees)
		// 	employees.POST("", handlers.CreateEmployee)
		// }
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Gradilište API on port %s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
