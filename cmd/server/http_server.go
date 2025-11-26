package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/spf13/viper"

	"tenant-crud-simply/cmd/server/routes"
)

type HTTPServer struct {
	server *http.Server
}

func NewHTTPServer() *HTTPServer {
	router := routes.SetupRouter()
	port := fmt.Sprintf(":%s", viper.GetString("server.http.port"))

	return &HTTPServer{server: &http.Server{
		Addr:              port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}}
}

func (s *HTTPServer) Start() error {
	log.Printf("[SERVER] Iniciando servidor na porta %s", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			log.Println("[SERVER] Servidor finalizado.")
			return nil
		}
		return err
	}
	return nil
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	log.Println("[SERVER] Encerrando servidor...")
	return s.server.Shutdown(ctx)
}
