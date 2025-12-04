package tenant

import (
	"net/http"
	"tenant-crud-simply/internal/iam/domain/model"
	"tenant-crud-simply/internal/iam/middleware"
	"tenant-crud-simply/internal/pkg/log/auditoria_log"
	"tenant-crud-simply/internal/pkg/rest_err"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Controller interface define os métodos do controller de tenant
type Controller interface {
	Routes(routes gin.IRouter)
	Create(c *gin.Context)
	Read(c *gin.Context)
	List(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

// controllerImpl implementa o Controller
type controllerImpl struct {
	service Service
	mw      middleware.Middleware
}

// NewController cria uma nova instância do controller
func NewController(service Service) Controller {
	mw := middleware.MustUse().Middleware

	return &controllerImpl{
		service: service,
		mw:      mw,
	}
}

func (ctrl *controllerImpl) logAudit(c *gin.Context, login *middleware.Login, action, function string, success bool, input, output interface{}) {
	var (
		tenantUUID *uuid.UUID
		userUUID   *uuid.UUID
		identifier string
		rayTrace   string
	)

	if login != nil {
		tenantUUID = login.User.TenantUUID
		if login.User.UUID != uuid.Nil {
			userUUID = &login.User.UUID
		}
		identifier = login.User.Email
		rayTrace = login.Metadata.RayTraceCode
	}

	auditoria_log.LogAsync(c.Request.Context(), auditoria_log.AuditLog{
		TenantUUID:   tenantUUID,
		UserUUID:     userUUID,
		Identifier:   identifier,
		RayTraceCode: rayTrace,
		Domain:       "tenant",
		Action:       action,
		Function:     function,
		Success:      success,
		InputData:    auditoria_log.SerializeData(input),
		OutputData:   auditoria_log.SerializeData(output),
	})
}

// Routes registra as rotas do tenant
func (ctrl *controllerImpl) Routes(routes gin.IRouter) {
	tenantGroup := routes.Group("/tenant")

	{
		// Rota protegida com autenticação e autorização de role
		tenantGroup.POST("/create", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin), ctrl.Create)
		tenantGroup.GET("", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin, model.RoleTenantAdmin), ctrl.Read)
		tenantGroup.GET("/list", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin), ctrl.List)
		tenantGroup.PATCH("/:uuid", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin, model.RoleTenantAdmin), ctrl.Update)
		tenantGroup.DELETE("", ctrl.mw.SetContextAutorization(), ctrl.mw.AuthorizeRole(model.RoleSystemAdmin), ctrl.Delete)
	}
}

// Create cria um novo tenant
// @Summary Cria um novo tenant
// @Description Cria um novo tenant no sistema
// @Tags Tenant
// @Accept json
// @Produce json
// @Security     BearerAuth
// @Param request body CreateTenantRequestDto true "Dados do tenant"
// @Success 201 {object} TenantResponseDto
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/tenant/create [post]
func (ctrl *controllerImpl) Create(c *gin.Context) {
	var req CreateTenantRequestDto

	ctxIdentify, _ := middleware.GetAuthenticatedUser(c)

	if err := c.ShouldBindJSON(&req); err != nil {
		response := gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		}
		c.JSON(http.StatusBadRequest, response)
		ctrl.logAudit(c, ctxIdentify, "create", "Create", false, req, response)
		return
	}

	// Cria o modelo Tenant
	tenant := model.Tenant{
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
			response := gin.H{
				"error": "document already exists",
			}
			c.JSON(http.StatusConflict, response)
			ctrl.logAudit(c, ctxIdentify, "create", "Create", false, req, response)
		default:
			response := gin.H{
				"error":   "failed to create tenant",
				"details": err.Error(),
			}
			c.JSON(http.StatusInternalServerError, response)
			ctrl.logAudit(c, ctxIdentify, "create", "Create", false, req, response)
		}
		return
	}

	resp := &TenantResponseDto{
		UUID:     created.UUID,
		Name:     created.Name,
		Document: created.Document,
		Live:     created.Live,
		CreateAt: created.CreateAt,
		UpdateAt: created.UpdateAt,
	}
	c.JSON(http.StatusCreated, resp)
	ctrl.logAudit(c, ctxIdentify, "create", "Create", true, req, resp)
}

// @Summary      Busca um Tenant
// @Description  Busca um tenant no sistema usando o UUID ou o Documento (CNPJ/CPF). Pelo menos um dos dois campos deve ser fornecido.
// @Tags         Tenant
// @Produce      json
// @Security     BearerAuth
//
// @Param        uuid query string false "UUID do tenant a ser buscado. (Ex: 8871abf3-ed11-4770-b986-e8d98d022d4f)"
// @Param        document query string false "Documento (CNPJ/CPF) do tenant a ser buscado. (Ex: 12345678901234)"
//
// @Success      200  {object}  TenantResponseDto  "model.Tenant encontrado com sucesso."
// @Failure      400  {object}  rest_err.RestErr    "Requisição inválida (UUID inválido, ou nenhum dos campos 'uuid'/'document' fornecido)."
// @Failure      404  {object}  rest_err.RestErr    "model.Tenant não encontrado com os dados fornecidos."
// @Failure      500  {object}  rest_err.RestErr    "Erro interno do servidor."
//
// @Router       /api/tenant [get]
func (ctrl *controllerImpl) Read(c *gin.Context) {
	var req ReadTenantRequestDto
	if err := c.ShouldBindQuery(&req); err != nil {
		restError := rest_err.NewBadRequestError(nil, "Parâmetros de busca inválidos.")
		c.JSON(restError.Code, restError)
		return
	}

	if req.UUID == "" && req.Document == "" {
		restError := rest_err.NewBadRequestError(nil, "É necessário fornecer o 'uuid' OU o 'document' para a busca.")
		c.JSON(restError.Code, restError)
		return
	}

	var tenantUUID uuid.UUID
	if req.UUID != "" {
		// Tentativa de conversão
		parsedUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			// Captura erro de UUID mal formatado (retorna 400)
			restError := rest_err.NewBadRequestError(nil, "O UUID fornecido não é um formato válido.")
			c.JSON(restError.Code, restError)
			return
		}
		tenantUUID = parsedUUID
	}

	ctxIdentify, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		e := rest_err.NewForbiddenError(nil, "Usuário não autenticado.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	rTenant := model.Tenant{
		UUID:     tenantUUID,
		Document: req.Document,
	}

	// analisar se o usuario tem permissao de ler outras empresas ou apenas a propria
	switch ctxIdentify.User.Role {
	case model.RoleSystemAdmin:
		//

	case model.RoleTenantAdmin:
		rTenant = model.Tenant{
			UUID:     ctxIdentify.User.Tenant.UUID,
			Document: ctxIdentify.User.Tenant.Document,
		}

	default:
		e := rest_err.NewForbiddenError(nil, "Ação não permitida.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	rTenant, err := ctrl.service.Read(c.Request.Context(), rTenant)
	if err != nil {
		var restError *rest_err.RestErr
		switch err {
		case ErrNotFound:
			// Tratamento para 404
			restError = rest_err.NewNotFoundError(&ctxIdentify.Metadata.RayTraceCode, ErrNotFound.Error())
		case ErrInvalidInput:
			// Tratamento para 400 (Assumindo que InvalidInput no Read é uma falha na query/dados)
			restError = rest_err.NewBadRequestError(&ctxIdentify.Metadata.RayTraceCode, ErrInvalidInput.Error())
		default:
			// Tratamento para 500
			restError = rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "Falha ao buscar tenant", nil)
		}

		ctrl.logAudit(c, ctxIdentify, "read", "Read", false, req, restError)
		c.JSON(restError.Code, restError)
		return
	}

	resp := &TenantResponseDto{
		UUID:     rTenant.UUID,
		Name:     rTenant.Name,
		Document: rTenant.Document,
		Live:     rTenant.Live,
		CreateAt: rTenant.CreateAt,
		UpdateAt: rTenant.UpdateAt,
	}
	c.JSON(http.StatusOK, resp)
	ctrl.logAudit(c, ctxIdentify, "read", "Read", true, req, resp)
}

// @Summary      Lista Tenants
// @Description  Retorna uma lista paginada de todos os tenants registrados no sistema.
// @Tags         Tenant
// @Produce      json
// @Security     BearerAuth
//
// @Param        page query int true "O número da página a ser retornada (deve ser >= 1)." default(1)
// @Param        pageSize query int true "O número de itens por página (máximo 100)." default(10)
//
// @Success      200  {object}  TenantsResponseDto  "Lista de tenants retornada com sucesso."
// @Failure      400  {object}  rest_err.RestErr    "Requisição inválida (parâmetros de paginação ausentes ou inválidos, ou pageSize > 100)."
// @Failure      500  {object}  rest_err.RestErr    "Erro interno do servidor."
//
// @Router       /api/tenant/list [get]
func (ctrl *controllerImpl) List(c *gin.Context) {
	var req ListTenantRequestDto
	if err := c.ShouldBindQuery(&req); err != nil {
		restError := rest_err.NewBadRequestError(nil, "Parâmetros de busca inválidos. Verifique 'page' e 'pageSize'.")
		c.JSON(restError.Code, restError)
		return
	}

	if req.PageSize > 100 {
		restError := rest_err.NewBadRequestError(nil, "É permitido um máximo de 100 listagens por página.")
		c.JSON(restError.Code, restError)
		return
	}
	ctxIdentify, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		e := rest_err.NewForbiddenError(nil, "Usuário não autenticado.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	lTenants, err := ctrl.service.List(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		var restError *rest_err.RestErr
		restError = rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "Falha ao buscar tenants", nil)
		ctrl.logAudit(c, ctxIdentify, "list", "List", false, req, restError)
		c.JSON(restError.Code, restError)
		return
	}

	tenantResponses := make([]TenantResponseDto, len(lTenants))
	for i, t := range lTenants {
		tenantResponses[i] = TenantResponseDto{
			UUID:     t.UUID,
			Name:     t.Name,
			Document: t.Document,
			Live:     t.Live,
			CreateAt: t.CreateAt,
			UpdateAt: t.UpdateAt,
		}
	}
	resp := &TenantsResponseDto{
		Tenants: tenantResponses,
		Page:    req.Page,
		Size:    req.PageSize,
	}
	c.JSON(http.StatusOK, resp)
	ctrl.logAudit(c, ctxIdentify, "list", "List", true, req, resp)
}

// @Summary      Atualiza um Tenant
// @Description  Atualiza dados de um tenant existente. O tenant a ser atualizado é identificado pelo UUID no path
// @Tags         Tenant
// @Accept       json
// @Produce      json
// @Security     BearerAuth
//
// @Param        uuid path string true "UUID do tenant a ser atualizado."
// @Param        request body UpdateTenantRequestDto true "Campos do tenant a serem atualizados. Apenas os campos presentes serão modificados."
//
// @Success      200  {object}  TenantResponseDto  "model.Tenant atualizado com sucesso."
// @Failure      400  {object}  rest_err.RestErr    "Requisição inválida (corpo JSON mal formatado, UUID inválido ou dados de entrada inválidos)."
// @Failure      404  {object}  rest_err.RestErr    "model.Tenant não encontrado para o UUID fornecido."
// @Failure      409  {object}  rest_err.RestErr    "Conflito (o novo 'document' fornecido já está em uso por outro tenant)."
// @Failure      500  {object}  rest_err.RestErr    "Erro interno do servidor."
//
// @Router       /api/tenant/{uuid} [patch]
func (ctrl *controllerImpl) Update(c *gin.Context) {
	uuidStr := c.Param("uuid")
	tenantUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		restError := rest_err.NewBadRequestError(nil, "O UUID fornecido na URL não é um formato válido.")
		c.JSON(restError.Code, restError)
		return
	}

	var request UpdateTenantRequestDto
	if err := c.ShouldBindJSON(&request); err != nil {
		restError := rest_err.NewBadRequestError(nil, "Corpo JSON inválido ou mal formatado.")
		c.JSON(restError.Code, restError)
		return
	}

	uTenant := model.Tenant{
		UUID:     tenantUUID,
		Document: request.Document,
		Live:     *request.Live,
		Name:     request.Name,
		UpdateAt: time.Now().UTC(),
	}

	ctxIdentify, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		e := rest_err.NewForbiddenError(nil, "Usuário não autenticado.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	// analisar se o usuario tem permissao de ler outras empresas ou apenas a propria
	switch ctxIdentify.User.Role {
	case model.RoleSystemAdmin:
		//
	case model.RoleTenantAdmin:
		uTenant = model.Tenant{
			UUID:     ctxIdentify.User.Tenant.UUID,
			Document: ctxIdentify.User.Tenant.Document,
			Live:     ctxIdentify.User.Tenant.Live,
			Name:     request.Name,
			UpdateAt: time.Now().UTC(),
		}

	default:
		e := rest_err.NewForbiddenError(nil, "Ação não permitida.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	tenantUpdated, err := ctrl.service.Update(c.Request.Context(), &uTenant)

	if err != nil {
		var restError *rest_err.RestErr

		switch err {
		case ErrNotFound:
			restError = rest_err.NewNotFoundError(&ctxIdentify.Metadata.RayTraceCode, ErrNotFound.Error())
		case ErrDocumentDuplicated:
			restError = rest_err.NewConflictValidationError(&ctxIdentify.Metadata.RayTraceCode, "O novo documento fornecido já está em uso por outro ", nil)
		case ErrInvalidInput:
			restError = rest_err.NewBadRequestError(&ctxIdentify.Metadata.RayTraceCode, ErrInvalidInput.Error())
		default:
			restError = rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "Falha ao atualizar tenant", nil)
		}

		ctrl.logAudit(c, ctxIdentify, "update", "Update", false, request, restError)
		c.JSON(restError.Code, restError)
		return
	}

	resp := &TenantResponseDto{
		UUID:     tenantUpdated.UUID,
		Name:     tenantUpdated.Name,
		Document: tenantUpdated.Document,
		Live:     tenantUpdated.Live,
		CreateAt: tenantUpdated.CreateAt,
		UpdateAt: tenantUpdated.UpdateAt,
	}
	c.JSON(http.StatusOK, resp)
	ctrl.logAudit(c, ctxIdentify, "update", "Update", true, request, resp)
}

// @Summary      Deleta um model.Tenant
// @Description  Exclui permanentemente um tenant no sistema usando o UUID ou o Documento (CNPJ/CPF). Pelo menos um dos dois campos deve ser fornecido.
// @Tags         Tenant
// @Produce      json
// @Security     BearerAuth
//
// @Param        uuid query string false "UUID do tenant a ser excluído. (Ex: 8871abf3-ed11-4770-b986-e8d98d022d4f)"
// @Param        document query string false "Documento (CNPJ/CPF) do tenant a ser excluído. (Ex: 12345678901234)"
//
// @Success      204  {string} string "model.Tenant excluído com sucesso (No Content)."
// @Failure      400  {object}  rest_err.RestErr    "Requisição inválida (UUID inválido, ou nenhum dos campos 'uuid'/'document' fornecido)."
// @Failure      404  {object}  rest_err.RestErr    "model.Tenant não encontrado com os dados fornecidos."
// @Failure      500  {object}  rest_err.RestErr    "Erro interno do servidor."
//
// @Router       /api/tenant [delete]
func (ctrl *controllerImpl) Delete(c *gin.Context) {
	var req ReadTenantRequestDto
	if err := c.ShouldBindQuery(&req); err != nil {
		restError := rest_err.NewBadRequestError(nil, "Parâmetros de busca inválidos.")
		c.JSON(restError.Code, restError)
		return
	}

	if req.UUID == "" && req.Document == "" {
		restError := rest_err.NewBadRequestError(nil, "É necessário fornecer o 'uuid' OU o 'document' para a exclusão.")
		c.JSON(restError.Code, restError)
		return
	}

	var tenantUUID uuid.UUID
	if req.UUID != "" {
		parsedUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			restError := rest_err.NewBadRequestError(nil, "O UUID fornecido não é um formato válido.")
			c.JSON(restError.Code, restError)
			return
		}
		tenantUUID = parsedUUID
	}
	ctxIdentify, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		e := rest_err.NewForbiddenError(nil, "Usuário não autenticado.")
		c.AbortWithStatusJSON(e.Code, e)
		return
	}

	err := ctrl.service.Delete(c.Request.Context(), model.Tenant{
		UUID:     tenantUUID,
		Document: req.Document,
	})

	if err != nil {
		var restError *rest_err.RestErr
		switch err {
		case ErrNotFound:
			restError = rest_err.NewNotFoundError(&ctxIdentify.Metadata.RayTraceCode, ErrNotFound.Error())
		case ErrInvalidInput:
			restError = rest_err.NewBadRequestError(&ctxIdentify.Metadata.RayTraceCode, ErrInvalidInput.Error())
		default:
			restError = rest_err.NewInternalServerError(&ctxIdentify.Metadata.RayTraceCode, "Falha ao excluir tenant", nil)
		}

		ctrl.logAudit(c, ctxIdentify, "delete", "Delete", false, req, restError)
		c.JSON(restError.Code, restError)
		return
	}
	ctrl.logAudit(c, ctxIdentify, "delete", "Delete", true, req, gin.H{"status": "deleted"})
	c.Status(http.StatusNoContent)
}
