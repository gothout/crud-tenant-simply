package auditoria_log

import (
	"context"
	"errors"
	"log"
	"sync"

	"gorm.io/gorm"
)

var (
	instance *Service
	once     sync.Once
	mu       sync.Mutex
	initErr  error

	ErrLogNotInitialized = errors.New("audit logger not initialized")
)

// Config usada somente no New()
type Config struct {
	LogEnabled bool
	Enabled    bool
}

// New inicializa apenas 1x (se Enabled=true)
func New(db *gorm.DB, cfg Config) (*Service, error) {
	once.Do(func() {
		if !cfg.LogEnabled {
			initErr = errors.New("logger disabled in config")
			return
		}
		if !cfg.Enabled {
			initErr = errors.New("audit logger disabled in config")
			return
		}

		if db == nil {
			initErr = errors.New("database required for audit log")
			return
		}

		repo := NewRepository(db)
		instance = NewService(repo)
	})

	return instance, initErr
}

// MustUse simplesmente retorna a instância (pode ser nil)
func MustUse() *Service {
	return instance
}

// Use garante que sempre existe um logger ativo
// Se não existir, inicializa automaticamente (modo DEFAULT, com DB obrigatório)
func Use(db *gorm.DB) *Service {
	mu.Lock()
	defer mu.Unlock()

	// Se já existe → usa
	if instance != nil {
		return instance
	}

	// Se não existe → cria
	if db == nil {
		panic("Use called without DB: database is required to initialize AuditLog")
	}

	repo := NewRepository(db)
	instance = NewService(repo)

	return instance
}

// Apenas valida se existe
func validate() error {
	if instance == nil {
		return ErrLogNotInitialized
	}
	return nil
}

// LogAsync registra auditoria em goroutine destacada.
func LogAsync(ctx context.Context, entry AuditLog) {
	if instance == nil {
		return
	}

	ctxDetached := context.WithoutCancel(ctx)
	go func() {
		if err := instance.Log(ctxDetached, entry); err != nil {
			log.Printf("Erro audit log: %v", err)
		}
	}()
}
