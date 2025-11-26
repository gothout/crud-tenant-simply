package util

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

// Configurações do Argon2
const (
	saltLength  = 16        // 128-bit salt
	memory      = 64 * 1024 // 64 MB memory usage
	iterations  = 3         // Number of iterations
	parallelism = 2         // Number of threads
	keyLength   = 32        // 256-bit derived key
)

// Password define o contrato para hash e comparação de senhas.
type Password interface {
	Hash(password string) (string, error)
	Compare(encodedHash, password string) error
}

// argon2Password é a implementação concreta da interface Password.
type argon2Password struct{}

// Variáveis para o padrão Singleton
var (
	passwordInstance Password
	once             sync.Once
)

// UsePassword retorna a instância única (Singleton) da interface Password.
// A primeira chamada instancia o objeto, as subsequentes retornam o mesmo objeto.
func UsePassword() Password {
	once.Do(func() {
		passwordInstance = &argon2Password{}
	})
	return passwordInstance
}

// Hash gera um hash Argon2id seguro a partir da senha.
func (p *argon2Password) Hash(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memory, iterations, parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash))

	return encoded, nil
}

// Compare verifica se a senha corresponde ao hash Argon2id fornecido.
func (p *argon2Password) Compare(encodedHash, password string) error {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return errors.New("invalid hash format")
	}

	var mem uint32
	var iter uint32
	var par uint8

	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &iter, &par)
	if err != nil {
		return fmt.Errorf("failed to parse hash parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return fmt.Errorf("failed to decode hash: %w", err)
	}

	computedHash := argon2.IDKey([]byte(password), salt, iter, mem, par, uint32(len(expectedHash)))

	if !p.constantTimeCompare(expectedHash, computedHash) {
		return errors.New("invalid password")
	}

	return nil
}

// constantTimeCompare realiza uma comparação de tempo constante para evitar ataques de timing.
func (p *argon2Password) constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
