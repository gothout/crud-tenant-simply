package routes

import (
	"fmt"
	"os"
	"tenant-crud-simply/internal/iam/application/auth"
	"tenant-crud-simply/internal/iam/domain/tenant"
	"tenant-crud-simply/internal/iam/domain/user"
	"tenant-crud-simply/internal/iam/middleware"
	"tenant-crud-simply/internal/web/handler"
	webMiddleware "tenant-crud-simply/internal/web/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "tenant-crud-simply/docs"
)

func SetupRouter() *gin.Engine {
	env := viper.GetString("app.env")
	// 1. Configuração do modo Gin
	switch env {
	case "dev":
		gin.SetMode(gin.DebugMode)
	case "prod":
		gin.SetMode(gin.ReleaseMode)
	case "":
		fmt.Println("WARNING: 'app.env' not set in config. Defaulting to 'dev' mode.")
		gin.SetMode(gin.DebugMode)
	default:
		fmt.Printf("ERROR: Invalid environment value '%s'. Must be 'dev' or 'prod'.\n", env)
		os.Exit(1)
	}

	r := gin.Default()

	// Acessível em /doc/index.html
	r.GET("/doc/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Setup API routes
	SetupApiRoutes(r)

	// Setup Web routes
	SetupWebRoutes(r)

	return r
}

func SetupApiRoutes(r *gin.Engine) {
	route := r.Group("/api")

	tenantController, err := tenant.Use()
	if err != nil {
		panic(err)
	}
	userController, err := user.Use()
	if err != nil {
		panic(err)
	}
	authController, err := auth.Use()
	if err != nil {
		panic(err)
	}
	tenantController.Routes(route, middleware.MustUse().Middleware.SetContextAutorization())
	userController.Routes(route)
	authController.Routes(route)
}

func SetupWebRoutes(r *gin.Engine) {
	// Cria session store
	sessionSecret := viper.GetString("security.session_secret")
	if sessionSecret == "" {
		sessionSecret = "default-secret-key-change-this-in-production"
	}
	sessionStore := sessions.NewCookieStore([]byte(sessionSecret))

	// Cria web handler
	webHandler, err := handler.NewWebHandler(sessionStore)
	if err != nil {
		panic(fmt.Errorf("failed to create web handler: %w", err))
	}

	// Serve arquivos estáticos
	r.Static("/assets", "./internal/web/assets")

	// Rotas públicas
	r.GET("/", webHandler.ServeLogin)
	r.GET("/login", webHandler.ServeLogin)
	r.POST("/login", webHandler.HandleLogin)

	// Rotas protegidas (requerem autenticação)
	protected := r.Group("/")
	protected.Use(webMiddleware.RequireAuth(sessionStore))
	{
		protected.GET("/dashboard", webHandler.ServeDashboard)
		protected.GET("/tenants", webHandler.ServeTenants)
		protected.GET("/users", webHandler.ServeUsers)
		protected.POST("/logout", webHandler.HandleLogout)

		// API proxy para HTMX
		apiWeb := protected.Group("/api/web")
		{
			// Proxy genérico para todas as rotas da API
			apiWeb.Any("/*path", webHandler.ProxyAPI)
		}
	}
}
