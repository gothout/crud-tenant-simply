package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	webMiddleware "tenant-crud-simply/internal/web/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
)

type WebHandler struct {
	sessionStore *sessions.CookieStore
	templates    map[string]*template.Template
}

// fetchTenants busca uma lista de tenants para uso em dropdowns e tabelas
func (h *WebHandler) fetchTenants(token string) ([]map[string]interface{}, error) {
	apiURL := fmt.Sprintf("http://localhost:%d/api/tenant/list?page=1&pageSize=100", viper.GetInt("server.http.port"))

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Tenants []map[string]interface{} `json:"tenants"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Tenants, nil
}

// NewWebHandler cria uma nova instância do handler web
func NewWebHandler(sessionStore *sessions.CookieStore) (*WebHandler, error) {
	// Mapa para armazenar templates compilados por página
	templates := make(map[string]*template.Template)

	// Funções auxiliares para templates
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
	}

	// Lista de páginas que usam o base.html
	pages := []string{
		"login.html",
		"dashboard.html",
		"tenants.html",
		"users.html",
	}

	for _, page := range pages {
		// Para cada página, cria um novo template isolado contendo base.html + pagina.html
		tmpl := template.New("").Funcs(funcMap)
		tmpl, err := tmpl.ParseFiles(
			"internal/web/templates/base.html",
			"internal/web/templates/"+page,
		)
		if err != nil {
			return nil, fmt.Errorf("error parsing template %s: %w", page, err)
		}
		templates[page] = tmpl
	}

	// Carrega parciais (sem base.html)
	partials := []string{
		"partials/tenants_table.html",
		"partials/users_table.html",
	}
	for _, partial := range partials {
		tmpl := template.New("").Funcs(funcMap)
		tmpl, err := tmpl.ParseFiles("internal/web/templates/" + partial)
		if err != nil {
			return nil, fmt.Errorf("error parsing partial %s: %w", partial, err)
		}
		templates[partial] = tmpl
	}

	return &WebHandler{
		sessionStore: sessionStore,
		templates:    templates,
	}, nil
}

// renderTemplate helper para renderizar templates com segurança
func (h *WebHandler) renderTemplate(c *gin.Context, page string, data gin.H) {
	tmpl, ok := h.templates[page]
	if !ok {
		c.String(http.StatusInternalServerError, "Template not found: "+page)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(c.Writer, "base.html", data); err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template: "+err.Error())
	}
}

// ServeLogin exibe a página de login
func (h *WebHandler) ServeLogin(c *gin.Context) {
	h.renderTemplate(c, "login.html", gin.H{
		"Title":      "Login",
		"ShowHeader": false,
		"ShowFooter": false,
	})
}

// HandleLogin processa o formulário de login
func (h *WebHandler) HandleLogin(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	// Chama a API de login
	loginData := map[string]string{
		"email":    email,
		"password": password,
	}
	jsonData, _ := json.Marshal(loginData)

	apiURL := fmt.Sprintf("http://localhost:%d/api/auth/login", viper.GetInt("server.http.port"))
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		c.Writer.WriteHeader(http.StatusBadRequest)
		h.renderTemplate(c, "login.html", gin.H{
			"Title":      "Login",
			"Error":      "Erro ao conectar com o servidor",
			"ShowHeader": false,
			"ShowFooter": false,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		h.renderTemplate(c, "login.html", gin.H{
			"Title":      "Login",
			"Error":      "Email ou senha incorretos",
			"ShowHeader": false,
			"ShowFooter": false,
		})
		return
	}

	// Lê a resposta
	var loginResponse struct {
		Token  string `json:"token"`
		Expire string `json:"expire"`
		User   struct {
			UUID  string `json:"uuid"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"user"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &loginResponse); err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		h.renderTemplate(c, "login.html", gin.H{
			"Title":      "Login",
			"Error":      "Erro ao processar resposta do servidor",
			"ShowHeader": false,
			"ShowFooter": false,
		})
		return
	}

	// Salva o token na sessão
	session, _ := h.sessionStore.Get(c.Request, webMiddleware.SessionName)
	session.Values[webMiddleware.SessionUserKey] = loginResponse.Token
	session.Values["user_name"] = loginResponse.User.Name
	session.Values["user_email"] = loginResponse.User.Email
	session.Values["user_role"] = loginResponse.User.Role
	if err := session.Save(c.Request, c.Writer); err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		h.renderTemplate(c, "login.html", gin.H{
			"Title":      "Login",
			"Error":      "Erro ao salvar sessão",
			"ShowHeader": false,
			"ShowFooter": false,
		})
		return
	}

	// Redireciona para o dashboard
	c.Redirect(http.StatusFound, "/dashboard")
}

// HandleLogout limpa a sessão do usuário
func (h *WebHandler) HandleLogout(c *gin.Context) {
	session, _ := h.sessionStore.Get(c.Request, webMiddleware.SessionName)

	// Limpa a sessão
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	session.Save(c.Request, c.Writer)

	c.Redirect(http.StatusFound, "/login")
}

// ServeDashboard exibe o dashboard principal
func (h *WebHandler) ServeDashboard(c *gin.Context) {
	session, _ := h.sessionStore.Get(c.Request, webMiddleware.SessionName)

	h.renderTemplate(c, "dashboard.html", gin.H{
		"Title":      "Dashboard",
		"ShowHeader": true,
		"ShowFooter": true,
		"User": gin.H{
			"Name":  session.Values["user_name"],
			"Email": session.Values["user_email"],
			"Role":  session.Values["user_role"],
			"Live":  true,
		},
	})
}

// ServeTenants exibe a página de gerenciamento de tenants
func (h *WebHandler) ServeTenants(c *gin.Context) {
	session, _ := h.sessionStore.Get(c.Request, webMiddleware.SessionName)

	h.renderTemplate(c, "tenants.html", gin.H{
		"Title":      "Gerenciar Tenants",
		"ShowHeader": true,
		"ShowFooter": true,
		"User": gin.H{
			"Name":  session.Values["user_name"],
			"Email": session.Values["user_email"],
		},
	})
}

// ServeUsers exibe a página de gerenciamento de usuários
func (h *WebHandler) ServeUsers(c *gin.Context) {
	session, _ := h.sessionStore.Get(c.Request, webMiddleware.SessionName)

	token, _ := session.Values[webMiddleware.SessionUserKey].(string)
	tenants, _ := h.fetchTenants(token)

	h.renderTemplate(c, "users.html", gin.H{
		"Title":      "Gerenciar Usuários",
		"ShowHeader": true,
		"ShowFooter": true,
		"User": gin.H{
			"Name":  session.Values["user_name"],
			"Email": session.Values["user_email"],
		},
		"Tenants": tenants,
	})
}
