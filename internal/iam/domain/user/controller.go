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
	Read(c *gin.Context)
	List(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
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
		userGroup.GET("", ctrl.Read)
		userGroup.GET("/list", ctrl.List)
		userGroup.PATCH("/:identifier", ctrl.Update)
		userGroup.DELETE("", ctrl.Delete)
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

// @Summary      Busca um Usuário
// @Description  Busca um usuário no sistema usando o UUID ou o Email. Pelo menos um dos dois campos deve ser fornecido.
// @Tags         User
// @Produce      json
// @Param        uuid   query     string  false  "UUID do usuário"
// @Param        email  query     string  false  "Email do usuário"
// @Success      200  {object}  UserResponseDto
// @Failure      400  {object}  rest_err.RestErr
// @Failure      404  {object}  rest_err.RestErr
// @Failure      500  {object}  rest_err.RestErr
// @Router       /api/user [get]
func (ctrl *controllerImpl) Read(c *gin.Context) {
	var req ReadUserRequestDto
	if err := c.ShouldBindQuery(&req); err != nil {
		restError := rest_err.NewBadRequestError("invalid query parameters")
		c.JSON(restError.Code, restError)
		return
	}

	if req.UUID == "" && req.Email == "" {
		restError := rest_err.NewBadRequestError("uuid or email is required")
		c.JSON(restError.Code, restError)
		return
	}

	userToFind := User{}
	if req.UUID != "" {
		if err := uuid.Validate(req.UUID); err != nil {
			restError := rest_err.NewBadRequestError("invalid uuid")
			c.JSON(restError.Code, restError)
			return
		}
		userToFind.UUID = uuid.MustParse(req.UUID)
	} else {
		userToFind.Email = req.Email
	}

	userFound, err := ctrl.Service.Read(c.Request.Context(), userToFind)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			restError := rest_err.NewNotFoundError("user not found")
			c.JSON(restError.Code, restError)
			return
		}
		restError := rest_err.NewInternalServerError("internal server error", nil)
		c.JSON(restError.Code, restError)
		return
	}

	response := UserResponseDto{
		UUID:       userFound.UUID,
		TenantUUID: userFound.TenantUUID,
		Name:       userFound.Name,
		Email:      userFound.Email,
		Role:       userFound.Role,
		Live:       userFound.Live,
		CreateAt:   userFound.CreateAt,
		UpdateAt:   userFound.UpdateAt,
	}
	c.JSON(http.StatusOK, response)
}

// @Summary      Lista Usuários
// @Description  Retorna uma lista paginada de todos os usuários registrados no sistema.
// @Tags         User
// @Produce      json
// @Param        page  query     int     false  "Número da página (padrão 1)"
// @Param        size  query     int     false  "Tamanho da página (padrão 10)"
// @Success      200  {array}   UserResponseDto
// @Failure      500  {object}  rest_err.RestErr
// @Router       /api/user/list [get]
func (ctrl *controllerImpl) List(c *gin.Context) {
	var req ListUserRequestDto
	if err := c.ShouldBindQuery(&req); err != nil {
		restError := rest_err.NewBadRequestError("invalid query parameters")
		c.JSON(restError.Code, restError)
		return
	}

	users, err := ctrl.Service.List(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		restError := rest_err.NewInternalServerError("internal server error", nil)
		c.JSON(restError.Code, restError)
		return
	}

	var response []UserResponseDto
	for _, u := range users {
		response = append(response, UserResponseDto{
			UUID:       u.UUID,
			TenantUUID: u.TenantUUID,
			Name:       u.Name,
			Email:      u.Email,
			Role:       u.Role,
			Live:       u.Live,
			CreateAt:   u.CreateAt,
			UpdateAt:   u.UpdateAt,
		})
	}
	c.JSON(http.StatusOK, response)
}

// @Summary      Atualiza um Usuário
// @Description  Atualiza dados de um usuário existente. O usuário a ser atualizado é identificado pelo UUID no path.
// @Tags         User
// @Accept       json
// @Produce      json
// @Param        identifier path      string                true  "Idenficador do usuário"
// @Param        request    body      UpdateUserRequestDto  true  "Dados para atualização"
// @Success      200  {object}  UserResponseDto
// @Failure      400  {object}  rest_err.RestErr
// @Failure      404  {object}  rest_err.RestErr
// @Failure      409  {object}  rest_err.RestErr
// @Failure      500  {object}  rest_err.RestErr
// @Router       /api/user/{identifier} [patch]
func (ctrl *controllerImpl) Update(c *gin.Context) {
	identificador := c.Param("identifier")
	var userUUID uuid.UUID

	if err := uuid.Validate(identificador); err != nil {
		rUser, err := ctrl.Service.Read(c.Request.Context(), User{Email: identificador})
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				restError := rest_err.NewNotFoundError("user not found")
				c.JSON(restError.Code, restError)
				return
			}
			restError := rest_err.NewInternalServerError("internal server error", nil)
			c.JSON(restError.Code, restError)
			return
		}
		userUUID = rUser.UUID
	} else {
		userUUID = uuid.MustParse(identificador)
		_, err := ctrl.Service.Read(c.Request.Context(), User{UUID: userUUID})
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				restError := rest_err.NewNotFoundError("user not found")
				c.JSON(restError.Code, restError)
				return
			}
			restError := rest_err.NewInternalServerError("internal server error", nil)
			c.JSON(restError.Code, restError)
			return
		}
	}

	var req UpdateUserRequestDto
	if err := c.ShouldBindJSON(&req); err != nil {
		restError := rest_err.NewBadRequestError("invalid json body")
		c.JSON(restError.Code, restError)
		return
	}

	userToUpdate := User{
		UUID:     userUUID,
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	}

	updatedUser, err := ctrl.Service.Update(c.Request.Context(), userToUpdate)
	if err != nil {
		var restError *rest_err.RestErr
		switch {
		case errors.Is(err, ErrNotFound):
			restError = rest_err.NewNotFoundError("user not found")
		case errors.Is(err, ErrEmailDuplicated):
			restError = rest_err.NewConflictValidationError(err.Error(), nil)
		default:
			restError = rest_err.NewInternalServerError("internal server error", nil)
		}
		c.JSON(restError.Code, restError)
		return
	}

	response := UserResponseDto{
		UUID:       updatedUser.UUID,
		TenantUUID: updatedUser.TenantUUID,
		Name:       updatedUser.Name,
		Email:      updatedUser.Email,
		Role:       updatedUser.Role,
		Live:       updatedUser.Live,
		CreateAt:   updatedUser.CreateAt,
		UpdateAt:   updatedUser.UpdateAt,
	}
	c.JSON(http.StatusOK, response)
}

// @Summary      Deleta um Usuário
// @Description  Exclui permanentemente um usuário no sistema usando o UUID ou o Email.
// @Tags         User
// @Produce      json
// @Param        uuid   query     string  false  "UUID do usuário"
// @Param        email  query     string  false  "Email do usuário"
// @Success      204  {object}  nil
// @Failure      400  {object}  rest_err.RestErr
// @Failure      404  {object}  rest_err.RestErr
// @Failure      500  {object}  rest_err.RestErr
// @Router       /api/user [delete]
func (ctrl *controllerImpl) Delete(c *gin.Context) {
	var req ReadUserRequestDto
	if err := c.ShouldBindQuery(&req); err != nil {
		restError := rest_err.NewBadRequestError("invalid query parameters")
		c.JSON(restError.Code, restError)
		return
	}

	if req.UUID == "" && req.Email == "" {
		restError := rest_err.NewBadRequestError("uuid or email is required")
		c.JSON(restError.Code, restError)
		return
	}

	userToDelete := User{}
	if req.UUID != "" {
		if err := uuid.Validate(req.UUID); err != nil {
			restError := rest_err.NewBadRequestError("invalid uuid")
			c.JSON(restError.Code, restError)
			return
		}
		userToDelete.UUID = uuid.MustParse(req.UUID)
	} else {
		userToDelete.Email = req.Email
	}

	// First find the user to ensure we have the UUID if only email was provided,
	// because Repository.Delete might rely on UUID or we want to ensure existence.
	// Actually Service.Delete calls Repository.Delete which uses UUID.
	// If we only have Email, we must Read first to get UUID.

	if userToDelete.UUID == uuid.Nil {
		found, err := ctrl.Service.Read(c.Request.Context(), userToDelete)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				restError := rest_err.NewNotFoundError("user not found")
				c.JSON(restError.Code, restError)
				return
			}
			restError := rest_err.NewInternalServerError("internal server error", nil)
			c.JSON(restError.Code, restError)
			return
		}
		userToDelete = found
	}

	err := ctrl.Service.Delete(c.Request.Context(), userToDelete)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			restError := rest_err.NewNotFoundError("user not found")
			c.JSON(restError.Code, restError)
			return
		}
		restError := rest_err.NewInternalServerError("internal server error", nil)
		c.JSON(restError.Code, restError)
		return
	}

	c.Status(http.StatusNoContent)
}
