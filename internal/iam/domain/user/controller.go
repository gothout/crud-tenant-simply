package user

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"tenant-crud-simply/internal/iam/domain/model"
	"tenant-crud-simply/internal/iam/domain/tenant"
	"tenant-crud-simply/internal/iam/middleware"
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
	mw      middleware.Middleware
}

func NewController(service Service) Controller {
	mw := middleware.MustUse().Middleware
	return &controllerImpl{
		Service: service,
		mw:      mw,
	}
}

func (ctrl *controllerImpl) Routes(routes gin.IRouter) {
	userGroup := routes.Group("/user")

	{
		userGroup.POST("/:identifier", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin, model.RoleTenantAdmin), ctrl.Create)
		userGroup.GET("", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin, model.RoleTenantAdmin, model.RoleTenantUser), ctrl.Read)
		userGroup.GET("/:identifier", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin, model.RoleTenantAdmin, model.RoleTenantUser), ctrl.Read)
		userGroup.GET("/list", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin, model.RoleTenantAdmin), ctrl.List)
		userGroup.PATCH("/:identifier", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin, model.RoleTenantAdmin, model.RoleTenantUser), ctrl.Update)
		userGroup.DELETE("/:identifier", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin, model.RoleTenantAdmin), ctrl.Delete)
	}
}

// @Summary      Cria um novo Usuário
// @Description  Registra um novo usuário no sistema, associado a um tenant (empresa/organização).
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
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
		restError := rest_err.NewBadRequestError(nil, "tenant identifier is required in URL path")
		c.JSON(restError.Code, restError)
		return
	}

	var req CreateUserRequestDto
	if err := c.ShouldBindJSON(&req); err != nil {
		restError := rest_err.NewBadRequestError(nil, "invalid json body")
		c.JSON(restError.Code, restError)
		return
	}

	ctxIdentify, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		e := rest_err.NewForbiddenError(nil, "Usuário não autenticado.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}
	var newUser User
	switch ctxIdentify.User.Role {
	case RoleSystemAdmin:
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

	case RoleTenantAdmin:
		newUser = User{
			Tenant: tenant.Tenant{
				UUID: ctxIdentify.User.Tenant.UUID,
			},
			Name:     req.Name,
			Email:    req.Email,
			Password: req.Password,
			Role:     req.Role,
			Live:     true,
		}
		if newUser.Role != RoleTenantAdmin && newUser.Role != RoleTenantUser {
			restError := rest_err.NewBadRequestError(&ctxIdentify.Metadata.RayTraceCode,
				fmt.Sprintf("invalid user role. Valid roles are: %s, %s", RoleTenantAdmin, RoleTenantUser),
			)
			c.JSON(restError.Code, restError)
			return
		}
	default:
		e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Ação não permitida.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	if !IsValidUserRole(newUser.Role) {
		validRolesStr := strings.Join(AllValidRoles, ", ")
		restError := rest_err.NewBadRequestError(&ctxIdentify.Metadata.RayTraceCode,
			fmt.Sprintf("invalid user role. Valid roles are: %s", validRolesStr),
		)
		c.JSON(restError.Code, restError)
		return
	}

	userCreated, err := ctrl.Service.Create(c.Request.Context(), newUser)

	if err != nil {
		var restError *rest_err.RestErr
		switch {
		case errors.Is(err, tenant.ErrNotFound):
			restError = rest_err.NewNotFoundError(&ctxIdentify.Metadata.RayTraceCode, err.Error())

		case errors.Is(err, ErrEmailDuplicated):
			restError = rest_err.NewConflictValidationError(&ctxIdentify.Metadata.RayTraceCode, err.Error(), nil)

		case errors.Is(err, ErrInvalidInput):
			restError = rest_err.NewBadRequestError(&ctxIdentify.Metadata.RayTraceCode, err.Error())

		default:
			restError = rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "internal server error", nil)
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
// @Description  Busca um usuário. Se o identificador for passado na URL, busca aquele usuário específico. Se for vazio (/api/user), busca o perfil do usuário logado.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Param        identifier path      string  false  "UUID ou Email do usuário (Opcional)"
// @Success      200  {object}  UserResponseDto
// @Failure      400  {object}  rest_err.RestErr
// @Failure      403  {object}  rest_err.RestErr
// @Failure      404  {object}  rest_err.RestErr
// @Failure      500  {object}  rest_err.RestErr
// @Router       /api/user/{identifier} [get]
func (ctrl *controllerImpl) Read(c *gin.Context) {
	ctxIdentify, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		e := rest_err.NewForbiddenError(nil, "Usuário não autenticado.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	identificador := c.Param("identifier")
	if identificador == "undefined" || identificador == "null" {
		identificador = ""
	}
	userToFind := User{}

	if identificador == "" {
		// --- CASO 1: Leitura do Próprio Usuário (Self) ---
		userToFind.UUID = ctxIdentify.User.UUID
	} else {
		// --- CASO 2: Leitura de Usuário Específico ---
		if err := uuid.Validate(identificador); err == nil {
			userToFind.UUID = uuid.MustParse(identificador)
		} else {
			userToFind.Email = identificador
		}
	}

	userFound, err := ctrl.Service.Read(c.Request.Context(), userToFind)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, tenant.ErrNotFound) {
			restError := rest_err.NewNotFoundError(&ctxIdentify.Metadata.RayTraceCode, "user not found")
			c.JSON(restError.Code, restError)
			return
		}
		restError := rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "internal server error", nil)
		c.JSON(restError.Code, restError)
		return
	}

	switch ctxIdentify.User.Role {
	case model.RoleSystemAdmin:
		// SystemAdmin vê tudo. Permissão concedida.

	case model.RoleTenantAdmin:
		// TenantAdmin só vê usuários do MESMO tenant.
		if userFound.Tenant.UUID.String() != ctxIdentify.User.Tenant.UUID.String() {
			e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Você não tem permissão para visualizar usuários de outro tenant.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

	case model.RoleTenantUser:
		// TenantUser só pode ver a SI MESMO.
		// Comparamos o UUID do banco com o UUID do token.
		if userFound.UUID.String() != ctxIdentify.User.UUID.String() {
			e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Você não tem permissão para visualizar dados de outros usuários.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

	default:
		e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Ação não permitida.")
		c.AbortWithStatusJSON(e.Code, e)
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
// @Description  Retorna uma lista paginada de usuários.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Param        page              query     int     false  "Número da página (padrão 1)"
// @Param        size              query     int     false  "Tamanho da página (padrão 10)"
// @Param        tenant_identifier query     string  false  "Filtro opcional: UUID ou Documento do Tenant (Apenas para SystemAdmin)"
// @Success      200  {array}   UserResponseDto
// @Failure      500  {object}  rest_err.RestErr
// @Router       /api/user/list [get]
func (ctrl *controllerImpl) List(c *gin.Context) {
	var req ListUserRequestDto
	if err := c.ShouldBindQuery(&req); err != nil {
		restError := rest_err.NewBadRequestError(nil, "invalid query parameters")
		c.JSON(restError.Code, restError)
		return
	}

	ctxIdentify, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		e := rest_err.NewForbiddenError(nil, "Usuário não autenticado.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	var users []User
	var err error

	switch ctxIdentify.User.Role {
	case model.RoleSystemAdmin:
		if req.TenantIdentifier != "" {
			t := tenant.Tenant{}
			if errUuid := uuid.Validate(req.TenantIdentifier); errUuid == nil {
				t.UUID = uuid.MustParse(req.TenantIdentifier)
			} else {
				t.Document = req.TenantIdentifier
			}
			users, err = ctrl.Service.ListByTenant(c, t, req.Page, req.PageSize)
		} else {
			users, err = ctrl.Service.List(c, req.Page, req.PageSize)
		}

	case model.RoleTenantAdmin:
		users, err = ctrl.Service.ListByTenant(c, ctxIdentify.User.Tenant, req.Page, req.PageSize)

	default:
		e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Ação não permitida.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	if err != nil {
		if errors.Is(err, tenant.ErrNotFound) {
			restError := rest_err.NewNotFoundError(&ctxIdentify.Metadata.RayTraceCode, "tenant not found")
			c.JSON(restError.Code, restError)
			return
		}

		restError := rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "internal server error", nil)
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

	if response == nil {
		response = []UserResponseDto{}
	}

	c.JSON(http.StatusOK, response)
}

// @Summary      Atualiza um Usuário
// @Description  Atualiza dados de um usuário existente. O usuário a ser atualizado é identificado pelo UUID/Email no path.
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
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

	var req UpdateUserRequestDto
	if err := c.ShouldBindJSON(&req); err != nil {
		restError := rest_err.NewBadRequestError(nil, "invalid json body")
		c.JSON(restError.Code, restError)
		return
	}

	userToFind := User{}
	if err := uuid.Validate(identificador); err == nil {
		userToFind.UUID = uuid.MustParse(identificador)
	} else {
		userToFind.Email = identificador
	}

	ctxIdentify, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		e := rest_err.NewForbiddenError(nil, "Usuário não autenticado.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}
	targetUser, err := ctrl.Service.Read(c.Request.Context(), userToFind)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, tenant.ErrNotFound) {
			restError := rest_err.NewNotFoundError(&ctxIdentify.Metadata.RayTraceCode, "user not found")
			c.JSON(restError.Code, restError)
			return
		}
		restError := rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "internal server error", nil)
		c.JSON(restError.Code, restError)
		return
	}

	userToUpdate := User{
		UUID: targetUser.UUID,
		Tenant: model.Tenant{
			UUID: targetUser.Tenant.UUID,
		},
	}

	switch ctxIdentify.User.Role {
	case model.RoleSystemAdmin:
		userToUpdate.Role = req.Role

	case model.RoleTenantAdmin:
		if targetUser.TenantUUID.String() != ctxIdentify.User.Tenant.UUID.String() {
			e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Você não tem permissão para alterar usuários de outro tenant.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}
		if req.Role != "" {
			if req.Role == model.RoleSystemAdmin {
				e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Tenant Admin não pode atribuir permissão de System Admin.")
				c.AbortWithStatusJSON(e.Code, e)
				return
			}
			if req.Role != model.RoleTenantAdmin && req.Role != model.RoleTenantUser {
				e := rest_err.NewBadRequestError(&ctxIdentify.Metadata.RayTraceCode, "Role inválida para Tenant Admin.")
				c.JSON(e.Code, e)
				return
			}
		}
		userToUpdate.Role = req.Role

	case model.RoleTenantUser:
		if targetUser.UUID != ctxIdentify.User.UUID {
			e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Você só pode alterar seus próprios dados.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}
		userToUpdate.Role = ""

	default:
		e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Ação não permitida.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	userToUpdate.Name = req.Name
	userToUpdate.Email = req.Email
	userToUpdate.Password = req.Password

	if userToUpdate.Role != "" {
		if !IsValidUserRole(userToUpdate.Role) {
			validRolesStr := strings.Join(AllValidRoles, ", ")
			restError := rest_err.NewBadRequestError(&ctxIdentify.Metadata.RayTraceCode,
				fmt.Sprintf("invalid user role. Valid roles are: %s", validRolesStr),
			)
			c.JSON(restError.Code, restError)
			return
		}
	}

	updatedUser, err := ctrl.Service.Update(c.Request.Context(), userToUpdate)
	if err != nil {
		var restError *rest_err.RestErr
		switch {
		case errors.Is(err, ErrNotFound):
			restError = rest_err.NewNotFoundError(&ctxIdentify.Metadata.RayTraceCode, "user not found")
		case errors.Is(err, ErrEmailDuplicated):
			restError = rest_err.NewConflictValidationError(&ctxIdentify.Metadata.RayTraceCode, err.Error(), nil)
		default:
			restError = rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "internal server error", nil)
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
// @Description  Exclui permanentemente um usuário no sistema usando o UUID ou o Email passado na URL.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Param        identifier path      string  true  "UUID ou Email do usuário a ser deletado"
// @Success      204  {object}  nil
// @Failure      400  {object}  rest_err.RestErr
// @Failure      403  {object}  rest_err.RestErr
// @Failure      404  {object}  rest_err.RestErr
// @Failure      500  {object}  rest_err.RestErr
// @Router       /api/user/{identifier} [delete]
func (ctrl *controllerImpl) Delete(c *gin.Context) {
	identificador := c.Param("identifier")
	if identificador == "" {
		restError := rest_err.NewBadRequestError(nil, "identifier parameter is required")
		c.JSON(restError.Code, restError)
		return
	}

	// 1. Determina se é UUID ou Email e prepara busca
	userToFind := User{}
	if err := uuid.Validate(identificador); err == nil {
		userToFind.UUID = uuid.MustParse(identificador)
	} else {
		userToFind.Email = identificador
	}
	// 3. Autenticação e Autorização
	ctxIdentify, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		e := rest_err.NewForbiddenError(nil, "Usuário não autenticado.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}
	// 2. Busca usuário ALVO no banco (Segurança: precisamos saber o Tenant dele)
	targetUser, err := ctrl.Service.Read(c.Request.Context(), userToFind)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, tenant.ErrNotFound) {
			restError := rest_err.NewNotFoundError(&ctxIdentify.Metadata.RayTraceCode, "user not found")
			c.JSON(restError.Code, restError)
			return
		}
		restError := rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "internal server error", nil)
		c.JSON(restError.Code, restError)
		return
	}

	switch ctxIdentify.User.Role {
	case model.RoleSystemAdmin:
		// SystemAdmin deleta qualquer um.

	case model.RoleTenantAdmin:
		// TenantAdmin só deleta do MESMO tenant
		if targetUser.Tenant.UUID.String() != ctxIdentify.User.Tenant.UUID.String() {
			e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Você não tem permissão para deletar usuários de outro tenant.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

	case model.RoleTenantUser:
		// TenantUser só deleta a SI MESMO
		if targetUser.UUID.String() != ctxIdentify.User.UUID.String() {
			e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Você não tem permissão para deletar outros usuários.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

	default:
		e := rest_err.NewForbiddenError(&ctxIdentify.Metadata.RayTraceCode, "Ação não permitida.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	// 4. Executa Delete
	err = ctrl.Service.Delete(c.Request.Context(), targetUser)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			restError := rest_err.NewNotFoundError(&ctxIdentify.Metadata.RayTraceCode, "user not found")
			c.JSON(restError.Code, restError)
			return
		}
		restError := rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "internal server error", nil)
		c.JSON(restError.Code, restError)
		return
	}

	c.Status(http.StatusNoContent)
}
