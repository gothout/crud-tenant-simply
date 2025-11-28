package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	webMiddleware "tenant-crud-simply/internal/web/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// ProxyAPI faz proxy de requisições HTMX para a API REST, adicionando autenticação
func (h *WebHandler) ProxyAPI(c *gin.Context) {
	token := webMiddleware.GetToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Constrói a URL da API removendo o prefixo /api/web
	apiPath := strings.Replace(c.Request.URL.Path, "/api/web", "/api", 1)
	apiURL := fmt.Sprintf("http://localhost:%d%s", viper.GetInt("server.http.port"), apiPath)

	// Adiciona query string se houver
	if c.Request.URL.RawQuery != "" {
		apiURL += "?" + c.Request.URL.RawQuery
	}

	// Lê o corpo da requisição (se houver)
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// Cria a requisição para a API
	req, err := http.NewRequest(c.Request.Method, apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Copia headers relevantes
	if contentType := c.Request.Header.Get("Content-Type"); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Executa a requisição
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute request"})
		return
	}
	defer resp.Body.Close()

	// Lê a resposta
	respBody, _ := io.ReadAll(resp.Body)

	// Retorna a resposta com o mesmo status code
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// ProxyTenantsListHTML retorna HTML parcial para a tabela de tenants
func (h *WebHandler) ProxyTenantsListHTML(c *gin.Context) {
	token := webMiddleware.GetToken(c)
	if token == "" {
		c.HTML(http.StatusUnauthorized, "partials/tenants_table.html", gin.H{
			"Error": "Não autorizado",
		})
		return
	}

	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("pageSize", "10")

	apiURL := fmt.Sprintf("http://localhost:%d/api/tenant/list?page=%s&pageSize=%s",
		viper.GetInt("server.http.port"), page, pageSize)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "partials/tenants_table.html", gin.H{
			"Error": "Erro ao carregar tenants",
		})
		return
	}
	defer resp.Body.Close()

	var result struct {
		Tenants []map[string]interface{} `json:"tenants"`
		Page    int                      `json:"page"`
		Size    int                      `json:"size"`
	}

	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	data := gin.H{
		"Tenants": result.Tenants,
		"Pagination": gin.H{
			"Page":     result.Page,
			"PageSize": result.Size,
			"HasNext":  len(result.Tenants) == result.Size,
		},
	}

	tmpl, ok := h.templates["partials/tenants_table.html"]
	if !ok {
		c.String(http.StatusInternalServerError, "Template partial not found")
		return
	}
	tmpl.ExecuteTemplate(c.Writer, "tenants_table.html", data)
}

// ProxyUsersListHTML retorna HTML parcial para a tabela de usuários
func (h *WebHandler) ProxyUsersListHTML(c *gin.Context) {
	token := webMiddleware.GetToken(c)
	if token == "" {
		c.HTML(http.StatusUnauthorized, "partials/users_table.html", gin.H{
			"Error": "Não autorizado",
		})
		return
	}

	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("pageSize", "10")

	apiURL := fmt.Sprintf("http://localhost:%d/api/user/list?page=%s&pageSize=%s",
		viper.GetInt("server.http.port"), page, pageSize)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "partials/users_table.html", gin.H{
			"Error": "Erro ao carregar usuários",
		})
		return
	}
	defer resp.Body.Close()

	var users []map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &users)

	for i, user := range users {
		if created, ok := user["create_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, created); err == nil {
				user["create_at"] = t.Format("02/01/2006")
				users[i] = user
			}
		}
	}

	// Busca tenants para mapear nomes
	tenants, _ := h.fetchTenants(token)
	tenantNames := make(map[string]string)
	for _, tenant := range tenants {
		uuid, _ := tenant["UUID"].(string)
		name, _ := tenant["Name"].(string)
		document, _ := tenant["Document"].(string)
		if uuid != "" {
			if document != "" {
				tenantNames[uuid] = fmt.Sprintf("%s (%s)", name, document)
			} else {
				tenantNames[uuid] = name
			}
		}
	}

	data := gin.H{
		"Users":       users,
		"TenantNames": tenantNames,
	}

	tmpl, ok := h.templates["partials/users_table.html"]
	if !ok {
		c.String(http.StatusInternalServerError, "Template partial not found")
		return
	}
	tmpl.ExecuteTemplate(c.Writer, "users_table.html", data)
}
