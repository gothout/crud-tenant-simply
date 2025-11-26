package user

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
	ErrNotInitialized  = errors.New("user controller not initialized")
)

type UseUser struct {
	Repository Repository
	Service    Service
	Controller Controller
}

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

func Use() (Controller, error) {
	if controllerInstance == nil {
		return nil, ErrNotInitialized
	}
	return controllerInstance, nil
}

func MustUse() *UseUser {
	if controllerInstance == nil || serviceInstance == nil || repositoryInstance == nil {
		panic(ErrNotInitialized)
	}
	return &UseUser{
		Repository: repositoryInstance,
		Service:    serviceInstance,
		Controller: controllerInstance,
	}
}
