package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/database"
	"github.com/kartikbazzad/bunbase/platform/internal/handlers"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

func main() {
	// Parse flags
	dbPath := flag.String("db-path", "./data/platform.db", "Database file path")
	port := flag.String("port", "3001", "Server port")
	functionsSocket := flag.String("functions-socket", "/tmp/functions.sock", "Functions service socket path")
	bundleBasePath := flag.String("bundle-path", "../functions/data/bundles", "Base path for function bundles")
	buncastSocket := flag.String("buncast-socket", "", "Buncast IPC socket path (optional; enables publish on deploy)")
	gatewayURL := flag.String("gateway-url", "http://localhost:8080", "Gateway (Traefik) base URL for client-facing project config")
	corsOrigin := flag.String("cors-origin", "http://localhost:5173", "Allowed CORS origin")
	flag.Parse()

	// Initialize database
	db, err := database.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize services
	authService := auth.NewAuth(db.DB)
	projectService := services.NewProjectService(db.DB)
	functionService, err := services.NewFunctionService(db.DB, *functionsSocket, *bundleBasePath, *buncastSocket)
	if err != nil {
		log.Fatalf("Failed to initialize function service: %v", err)
	}
	defer functionService.Close()

	tokenService := services.NewTokenService(db.DB)
	projectConfigService := services.NewProjectConfigService(*gatewayURL)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	projectHandler := handlers.NewProjectHandler(projectService)
	projectConfigHandler := handlers.NewProjectConfigHandler(projectService, projectConfigService)
	functionHandler := handlers.NewFunctionHandler(functionService, projectService)
	tokenHandler := handlers.NewTokenHandler(tokenService)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware(*corsOrigin))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api")

	// Auth routes
	authRoutes := api.Group("/auth")
	// Public auth endpoints
	authRoutes.POST("/register", authHandler.Register)
	authRoutes.POST("/login", authHandler.Login)
	// Protected auth endpoints (require session cookie)
	authProtected := authRoutes.Group("")
	authProtected.Use(middleware.AuthMiddleware(authService))
	authProtected.POST("/logout", authHandler.Logout)
	authProtected.GET("/me", authHandler.Me)
	authProtected.GET("/stream", authHandler.AuthStream)

	// Project routes (protected; accept cookie or token auth)
	projectsAPI := api.Group("/projects")
	projectsAPI.Use(middleware.AuthAnyMiddleware(authService, tokenService))
	projectsAPI.GET("", projectHandler.ListProjects)
	projectsAPI.POST("", projectHandler.CreateProject)
	projectsAPI.GET("/:id/config", projectConfigHandler.GetProjectConfig)
	projectsAPI.GET("/:id/services", projectConfigHandler.GetProjectConfig)
	projectsAPI.GET("/:id", projectHandler.GetProject)
	projectsAPI.PUT("/:id", projectHandler.UpdateProject)
	projectsAPI.PATCH("/:id", projectHandler.UpdateProject)
	projectsAPI.DELETE("/:id", projectHandler.DeleteProject)

	// Function routes (protected)
	projectsAPI.GET("/:id/functions", functionHandler.ListFunctions)
	projectsAPI.POST("/:id/functions", functionHandler.DeployFunction)
	projectsAPI.DELETE("/:id/functions/:functionId", functionHandler.DeleteFunction)

	// Token management routes (cookie-authenticated)
	tokenAPI := api.Group("/auth/tokens")
	tokenAPI.Use(middleware.AuthMiddleware(authService))
	tokenAPI.GET("", tokenHandler.List)
	tokenAPI.POST("", tokenHandler.Create)
	tokenAPI.DELETE("/:id", tokenHandler.Delete)

	// Start cleanup goroutine for expired sessions
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := authService.CleanupExpiredSessions(); err != nil {
				log.Printf("Failed to cleanup expired sessions: %v", err)
			}
		}
	}()

	// Graceful shutdown
	go func() {
		addr := fmt.Sprintf(":%s", *port)
		log.Printf("Platform API server starting on %s", addr)
		if err := router.Run(addr); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	log.Println("Server stopped")
}
