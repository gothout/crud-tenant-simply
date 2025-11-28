package routes

import (
	"fmt"
	"os"
	"tenant-crud-simply/internal/iam/application/auth"
	"tenant-crud-simply/internal/iam/domain/tenant"
	"tenant-crud-simply/internal/iam/domain/user"
	"tenant-crud-simply/internal/iam/middleware"

	"github.com/gin-gonic/gin"
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
	SetupApiRoutes(r)
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
