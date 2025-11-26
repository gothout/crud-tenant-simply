package tenant

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Controller interface define os métodos do controller de tenant
type Controller interface {
	Routes(routes gin.IRouter)
	Create(c *gin.Context)
}

// controllerImpl implementa o Controller
type controllerImpl struct {
	service Service
}

// NewController cria uma nova instância do controller
func NewController(service Service) Controller {
	return &controllerImpl{
		service: service,
	}
}

// Routes registra as rotas do tenant
func (ctrl *controllerImpl) Routes(routes gin.IRouter) {
	tenantGroup := routes.Group("/tenant")
	{
		tenantGroup.POST("/create", ctrl.Create)
	}
}

// Create cria um novo tenant
// @Summary Cria um novo tenant
// @Description Cria um novo tenant no sistema
// @Tags tenant
// @Accept json
// @Produce json
// @Param request body CreateTenantRequest true "Dados do tenant"
// @Success 201 {object} TenantResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/tenant/create [post]
func (ctrl *controllerImpl) Create(c *gin.Context) {
	var req CreateTenantRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// Cria o modelo Tenant
	tenant := Tenant{
		Name:     req.Name,
		Document: req.Document,
		Live:     true,
		CreateAt: time.Now().UTC(),
		UpdateAt: time.Now().UTC(),
	}

	// Chama o serviço para criar
	created, err := ctrl.service.Create(c.Request.Context(), tenant)
	if err != nil {
		switch err {
		case ErrDocumentDuplicated:
			c.JSON(http.StatusConflict, gin.H{
				"error": "document already exists",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to create tenant",
				"details": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, ToResponse(created))
}
