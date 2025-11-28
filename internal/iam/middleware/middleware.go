package middleware

import (
	"errors"
	"fmt"
	"strings"
	"tenant-crud-simply/internal/iam/domain/user"
	"time"

	"tenant-crud-simply/internal/pkg/rest_err"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Middleware interface {
	SetContextAutorization() gin.HandlerFunc
	AuthorizeRole(requiredRoles ...user.UserRole) gin.HandlerFunc
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
		authHeader := c.GetHeader("Authorization")
		token := extractBearerToken(authHeader)

		if token == "" {
			err := rest_err.NewForbiddenError("Token ausente ou inválido.")
			c.AbortWithStatusJSON(err.Code, err)
			return
		}

		ctx := c.Request.Context()
		login, err := mw.repository.GetLogin(ctx, token)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				e := rest_err.NewForbiddenError("Token de acesso não encontrado.")
				c.AbortWithStatusJSON(e.Code, e)
				return
			}
			e := rest_err.NewForbiddenError("Falha ao validar token de acesso.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

		if login.AcessToken.UserUUID == nil || *login.AcessToken.UserUUID == uuid.Nil {
			e := rest_err.NewForbiddenError("Token não associado a nenhum usuário válido.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

		if time.Now().UTC().After(login.AcessToken.Expiry) {
			e := rest_err.NewForbiddenError("Token expirado. Efetue login novamente.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

		SetAuthenticatedUser(c, login)
		c.Next()
	}
}

func (mw *impl) AuthorizeRole(requiredRoles ...user.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		lUser, ok := GetAuthenticatedUser(c)

		if !ok {
			e := rest_err.NewForbiddenError("Usuário não autenticado.")
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

		if !isRoleAuthorized(lUser.User.Role, requiredRoles) {
			requiredRoleStrings := make([]string, len(requiredRoles))
			for i, role := range requiredRoles {
				requiredRoleStrings[i] = string(role)
			}

			e := rest_err.NewForbiddenError(fmt.Sprintf(
				"Acesso negado. É necessário possuir uma das permissões: %v.",
				requiredRoleStrings,
			))
			c.AbortWithStatusJSON(e.Code, e)
			return
		}

		c.Next()
	}
}

func isRoleAuthorized(userRole user.UserRole, requiredRoles []user.UserRole) bool {
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
