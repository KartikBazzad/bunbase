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

	"github.com/kartikbazzad/bunbase/buncast/pkg/client"
	fnclient "github.com/kartikbazzad/bunbase/functions/pkg/client"
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
	if val := os.Getenv("PLATFORM_BUNCAST_SOCKET"); val != "" {
		*buncastSocket = val
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
	// Use RPC client for bundoc when configured (faster); otherwise HTTP client
	var bundocProxy bundoc.Proxy
	if bundocRPCAddr := os.Getenv("PLATFORM_BUNDOC_RPC_ADDR"); bundocRPCAddr != "" {
		bundocProxy = bundoc.NewRPCClient(bundocRPCAddr)
		log.Printf("Bundoc: using RPC client (addr=%s)", bundocRPCAddr)
	} else {
		bundocProxy = bundoc.NewClient(cfg.Bundoc.URL)
	}

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

	// Initialize Buncast client for realtime subscriptions (optional)
	var buncastClient *client.Client
	if *buncastSocket != "" {
		buncastClient = client.New(*buncastSocket)
		log.Printf("Realtime: Buncast client enabled (socket=%s)", *buncastSocket)
	} else {
		log.Printf("Realtime: Buncast client disabled (no -buncast-socket)")
	}

	subscriptionManager := services.NewSubscriptionManager(buncastClient)
	if buncastClient != nil && *buncastSocket != "" {
		subscriptionManager.SetSocketPath(*buncastSocket)
	}

	tokenService := services.NewTokenService(db.Pool)
	projectConfigService := services.NewProjectConfigService(*gatewayURL)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	tenantAuthHandler := handlers.NewTenantAuthHandler(tenantAuthClient, projectService)
	projectHandler := handlers.NewProjectHandler(projectService)
	projectConfigHandler := handlers.NewProjectConfigHandler(projectService, projectConfigService)
	var functionsRPC *fnclient.Client
	if rpcAddr := os.Getenv("PLATFORM_FUNCTIONS_RPC_ADDR"); rpcAddr != "" {
		functionsRPC = fnclient.New("tcp://" + rpcAddr)
		log.Printf("Functions: using RPC client (addr=%s)", rpcAddr)
	}
	functionHandler := handlers.NewFunctionHandler(functionService, projectService, projectConfigService, *functionsURL, functionsRPC)
	tokenHandler := handlers.NewTokenHandler(tokenService)
	databaseHandler := handlers.NewDatabaseHandler(bundocProxy, projectService, subscriptionManager)

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

	// Key-scoped routes: API key only, no project ID in path. Project inferred from key. Register first.
	v1Key := v1.Group("")
	v1Key.Use(middleware.RequireProjectKeyMiddleware(projectService))
	v1Key.GET("/project", projectConfigHandler.GetCurrentProject)
	v1KeyDB := v1Key.Group("/database")
	v1KeyDB.GET("/collections", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.POST("/collections", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.GET("/collections/:collection", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.PATCH("/collections/:collection", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.PATCH("/collections/:collection/rules", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.DELETE("/collections/:collection", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.GET("/collections/:collection/indexes", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.POST("/collections/:collection/indexes", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.DELETE("/collections/:collection/indexes/:field", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.GET("/collections/:collection/documents", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.POST("/collections/:collection/documents", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.POST("/collections/:collection/documents/query", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.GET("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.PUT("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.PATCH("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.DELETE("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	v1KeyDB.GET("/collections/:collection/subscribe", databaseHandler.HandleCollectionSubscribe)
	v1KeyDB.POST("/collections/:collection/documents/query/subscribe", databaseHandler.HandleQuerySubscribe)
	v1Key.POST("/functions/:name/invoke", functionHandler.InvokeProjectFunction)
	v1Key.GET("/functions/:name/invoke", functionHandler.InvokeProjectFunction)
	v1Key.GET("/functions", functionHandler.ListFunctions)
	v1Key.GET("/auth/users", tenantAuthHandler.ListProjectUsers)
	v1Key.POST("/auth/users", tenantAuthHandler.CreateProjectUser)
	v1Key.GET("/auth/config", tenantAuthHandler.GetProjectAuthConfig)
	v1Key.PUT("/auth/config", tenantAuthHandler.UpdateProjectAuthConfig)

	// Also alias Project routes to /v1/projects for Web Console.
	// Routes that accept project API key (database, function invoke) use ProjectKeyOrUserAuthMiddleware and are registered first.
	v1ProjectsKeyOrUser := v1.Group("/projects")
	v1ProjectsKeyOrUser.Use(middleware.ProjectKeyOrUserAuthMiddleware(authService, tokenService, projectService))
	v1ProjectDBKeyOrUser := v1ProjectsKeyOrUser.Group("/:id/database")
	v1ProjectDBKeyOrUser.GET("/collections", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.POST("/collections", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.GET("/collections/:collection", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.PATCH("/collections/:collection", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.PATCH("/collections/:collection/rules", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.DELETE("/collections/:collection", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.GET("/collections/:collection/indexes", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.POST("/collections/:collection/indexes", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.DELETE("/collections/:collection/indexes/:field", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.GET("/collections/:collection/documents", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.POST("/collections/:collection/documents", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.POST("/collections/:collection/documents/query", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.GET("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.PUT("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.PATCH("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	v1ProjectDBKeyOrUser.DELETE("/collections/:collection/documents/:docID", databaseHandler.DeveloperProxyHandler)
	// Realtime subscriptions (SSE) - also available via v1 API
	v1ProjectDBKeyOrUser.GET("/collections/:collection/subscribe", databaseHandler.HandleCollectionSubscribe)
	v1ProjectDBKeyOrUser.POST("/collections/:collection/documents/query/subscribe", databaseHandler.HandleQuerySubscribe)
	v1ProjectsKeyOrUser.POST("/:id/functions/:name/invoke", functionHandler.InvokeProjectFunction)
	v1ProjectsKeyOrUser.GET("/:id/functions/:name/invoke", functionHandler.InvokeProjectFunction)

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
	v1Projects.POST("/:id/regenerate-api-key", projectHandler.RegenerateProjectAPIKey)
	v1Projects.GET("/:id/functions", functionHandler.ListFunctions)
	v1Projects.GET("/:id/functions/logs", functionHandler.GetProjectFunctionLogs)
	v1Projects.POST("/:id/functions", functionHandler.DeployFunction)
	v1Projects.DELETE("/:id/functions/:functionId", functionHandler.DeleteFunction)

	// Tenant auth routes (v1 - for Web Console)
	v1ProjectAuth := v1Projects.Group("/:id/auth")
	v1ProjectAuth.GET("/users", tenantAuthHandler.ListProjectUsers)
	v1ProjectAuth.POST("/users", tenantAuthHandler.CreateProjectUser)
	v1ProjectAuth.GET("/config", tenantAuthHandler.GetProjectAuthConfig)
	v1ProjectAuth.PUT("/config", tenantAuthHandler.UpdateProjectAuthConfig)

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
	projectsAPI.POST("/:id/regenerate-api-key", projectHandler.RegenerateProjectAPIKey)

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
	projectDB.GET("/collections/:collection", databaseHandler.DeveloperProxyHandler)   // Get collection (schema, rules)
	projectDB.PATCH("/collections/:collection", databaseHandler.DeveloperProxyHandler) // Schema Update
	projectDB.PATCH("/collections/:collection/rules", databaseHandler.DeveloperProxyHandler)
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

	// Realtime subscriptions (SSE)
	projectDB.GET("/collections/:collection/subscribe", databaseHandler.HandleCollectionSubscribe)
	projectDB.POST("/collections/:collection/documents/query/subscribe", databaseHandler.HandleQuerySubscribe)

	// Auth Configuration routes (protected)
	projectAuth := projectsAPI.Group("/:id/auth")
	projectAuth.GET("/users", tenantAuthHandler.ListProjectUsers)
	projectAuth.POST("/users", tenantAuthHandler.CreateProjectUser)
	projectAuth.GET("/config", tenantAuthHandler.GetProjectAuthConfig)
	projectAuth.PUT("/config", tenantAuthHandler.UpdateProjectAuthConfig)

	// Token management routes (cookie-authenticated)
	tokenAPI := api.Group("/auth/tokens")
	tokenAPI.Use(middleware.AuthMiddleware(authService))
	tokenAPI.GET("", tokenHandler.List)
	tokenAPI.POST("", tokenHandler.Create)
	tokenAPI.DELETE("/:id", tokenHandler.Delete)

	// Catch-all route for custom function domains routed from Traefik.
	// This must be registered last so that it only handles requests that
	// weren't matched by the more specific API/web routes above.
	router.Any("/*path", functionHandler.HandleCustomDomainInvoke)

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
