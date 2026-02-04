package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/pkg/bunauth"
	"github.com/kartikbazzad/bunbase/pkg/config"

	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/bundoc"
	"github.com/kartikbazzad/bunbase/platform/internal/database"
	"github.com/kartikbazzad/bunbase/platform/internal/handlers"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

type AppConfig struct {
	Port   int             `mapstructure:"port"`
	DB     database.Config `mapstructure:"db"`
	Bundoc struct {
		URL string `mapstructure:"url"`
	} `mapstructure:"bundoc"`
}

func main() {
	// Parse flags
	port := flag.String("port", "3001", "Server port")
	// Note: functions-socket etc should also be env vars or config, but keeping flags for now
	bundleBasePath := flag.String("bundle-path", "../functions/data/bundles", "Base path for function bundles")
	buncastSocket := flag.String("buncast-socket", "", "Buncast IPC socket path (optional; enables publish on deploy)")
	gatewayURL := flag.String("gateway-url", "http://localhost:8080", "Gateway (Traefik) base URL for client-facing project config")
	corsOrigin := flag.String("cors-origin", "http://localhost:5173", "Allowed CORS origin")
	authURL := flag.String("auth-url", "http://localhost:8081", "BunAuth Service URL")
	functionsURL := flag.String("functions-url", "http://localhost:8080", "Functions Service URL")
	// Use relative path for internal builder script by default (works if running from platform root)
	builderScript := flag.String("builder-script", "./internal/builder/builder.ts", "Path to builder script")

	flag.Parse()

	// Docker Support: Override flags from ENV if present
	if val := os.Getenv("PLATFORM_AUTH_URL"); val != "" {
		*authURL = val
	}
	if val := os.Getenv("PLATFORM_FUNCTIONS_URL"); val != "" {
		*functionsURL = val
	}
	if val := os.Getenv("PLATFORM_BUNDLE_PATH"); val != "" {
		*bundleBasePath = val
	}
	if val := os.Getenv("PLATFORM_BUILDER_SCRIPT"); val != "" {
		*builderScript = val
	}

	// Load Config
	var cfg AppConfig

	// Set defaults
	// Set defaults
	if os.Getenv("PLATFORM_PORT") == "" {
		os.Setenv("PLATFORM_PORT", *port)
	}
	if os.Getenv("PLATFORM_DB_HOST") == "" {
		os.Setenv("PLATFORM_DB_HOST", "localhost")
	}
	if os.Getenv("PLATFORM_DB_PORT") == "" {
		os.Setenv("PLATFORM_DB_PORT", "5432")
	}
	if os.Getenv("PLATFORM_DB_USER") == "" {
		os.Setenv("PLATFORM_DB_USER", "bunadmin")
	}
	if os.Getenv("PLATFORM_DB_PASSWORD") == "" {
		os.Setenv("PLATFORM_DB_PASSWORD", "bunpassword")
	}
	if os.Getenv("PLATFORM_DB_NAME") == "" {
		os.Setenv("PLATFORM_DB_NAME", "bunbase_system")
	}
	if os.Getenv("PLATFORM_DB_MIGRATIONSPATH") == "" {
		os.Setenv("PLATFORM_DB_MIGRATIONSPATH", "./platform/migrations")
	}
	if os.Getenv("PLATFORM_BUNDOC_URL") == "" {
		os.Setenv("PLATFORM_BUNDOC_URL", "http://localhost:8080")
	}

	if err := config.Load("PLATFORM_", &cfg); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Loaded DB Config: Host=%s Port=%d MigrationsPath=%s", cfg.DB.Host, cfg.DB.Port, cfg.DB.MigrationsPath)

	// Initialize database
	db, err := database.NewDB(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to Postgres successfully")

	// Initialize services
	// Use localhost for local dev, or env for docker.
	// Use localhost for local dev, or env for docker.
	tenantAuthURL := "http://localhost:8083"
	if url := os.Getenv("TENANT_AUTH_URL"); url != "" {
		tenantAuthURL = url
	}

	authClient := bunauth.NewClient(*authURL)
	tenantAuthClient := auth.NewTenantClient(tenantAuthURL)
	bundocClient := bundoc.NewClient(cfg.Bundoc.URL)

	authService := auth.NewAuth(authClient)

	projectService := services.NewProjectService(db.Pool)

	absBundlePath, err := filepath.Abs(*bundleBasePath)
	if err != nil {
		log.Fatalf("Failed to resolve bundle path: %v", err)
	}

	absBuilderScript, err := filepath.Abs(*builderScript)
	if err != nil {
		log.Fatalf("Failed to resolve builder script path: %v", err)
	}

	functionService, err := services.NewFunctionService(db.Pool, *functionsURL, absBundlePath, *buncastSocket, absBuilderScript)
	if err != nil {
		log.Fatalf("Failed to initialize function service: %v", err)
	}
	defer functionService.Close()

	tokenService := services.NewTokenService(db.Pool)
	projectConfigService := services.NewProjectConfigService(*gatewayURL)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	tenantAuthHandler := handlers.NewTenantAuthHandler(tenantAuthClient, projectService)
	projectHandler := handlers.NewProjectHandler(projectService)
	projectConfigHandler := handlers.NewProjectConfigHandler(projectService, projectConfigService)
	functionHandler := handlers.NewFunctionHandler(functionService, projectService, *functionsURL)
	tokenHandler := handlers.NewTokenHandler(tokenService)
	databaseHandler := handlers.NewDatabaseHandler(bundocClient, projectService)

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

	// v1 API (Public SDK)
	v1 := router.Group("/v1")
	v1.Use(middleware.CORSMiddleware(*corsOrigin)) // Use configured origin for v1 as well

	// v1Auth := v1.Group("/auth")
	// v1Auth.POST("/register", tenantAuthHandler.Register)
	// v1Auth.POST("/login", tenantAuthHandler.Login)

	v1DB := v1.Group("/databases")
	// /v1/databases/:dbName/collections/:collection/documents
	v1DB.POST("/:dbName/collections/:collection/documents", databaseHandler.ProxyHandler)
	v1DB.GET("/:dbName/collections/:collection/documents", databaseHandler.ProxyHandler)
	v1DB.GET("/:dbName/collections/:collection/documents/:docID", databaseHandler.ProxyHandler)
	v1DB.PUT("/:dbName/collections/:collection/documents/:docID", databaseHandler.ProxyHandler)
	v1DB.DELETE("/:dbName/collections/:collection/documents/:docID", databaseHandler.ProxyHandler)
	v1DB.POST("/:dbName/collections/:collection/indexes", databaseHandler.ProxyHandler)
	v1DB.DELETE("/:dbName/collections/:collection/indexes/:field", databaseHandler.ProxyHandler)

	v1Functions := v1.Group("/functions")
	v1Functions.POST("/:name/invoke", functionHandler.InvokeFunction)

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
	authProtected.GET("/stream", authHandler.AuthStream) // Added Stream

	// Alias v1 auth routes to standard auth handlers for Web Console consistency if using v1 prefix
	// The Web Console uses /v1 in its API client now, so we should expose the standard auth handlers there too.
	v1AuthStandard := v1.Group("/auth")
	v1AuthStandard.POST("/register", authHandler.Register)
	v1AuthStandard.POST("/login", authHandler.Login)
	v1AuthStandard.Use(middleware.AuthMiddleware(authService))
	v1AuthStandard.POST("/logout", authHandler.Logout)
	v1AuthStandard.GET("/me", authHandler.Me)
	v1AuthStandard.GET("/stream", authHandler.AuthStream)

	// Also alias Project routes to /v1/projects for Web Console
	v1Projects := v1.Group("/projects")
	v1Projects.Use(middleware.AuthAnyMiddleware(authService, tokenService))
	v1Projects.GET("", projectHandler.ListProjects)
	v1Projects.POST("", projectHandler.CreateProject)
	v1Projects.GET("/:id/config", projectConfigHandler.GetProjectConfig)
	v1Projects.GET("/:id/services", projectConfigHandler.GetProjectConfig)
	v1Projects.GET("/:id", projectHandler.GetProject)
	v1Projects.PUT("/:id", projectHandler.UpdateProject)
	v1Projects.PATCH("/:id", projectHandler.UpdateProject)
	v1Projects.DELETE("/:id", projectHandler.DeleteProject)

	// Function routes (v1)
	v1Projects.GET("/:id/functions", functionHandler.ListFunctions)
	v1Projects.GET("/:id/functions/logs", functionHandler.GetProjectFunctionLogs)
	v1Projects.POST("/:id/functions", functionHandler.DeployFunction)
	v1Projects.DELETE("/:id/functions/:functionId", functionHandler.DeleteFunction)

	// Database routes (v1 - developer access)
	v1ProjectDB := v1Projects.Group("/:id/database")
	v1ProjectDB.GET("/collections", databaseHandler.DeveloperProxyHandler)
	v1ProjectDB.POST("/collections", databaseHandler.DeveloperProxyHandler)
	v1ProjectDB.DELETE("/collections/:collection", databaseHandler.DeveloperProxyHandler)
	v1ProjectDB.POST("/collections/:collection/indexes", databaseHandler.DeveloperProxyHandler)
	v1ProjectDB.DELETE("/collections/:collection/indexes/:field", databaseHandler.DeveloperProxyHandler)
	v1ProjectDB.GET("/collections/:collection/documents", databaseHandler.DeveloperProxyHandler)
	v1ProjectDB.POST("/collections/:collection/documents", databaseHandler.DeveloperProxyHandler)
	v1ProjectDB.GET("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	v1ProjectDB.PUT("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	v1ProjectDB.DELETE("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)

	// Project routes (protected; accept cookie or token auth)
	projectsAPI := api.Group("/projects")
	projectsAPI.Use(middleware.AuthAnyMiddleware(authService, tokenService))
	projectsAPI.GET("", projectHandler.ListProjects)
	projectsAPI.POST("", projectHandler.CreateProject)
	projectsAPI.GET("/:id/config", projectConfigHandler.GetProjectConfig)
	projectsAPI.GET("/:id/services", projectConfigHandler.GetProjectConfig) // Alias
	projectsAPI.GET("/:id", projectHandler.GetProject)
	projectsAPI.PUT("/:id", projectHandler.UpdateProject)
	projectsAPI.PATCH("/:id", projectHandler.UpdateProject)
	projectsAPI.DELETE("/:id", projectHandler.DeleteProject)

	// Function routes (protected)
	projectsAPI.GET("/:id/functions", functionHandler.ListFunctions)
	projectsAPI.GET("/:id/functions/logs", functionHandler.GetProjectFunctionLogs)
	projectsAPI.POST("/:id/functions", functionHandler.DeployFunction)
	projectsAPI.DELETE("/:id/functions/:functionId", functionHandler.DeleteFunction)
	projectsAPI.POST("/:id/functions/:name/invoke", functionHandler.InvokeProjectFunction)
	projectsAPI.GET("/:id/functions/:name/invoke", functionHandler.InvokeProjectFunction)

	// Database routes (protected - developer access)
	projectDB := projectsAPI.Group("/:id/database")
	// /api/projects/:id/database/collections
	projectDB.GET("/collections", databaseHandler.DeveloperProxyHandler)
	projectDB.POST("/collections", databaseHandler.DeveloperProxyHandler)
	projectDB.PATCH("/collections/:collection", databaseHandler.DeveloperProxyHandler) // Schema Update
	projectDB.DELETE("/collections/:collection", databaseHandler.DeveloperProxyHandler)

	// Indexes
	projectDB.GET("/collections/:collection/indexes", databaseHandler.DeveloperProxyHandler)
	projectDB.POST("/collections/:collection/indexes", databaseHandler.DeveloperProxyHandler)
	projectDB.DELETE("/collections/:collection/indexes/:field", databaseHandler.DeveloperProxyHandler)

	// Documents
	// /api/projects/:id/database/collections/:collection/documents
	projectDB.GET("/collections/:collection/documents", databaseHandler.DeveloperProxyHandler)
	projectDB.POST("/collections/:collection/documents", databaseHandler.DeveloperProxyHandler)
	projectDB.POST("/collections/:collection/documents/query", databaseHandler.DeveloperProxyHandler) // Query with filters

	// /api/projects/:id/database/collections/:collection/documents/:docID
	projectDB.GET("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	projectDB.PUT("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	projectDB.DELETE("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)

	// Auth Configuration routes (protected)
	projectAuth := projectsAPI.Group("/:id/auth")
	projectAuth.GET("/users", tenantAuthHandler.ListProjectUsers)
	projectAuth.GET("/config", tenantAuthHandler.GetProjectAuthConfig)
	projectAuth.PUT("/config", tenantAuthHandler.UpdateProjectAuthConfig)

	// Token management routes (cookie-authenticated)
	tokenAPI := api.Group("/auth/tokens")
	tokenAPI.Use(middleware.AuthMiddleware(authService))
	tokenAPI.GET("", tokenHandler.List)
	tokenAPI.POST("", tokenHandler.Create)
	tokenAPI.DELETE("/:id", tokenHandler.Delete)

	// Start cleanup goroutine - Not needed for Stateless Auth (bun-auth handles it)
	// But if we want local cache cleanup, we can add it later.

	// Graceful shutdown
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
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
