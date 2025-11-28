package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

const (
	SessionName    = "tenant_session"
	SessionUserKey = "user_token"
)

// RequireAuth middleware para proteger rotas HTML que requerem autenticação
func RequireAuth(store *sessions.CookieStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := store.Get(c.Request, SessionName)
		if err != nil {
			// Se for requisição HTMX, retorna erro ao invés de redirect
			if c.GetHeader("HX-Request") == "true" {
				c.Header("HX-Redirect", "/login")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		token, ok := session.Values[SessionUserKey].(string)
		if !ok || token == "" {
			// Se for requisição HTMX, retorna erro ao invés de redirect
			if c.GetHeader("HX-Request") == "true" {
				c.Header("HX-Redirect", "/login")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Armazena o token no contexto para uso posterior
		c.Set("token", token)
		c.Next()
	}
}

// GetToken retorna o token JWT da sessão
func GetToken(c *gin.Context) string {
	token, exists := c.Get("token")
	if !exists {
		return ""
	}
	return token.(string)
}
