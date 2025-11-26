package migrations

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	seedCategory   = "seed"
	updateCategory = "update"
)

var (
	//go:embed sql/seed/*.sql sql/update/*.sql
	embeddedMigrations embed.FS
)

type migrationFile struct {
	Name      string
	Content   string
	Category  string
	Timestamp time.Time
}

type Manager struct {
	db *gorm.DB
}

func NewManager(db *gorm.DB) *Manager {
	return &Manager{db: db}
}

func (m *Manager) ApplySeed() error {
	return m.applyCategory(seedCategory)
}

func (m *Manager) ApplyUpdate() error {
	return m.applyCategory(updateCategory)
}

func (m *Manager) applyCategory(category string) error {
	files, err := loadFiles(category)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	return m.db.Transaction(func(tx *gorm.DB) error {
		if err := ensureSchemaMigrationsTable(tx); err != nil {
			return err
		}

		applied, err := fetchApplied(tx, category)
		if err != nil {
			return err
		}

		for _, file := range files {
			if applied[file.Name] {
				continue
			}

			if err := executeMigration(tx, file); err != nil {
				return err
			}
		}

		return nil
	})
}

func loadFiles(category string) ([]migrationFile, error) {
	var dir string
	switch category {
	case seedCategory:
		dir = "sql/seed"
	case updateCategory:
		dir = "sql/update"
	default:
		return nil, fmt.Errorf("categoria de migration desconhecida: %s", category)
	}

	entries, err := embeddedMigrations.ReadDir(dir)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return nil, nil
		}
		return nil, fmt.Errorf("falha ao ler diretório de migrations %s: %w", dir, err)
	}

	files := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		content, err := embeddedMigrations.ReadFile(filepath.ToSlash(filepath.Join(dir, name)))
		if err != nil {
			return nil, fmt.Errorf("falha ao ler migration %s: %w", name, err)
		}

		ts, err := parseTimestamp(name)
		if err != nil {
			return nil, err
		}

		files = append(files, migrationFile{
			Name:      name,
			Content:   string(content),
			Category:  category,
			Timestamp: ts,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Timestamp.Equal(files[j].Timestamp) {
			return files[i].Name < files[j].Name
		}
		return files[i].Timestamp.Before(files[j].Timestamp)
	})

	return files, nil
}

func parseTimestamp(name string) (time.Time, error) {
	base := filepath.Base(name)
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 0 {
		return time.Time{}, fmt.Errorf("migration %s não segue o padrão 'YYYYMMDDHHMMSS_nome.sql'", name)
	}
	ts := parts[0]
	if len(ts) != 14 {
		return time.Time{}, fmt.Errorf("migration %s não contém carimbo de data e hora válido", name)
	}

	parsed, err := time.Parse("20060102150405", ts)
	if err != nil {
		return time.Time{}, fmt.Errorf("falha ao interpretar data da migration %s: %w", name, err)
	}
	return parsed, nil
}

func ensureSchemaMigrationsTable(tx *gorm.DB) error {
	const createTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(50) NOT NULL,
    applied_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (name, category)
);`
	return tx.Exec(createTable).Error
}

func fetchApplied(tx *gorm.DB, category string) (map[string]bool, error) {
	type record struct {
		Name string
	}
	var rows []record
	if err := tx.Raw(
		"SELECT name FROM schema_migrations WHERE category = ?",
		category,
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("falha ao consultar migrations aplicadas (%s): %w", category, err)
	}

	applied := make(map[string]bool, len(rows))
	for _, row := range rows {
		applied[row.Name] = true
	}
	return applied, nil
}

func executeMigration(tx *gorm.DB, file migrationFile) error {
	if err := tx.Exec(file.Content).Error; err != nil {
		return fmt.Errorf("falha ao aplicar migration %s: %w", file.Name, err)
	}

	if err := tx.Exec(
		"INSERT INTO schema_migrations (name, category) VALUES (?, ?)",
		file.Name,
		file.Category,
	).Error; err != nil {
		return fmt.Errorf("falha ao registrar migration %s: %w", file.Name, err)
	}

	return nil
}
