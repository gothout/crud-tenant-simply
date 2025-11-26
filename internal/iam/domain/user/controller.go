package user

import (
	"errors"
	"net/http"
	"tenant-crud-simply/internal/iam/domain/tenant"
	"tenant-crud-simply/internal/pkg/rest_err"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller interface {
	Routes(routes gin.IRouter)
	Create(c *gin.Context)
}

type controllerImpl struct {
	Service Service
}

func NewController(service Service) Controller {
	return &controllerImpl{
		Service: service,
	}
}

func (ctrl *controllerImpl) Routes(routes gin.IRouter) {
	userGroup := routes.Group("/user")
	{
		userGroup.POST("/:identifier", ctrl.Create)
	}
}

// @Summary      Cria um novo Usuário
// @Description  Registra um novo usuário no sistema, associado a um tenant (empresa/organização).
// @Tags         User
// @Accept       json
// @Produce      json
//
// @Param        identifier path string true "Identificador (UUID ou Documento) do Tenant ao qual o usuário será associado."
// @Param        request body CreateUserRequestDto true "Objeto do usuário que precisa ser criado."
//
// @Success      201  {object}  UserResponseDto  "Usuário criado com sucesso."
// @Failure      400  {object}  rest_err.RestErr    "Requisição inválida (corpo JSON mal formatado, dados de entrada inválidos, ou 'identifier' do tenant ausente)."
// @Failure      404  {object}  rest_err.RestErr    "Tenant não encontrado (o 'identifier' fornecido não corresponde a nenhum tenant existente)."
// @Failure      409  {object}  rest_err.RestErr    "Conflito (o 'email' fornecido já está em uso)."
// @Failure      500  {object}  rest_err.RestErr    "Erro interno do servidor."
//
// @Router       /api/user/{identifier} [post]
func (ctrl *controllerImpl) Create(c *gin.Context) {
	tenantIdentifier := c.Param("identifier")
	if tenantIdentifier == "" {
		restError := rest_err.NewBadRequestError("tenant identifier is required in URL path")
		c.JSON(restError.Code, restError)
		return
	}

	var req CreateUserRequestDto
	if err := c.ShouldBindJSON(&req); err != nil {
		restError := rest_err.NewBadRequestError("invalid json body")
		c.JSON(restError.Code, restError)
		return
	}

	var newUser User
	if err := uuid.Validate(tenantIdentifier); err != nil {
		newUser = User{
			Tenant: tenant.Tenant{
				Document: tenantIdentifier,
			},
			Name:     req.Name,
			Email:    req.Email,
			Password: req.Password,
			Role:     req.Role,
			Live:     true,
		}
	} else {
		newUser = User{
			Tenant: tenant.Tenant{
				UUID: uuid.MustParse(tenantIdentifier),
			},
			Name:     req.Name,
			Email:    req.Email,
			Password: req.Password,
			Role:     req.Role,
			Live:     true,
		}
	}

	userCreated, err := ctrl.Service.Create(c.Request.Context(), newUser)

	if err != nil {
		var restError *rest_err.RestErr
		switch {
		case errors.Is(err, tenant.ErrNotFound):
			restError = rest_err.NewNotFoundError(err.Error())

		case errors.Is(err, ErrEmailDuplicated):
			restError = rest_err.NewConflictValidationError(err.Error(), nil)

		case errors.Is(err, ErrInvalidInput):
			restError = rest_err.NewBadRequestError(err.Error())

		default:
			restError = rest_err.NewInternalServerError("internal server error", nil)
		}

		c.JSON(restError.Code, restError)
		return
	}

	response := UserResponseDto{
		UUID:       userCreated.UUID,
		TenantUUID: userCreated.TenantUUID,
		Name:       userCreated.Name,
		Email:      userCreated.Email,
		Role:       userCreated.Role,
		Live:       userCreated.Live,
		CreateAt:   userCreated.CreateAt,
		UpdateAt:   userCreated.UpdateAt,
	}
	c.JSON(http.StatusCreated, response)
}
