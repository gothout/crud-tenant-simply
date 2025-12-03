package middleware

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"tenant-crud-simply/internal/iam/domain/model"
	"tenant-crud-simply/internal/pkg/log/acess_log"
	"time"

	"tenant-crud-simply/internal/pkg/rest_err"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Middleware interface {
	SetContextAutorization() gin.HandlerFunc
	AuthorizeRole(requiredRoles ...model.UserRole) gin.HandlerFunc
}

type impl struct {
	repository Repository
}

func NewMiddleware(repository Repository) Middleware {
	return &impl{
		repository: repository,
	}
}

func (mw *impl) SetContextAutorization() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		traceID := c.GetHeader("X-Request-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}

		authHeader := c.GetHeader("Authorization")
		token := extractBearerToken(authHeader)

		if token == "" {
			err := rest_err.NewForbiddenError(nil, "Token ausente ou inválido.")
			c.Header("X-Request-ID", traceID)
			c.AbortWithStatusJSON(err.Code, err)
			return
		}

		ctx := c.Request.Context()
		login, err := mw.repository.GetLogin(ctx, token)
		if err != nil {
			var e *rest_err.RestErr
			if errors.Is(err, gorm.ErrRecordNotFound) {
				e = rest_err.NewForbiddenError(nil, "Token de acesso não encontrado.")
			} else {
				e = rest_err.NewForbiddenError(nil, "Falha ao validar token de acesso.")
			}

			c.Header("X-Request-ID", traceID)
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

		if login.AcessToken.UserUUID == nil || *login.AcessToken.UserUUID == uuid.Nil {
			e := rest_err.NewForbiddenError(nil, "Token não associado a nenhum usuário válido.")
			c.Header("X-Request-ID", traceID)
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

		if time.Now().UTC().After(login.AcessToken.Expiry) {
			e := rest_err.NewForbiddenError(&login.Metadata.RayTraceCode, "Token expirado. Efetue login novamente.")
			c.Header("X-Request-ID", traceID)
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

		// 2. Preenche metadata
		login.Metadata = Metadata{
			RayTraceCode: traceID,
			IP:           c.ClientIP(),
			Agent:        c.Request.UserAgent(),
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			Host:         c.Request.Host,
			Referer:      c.Request.Referer(),
			ContentType:  c.ContentType(),
			UserLanguage: c.GetHeader("Accept-Language"),
			TimeRequest:  start.UTC(),
		}

		// 3. RayTrace no contexto e header
		c.Set("rayTraceCode", traceID)
		c.Header("X-Request-ID", traceID)

		SetAuthenticatedUser(c, login)

		// processa handler
		c.Next()

		// 4. calcular latência
		login.Metadata.RequestLatency = time.Since(start)

		// 5. Preparar dados para log
		statusCode := c.Writer.Status()

		var userUUID *uuid.UUID
		if login.User.UUID != uuid.Nil {
			userUUID = &login.User.UUID
		}

		identifier := login.User.Email

		accessLog := acess_log.AccessLog{
			TenantUUID:   login.User.TenantUUID,
			UserUUID:     userUUID,
			Identifier:   identifier,
			RayTraceCode: login.Metadata.RayTraceCode,
			Method:       login.Metadata.Method,
			Path:         login.Metadata.Path,
			Host:         login.Metadata.Host,
			StatusCode:   statusCode,
			IP:           login.Metadata.IP,
			UserAgent:    login.Metadata.Agent,
			Referer:      login.Metadata.Referer,
			ContentType:  login.Metadata.ContentType,
			UserLanguage: login.Metadata.UserLanguage,
			RequestTime:  login.Metadata.TimeRequest,
			LatencyMs:    float64(login.Metadata.RequestLatency.Microseconds()) / 1000.0,
		}
		ctxDetached := context.WithoutCancel(ctx)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered in access log: %v", r)
				}
			}()
			if err := acess_log.MustUse().Log(ctxDetached, accessLog); err != nil {
				log.Printf("Erro log: %v", err)
			}
		}()
	}
}

func (mw *impl) AuthorizeRole(requiredRoles ...model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		lUser, ok := GetAuthenticatedUser(c)

		if !ok {
			e := rest_err.NewForbiddenError(nil, "Usuário não autenticado.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

		if !isRoleAuthorized(lUser.User.Role, requiredRoles) {
			requiredRoleStrings := make([]string, len(requiredRoles))
			for i, role := range requiredRoles {
				requiredRoleStrings[i] = string(role)
			}
			e := rest_err.NewForbiddenError(nil, fmt.Sprintf(
				"Acesso negado. É necessário possuir uma das permissões: %v.",
				requiredRoleStrings,
			))
			c.AbortWithStatusJSON(e.Code, e)
			return
		}
		c.Next()
	}
}

func isRoleAuthorized(userRole model.UserRole, requiredRoles []model.UserRole) bool {
	if len(requiredRoles) == 0 {
		return true
	}
	for _, requiredRole := range requiredRoles {
		if userRole == requiredRole {
			return true
		}
	}
	return false
}

func extractBearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
