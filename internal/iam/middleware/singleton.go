package middleware

import (
	"errors"
	"sync"

	"gorm.io/gorm"
)

var (
	middlewareInstance Middleware
	repositoryInstance Repository
	once               sync.Once
	initErr            error
	ErrNotInitialized  = errors.New("middleware not initialized")
)

// UseMiddleware agrupa todas as camadas (Repository, Middleware)
type UseMiddleware struct {
	Repository Repository
	Middleware Middleware
}

// New inicializa o singleton do middleware com todas as suas dependências
func New(db *gorm.DB) (Middleware, error) {
	once.Do(func() {
		if db == nil {
			initErr = errors.New("database connection cannot be nil")
			return
		}

		// Inicializa as dependências em camadas
		repositoryInstance = NewRepository(db)
		middlewareInstance = NewMiddleware(repositoryInstance)
	})

	return middlewareInstance, initErr
}

// Use retorna a instância singleton do middleware
// Retorna erro se o middleware não foi inicializado
func Use() (Middleware, error) {
	if middlewareInstance == nil {
		return nil, ErrNotInitialized
	}
	return middlewareInstance, nil
}

// MustUse retorna todas as camadas (Repository, Middleware)
// Entra em pânico se o singleton não foi inicializado
func MustUse() *UseMiddleware {
	if middlewareInstance == nil || repositoryInstance == nil {
		panic(ErrNotInitialized)
	}
	return &UseMiddleware{
		Repository: repositoryInstance,
		Middleware: middlewareInstance,
	}
}
