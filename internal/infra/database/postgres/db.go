package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/viper"

	"log"
	"sync"

	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Constantes para os modos SSL permitidos no PostgreSQL
const (
	SSLDisable    = "disable"
	SSLRequire    = "require"
	SSLVerifyFull = "verify-full"
	SSLVerifyCA   = "verify-ca"
)

var (
	db   *gorm.DB
	once sync.Once
)

// InitPostgres inicializa a conexão com o banco de dados PostgreSQL usando GORM.
// A função utiliza sync.Once para garantir que a conexão seja criada apenas uma vez.
func InitPostgres() *gorm.DB {
	once.Do(func() {
		dsn := buildDSN()
		var err error

		// 1. Abre a conexão usando o dialeto GORM e o DSN
		db, err = gorm.Open(gormPostgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("[DATABASE] erro ao abrir conexão GORM: %v", err)
		}

		var sqlDB *sql.DB
		sqlDB, err = db.DB()
		if err != nil {
			log.Fatalf("[DATABASE] erro ao obter *sql.DB do GORM: %v", err)
		}

		if err := sqlDB.Ping(); err != nil {
			log.Fatalf("[DATABASE] erro ao testar conexão com o banco de dados: %v", err)
		}

		log.Println("[DATABASE] Conexão GORM com PostgreSQL estabelecida com sucesso.")
	})

	return db
}

// GetDB retorna a instância atual da conexão GORM.
func GetDB() *gorm.DB {
	if db == nil {

		log.Fatal("[DATABASE] a conexão GORM não foi inicializada. Chame InitPostgres() primeiro.")
	}
	return db
}

// Close encerra a conexão com o banco de dados e permite nova inicialização.
func Close() {
	if db == nil {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("[DATABASE] erro ao obter *sql.DB para fechamento: %v", err)
	} else {
		if err := sqlDB.Close(); err != nil {
			log.Printf("[DATABASE] erro ao fechar conexão com banco: %v", err)
		}
	}

	db = nil
	once = sync.Once{}
}

// buildDSN monta a string de conexão (Data Source Name) para o PostgreSQL
// utilizando as variáveis de ambiente definidas no pacote env.
func buildDSN() string {
	host := viper.GetString("databases.postgres.host")
	port := viper.GetString("databases.postgres.port")
	user := viper.GetString("databases.postgres.user")
	pass := viper.GetString("databases.postgres.pwd")
	name := viper.GetString("databases.postgres.db_name")
	if name == "" {
		name = "appdb"
	}
	ssl := viper.GetString("databases.postgres.ssl_mode")
	if !isValidSSLMode(ssl) {
		log.Printf("[DATABASE] Modo SSL '%s' inválido. Usando o padrão '%s'.", ssl, SSLDisable)
		ssl = SSLDisable
	}

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, ssl,
	)
}

// isValidSSLMode verifica se a string de modo SSL fornecida é um valor válido.
func isValidSSLMode(mode string) bool {
	switch mode {
	case SSLDisable, SSLRequire, SSLVerifyFull, SSLVerifyCA:
		return true
	default:
		return false
	}
}
