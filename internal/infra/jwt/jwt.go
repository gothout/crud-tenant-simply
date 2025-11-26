package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// 1. Variável global privada que guardará a instância única
var singleton *TokenGenerator

type AccessTokenClaims struct {
	TenantID string `json:"tenant"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	jwt.RegisteredClaims
}

type TokenGenerator struct {
	accessSecretKey  []byte
	refreshSecretKey []byte
	issuer           string
	accessExpiry     time.Duration
}

type Config struct {
	AccessSecret  string
	RefreshSecret string
	Issuer        string
	AccessExpiry  time.Duration
}

// 2. Função Init: Inicializa o Singleton (chame isso apenas uma vez, no main)
func Init(cfg Config) error {
	if cfg.AccessSecret == "" || cfg.RefreshSecret == "" {
		return fmt.Errorf("segredos JWT não podem estar vazios")
	}
	if cfg.Issuer == "" {
		return fmt.Errorf("emissor (issuer) JWT não pode estar vazio")
	}
	if cfg.AccessExpiry <= 0 {
		return fmt.Errorf("expiração do token deve ser positiva")
	}

	singleton = &TokenGenerator{
		accessSecretKey:  []byte(cfg.AccessSecret),
		refreshSecretKey: []byte(cfg.RefreshSecret),
		issuer:           cfg.Issuer,
		accessExpiry:     cfg.AccessExpiry,
	}

	return nil
}

// 3. Função Use: Retorna a instância global para ser usada em qualquer lugar
func Use() *TokenGenerator {
	if singleton == nil {
		panic("JWT package não foi inicializado. Chame jwt.Init(cfg) no startup da aplicação.")
	}
	return singleton
}

// Métodos continuam iguais, atrelados ao struct TokenGenerator

func (tg *TokenGenerator) GenerateAccessToken(userID uuid.UUID, tenantID uuid.UUID) (string, time.Time, error) {
	expirationTime := time.Now().UTC().Add(tg.accessExpiry)

	claims := &AccessTokenClaims{
		TenantID: tenantID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    tg.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(tg.accessSecretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("erro ao assinar o access token: %w", err)
	}

	return tokenString, expirationTime, nil
}

func (tg *TokenGenerator) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	claims := &RefreshTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  userID.String(),
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   tg.issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(tg.refreshSecretKey)
	if err != nil {
		return "", fmt.Errorf("erro ao assinar o refresh token: %w", err)
	}
	return tokenString, nil
}
