package admin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Status struct {
	Tables    []string
	CheckedAt time.Time
}

func Check(db *gorm.DB) (Status, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return Status{}, fmt.Errorf("falha ao obter conexão subjacente: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return Status{}, fmt.Errorf("banco de dados indisponível: %w", err)
	}

	var tables []string
	err = db.Raw(`
SELECT tablename
FROM pg_catalog.pg_tables
WHERE schemaname = current_schema()
ORDER BY tablename;
        `).Scan(&tables).Error
	if err != nil {
		return Status{}, fmt.Errorf("falha ao listar tabelas: %w", err)
	}

	return Status{Tables: tables, CheckedAt: time.Now()}, nil
}

func DeleteAll(db *gorm.DB) error {
	var tables []string
	if err := db.Raw(`
SELECT tablename
FROM pg_catalog.pg_tables
WHERE schemaname = current_schema();
        `).Scan(&tables).Error; err != nil {
		return fmt.Errorf("falha ao buscar tabelas para exclusão: %w", err)
	}

	for _, table := range tables {
		query := fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE", table)
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("falha ao remover tabela %s: %w", table, err)
		}
	}
	return nil
}

type BackupOptions struct {
	Destination string
}

func Backup(opts BackupOptions) error {
	if opts.Destination == "" {
		return errors.New("destino do backup não informado (use --local=<caminho>)")
	}

	info := connectionInfoFromConfig()
	if info.Database == "" {
		return errors.New("nome do banco de dados não configurado")
	}

	destination := normalizeDestination(opts.Destination)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("falha ao criar diretório do backup: %w", err)
	}

	formatFlag := detectFormat(destination)

	cmd := exec.Command(
		"pg_dump",
		"-h", info.Host,
		"-p", info.Port,
		"-U", info.User,
		"-d", info.Database,
		"-F", formatFlag,
		"-f", destination,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", info.Password))

	if output, err := cmd.CombinedOutput(); err != nil {
		if len(output) > 0 {
			return fmt.Errorf("pg_dump falhou: %w - %s", err, strings.TrimSpace(string(output)))
		}
		return fmt.Errorf("pg_dump falhou: %w", err)
	}

	return nil
}

type connectionInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func connectionInfoFromConfig() connectionInfo {
	return connectionInfo{
		Host:     viper.GetString("databases.postgres.host"),
		Port:     viper.GetString("databases.postgres.port"),
		User:     viper.GetString("databases.postgres.user"),
		Password: viper.GetString("databases.postgres.pwd"),
		Database: viper.GetString("databases.postgres.db_name"),
	}
}

func normalizeDestination(path string) string {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		cwd, err := os.Getwd()
		if err == nil {
			return filepath.Join(cwd, clean)
		}
	}
	return clean
}

func detectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".sql":
		return "p"
	case ".tar":
		return "t"
	default:
		if runtime.GOOS == "windows" {
			return "c"
		}
		return "c"
	}
}
