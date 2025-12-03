package acess_log

import (
	"errors"
	"sync"

	"gorm.io/gorm"
)

var (
	instance *Service
	once     sync.Once
	mu       sync.Mutex
	initErr  error

	ErrLogNotInitialized = errors.New("access logger not initialized")
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
			initErr = errors.New("access logger disabled in config")
			return
		}

		if db == nil {
			initErr = errors.New("database required for access log")
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
		panic("MustUse called without DB: database is required to initialize AccessLog")
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
