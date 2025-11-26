package tenant

import (
	"errors"
	"sync"

	"gorm.io/gorm"
)

var (
	controllerInstance Controller
	serviceInstance    Service
	repositoryInstance Repository
	once               sync.Once
	initErr            error
	ErrNotInitialized  = errors.New("tenant controller not initialized")
)

// UseTenant agrupa todas as camadas (Repository, Service, Controller)
type UseTenant struct {
	Repository Repository
	Service    Service
	Controller Controller
}

// New inicializa o singleton do controller de tenant com todas as suas dependências
func New(db *gorm.DB) (Controller, error) {
	once.Do(func() {
		if db == nil {
			initErr = errors.New("database connection cannot be nil")
			return
		}

		// Inicializa as dependências em camadas
		repositoryInstance = NewRepository(db)
		serviceInstance = NewService(repositoryInstance)
		controllerInstance = NewController(serviceInstance)
	})

	return controllerInstance, initErr
}

// Use retorna a instância singleton do controller
// Retorna erro se o controller não foi inicializado
func Use() (Controller, error) {
	if controllerInstance == nil {
		return nil, ErrNotInitialized
	}
	return controllerInstance, nil
}

// MustUse retorna todas as camadas (Repository, Service, Controller)
// Entra em pânico se o singleton não foi inicializado
func MustUse() *UseTenant {
	if controllerInstance == nil || serviceInstance == nil || repositoryInstance == nil {
		panic(ErrNotInitialized)
	}
	return &UseTenant{
		Repository: repositoryInstance,
		Service:    serviceInstance,
		Controller: controllerInstance,
	}
}
