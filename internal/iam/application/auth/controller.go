package auth

import (
	"errors"
	"net/http"
	"tenant-crud-simply/internal/iam/domain/user"
	"tenant-crud-simply/internal/iam/middleware"
	"tenant-crud-simply/internal/pkg/log/auditoria_log"
	"tenant-crud-simply/internal/pkg/mailer"
	"tenant-crud-simply/internal/pkg/rest_err"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller interface {
	Routes(routes gin.IRouter)
	Healthcheck(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
	CreateOTP(c *gin.Context)
	ResetPassword(c *gin.Context)
}

type controllerImpl struct {
	Service Service
}

func NewController(Service Service) Controller {
	return &controllerImpl{
		Service: Service,
	}
}

// Routes registra as rotas do tenant
func (ctrl *controllerImpl) Routes(routes gin.IRouter) {
	authGroup := routes.Group("/auth")
	{
		authGroup.POST("/login", ctrl.Login)
		authGroup.POST("/logout/:token", ctrl.Logout)
		authGroup.POST("/otp", ctrl.CreateOTP)
		authGroup.POST("/password/reset", ctrl.ResetPassword)
		authGroup.GET("/healthcheck", middleware.MustUse().Middleware.SetContextAutorization(), ctrl.Healthcheck)
	}
}

// @Summary Efetua o login do usuário
// @Description Recebe email e senha, autentica o usuário e retorna o token de acesso.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Credenciais do Usuário (Email e Senha)"
// @Success 200 {object} LoginResponse "Login bem-sucedido"
// @Failure 400 {object} rest_err.RestErr "Requisição inválida (JSON mal formatado)"
// @Failure 404 {object} rest_err.RestErr "Credenciais inválidas (usuário/senha errados)"
// @Failure 409 {object} rest_err.RestErr "Token duplicado ou conflito"
// @Failure 500 {object} rest_err.RestErr "Erro interno do servidor"
// @Router /api/auth/login [post]
func (ctrl *controllerImpl) Login(c *gin.Context) {
	traceID := c.GetHeader("X-Request-ID")
	if traceID == "" {
		traceID = uuid.NewString()
	}
	c.Header("X-Request-ID", traceID)

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		restErr := rest_err.NewBadRequestError(nil, "invalid json body")
		auditoria_log.LogAsync(c.Request.Context(), auditoria_log.AuditLog{
			Identifier:   req.Email,
			RayTraceCode: traceID,
			Domain:       "auth",
			Action:       "login",
			Function:     "Login",
			Success:      false,
			InputData:    auditoria_log.SerializeData(req),
			OutputData:   auditoria_log.SerializeData(restErr),
		})
		c.JSON(restErr.Code, restErr)
		return
	}

	uLogin, err := ctrl.Service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		var restError *rest_err.RestErr
		switch {
		case errors.Is(err, ErrPwdWrong):
			restError = rest_err.NewNotFoundError(nil, err.Error())

		case errors.Is(err, ErrTokenDuplicated):
			restError = rest_err.NewConflictValidationError(nil, err.Error(), nil)

		default:
			restError = rest_err.NewInternalServerError(nil, "internal server error", nil)
		}

		auditoria_log.LogAsync(c.Request.Context(), auditoria_log.AuditLog{
			Identifier:   req.Email,
			RayTraceCode: traceID,
			Domain:       "auth",
			Action:       "login",
			Function:     "Login",
			Success:      false,
			InputData:    auditoria_log.SerializeData(req),
			OutputData:   auditoria_log.SerializeData(restError),
		})

		c.JSON(restError.Code, restError)
		return
	}

	response := LoginResponse{
		User: user.UserResponseDto{
			UUID:       uLogin.User.UUID,
			TenantUUID: uLogin.User.TenantUUID,
			Name:       uLogin.User.Name,
			Email:      uLogin.User.Email,
			Role:       uLogin.User.Role,
			Live:       uLogin.User.Live,
			CreateAt:   uLogin.User.CreateAt,
			UpdateAt:   uLogin.User.UpdateAt,
		},
		Token:  uLogin.AcessToken.Token,
		Expire: uLogin.AcessToken.Expiry,
	}

	auditoria_log.LogAsync(c.Request.Context(), auditoria_log.AuditLog{
		TenantUUID:   uLogin.User.TenantUUID,
		UserUUID:     &uLogin.User.UUID,
		Identifier:   uLogin.User.Email,
		RayTraceCode: traceID,
		Domain:       "auth",
		Action:       "login",
		Function:     "Login",
		Success:      true,
		InputData:    auditoria_log.SerializeData(req),
		OutputData:   auditoria_log.SerializeData(response),
	})

	c.JSON(http.StatusOK, response)
}

// @Summary Revoga o token de acesso
// @Description Invalida o token de acesso atual do usuário.
// @Tags Auth
// @Accept json
// @Produce json
// @Param token path string true "Token de acesso a ser revogado"
// @Success 202 "Token revogado com sucesso"
// @Failure 404 {object} rest_err.RestErr "Token não encontrado"
// @Failure 500 {object} rest_err.RestErr "Erro interno do servidor"
// @Router /api/auth/logout/{token} [post]
func (ctrl *controllerImpl) Logout(c *gin.Context) {
	token := c.Param("token")
	if err := ctrl.Service.RevokeAcessToken(c.Request.Context(), token); err != nil {
		restErr := rest_err.NewForbiddenError(nil, "user not authorized")
		c.JSON(restErr.Code, restErr)
		return
	}
	c.JSON(http.StatusAccepted, nil)
}

// @Summary Solicita um código OTP
// @Description Gera um OTP vinculado ao e-mail e envia por e-mail.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body OTPRequest true "Email para envio do OTP"
// @Success 202 "OTP enviado com sucesso"
// @Failure 400 {object} rest_err.RestErr "JSON inválido"
// @Failure 409 {object} rest_err.RestErr "OTP já existente"
// @Failure 500 {object} rest_err.RestErr "Erro interno"
// @Router /api/auth/otp [post]
func (ctrl *controllerImpl) CreateOTP(c *gin.Context) {
	var req OTPRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		restErr := rest_err.NewBadRequestError(nil, "invalid json body")
		c.JSON(restErr.Code, restErr)
		return
	}

	if err := ctrl.Service.CreateOTPCode(c.Request.Context(), req.Email); err != nil {
		var restErr *rest_err.RestErr

		switch {
		case errors.Is(err, OTPCodeExist):
			restErr = rest_err.NewConflictValidationError(nil, err.Error(), nil)
		case errors.Is(err, user.ErrNotFound):
			restErr = rest_err.NewNotFoundError(nil, err.Error())
		case errors.Is(err, mailer.ErrMailerNotInitialized):
			causes := []rest_err.Causes{rest_err.NewCause("Mailer", "mailer not initialized")}
			restErr = rest_err.NewInternalServerError(nil, "internal server error", causes)
		default:
			restErr = rest_err.NewInternalServerError(nil, "internal server error", nil)
		}

		c.JSON(restErr.Code, restErr)
		return
	}

	c.Status(http.StatusAccepted)
}

// @Summary Troca a senha usando OTP
// @Description Valida o OTP e troca a senha do usuário.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body OTPResetPasswordRequest true "Email, OTP e nova senha"
// @Success 200 "Senha alterada com sucesso"
// @Failure 400 {object} rest_err.RestErr "JSON inválido"
// @Failure 403 {object} rest_err.RestErr "OTP inválido"
// @Failure 500 {object} rest_err.RestErr "Erro interno"
// @Router /api/auth/password/reset [post]
func (ctrl *controllerImpl) ResetPassword(c *gin.Context) {
	var req OTPResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		restErr := rest_err.NewBadRequestError(nil, "invalid json body")
		c.JSON(restErr.Code, restErr)
		return
	}

	ok, err := ctrl.Service.ChangeUserPwd(
		c.Request.Context(),
		req.OTPCode,
		req.Email,
		req.Password,
	)
	if err != nil {
		var restErr *rest_err.RestErr

		switch {
		case errors.Is(err, OTPCodeWrong):
			restErr = rest_err.NewForbiddenError(nil, err.Error())
		default:
			restErr = rest_err.NewInternalServerError(nil, "internal server error", nil)
		}

		c.JSON(restErr.Code, restErr)
		return
	}

	if !ok {
		// fallback defensivo, teoricamente não deveria cair aqui
		restErr := rest_err.NewInternalServerError(nil, "could not change password", nil)
		c.JSON(restErr.Code, restErr)
		return
	}

	c.Status(http.StatusOK)
}

// @Summary Verifica o status do login
// @Description Retorna os dados do usuário logado se o token for válido.
// @Tags Auth
// @Accept json
// @Produce json
// @Security     BearerAuth
// @Success 200 {object} LoginResponse "Dados do usuário logado"
// @Failure 401 {object} rest_err.RestErr "Não autorizado"
// @Router /api/auth/healthcheck [get]
func (ctrl *controllerImpl) Healthcheck(c *gin.Context) {
	lUser, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		restErr := rest_err.NewForbiddenError(nil, "user not authorized")
		c.JSON(restErr.Code, restErr)
		return
	}

	response := LoginResponse{
		User: user.UserResponseDto{
			UUID:       lUser.User.UUID,
			TenantUUID: lUser.User.TenantUUID,
			Name:       lUser.User.Name,
			Email:      lUser.User.Email,
			Role:       lUser.User.Role,
			Live:       lUser.User.Live,
			CreateAt:   lUser.User.CreateAt,
			UpdateAt:   lUser.User.UpdateAt,
		},
		Token:         lUser.AcessToken.Token,
		SystemTimeUTC: time.Now().UTC(),
		Expire:        lUser.AcessToken.Expiry,
	}

	c.JSON(http.StatusOK, response)
}
