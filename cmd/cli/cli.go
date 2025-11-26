package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"gorm.io/gorm"

	"tenant-crud-simply/cmd/bootstrap"
	"tenant-crud-simply/internal/infra/database/admin"
	"tenant-crud-simply/internal/infra/database/migrations"
	"tenant-crud-simply/internal/infra/database/postgres"
)

type options struct {
	Start             bool
	Stop              bool
	Seed              bool
	Update            bool
	DBCheck           bool
	DBDelete          bool
	DBBackup          bool
	BackupDestination string
}

func Execute() error {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		return err
	}

	if !opts.anyOperation() {
		log.Println("Nenhuma operação informada. Use --help para listar as opções disponíveis.")
		return nil
	}

	if opts.Stop {
		if err := stopServer(); err != nil {
			return fmt.Errorf("falha ao parar servidor: %w", err)
		}
		log.Println("Servidor finalizado com sucesso.")
		return nil
	}

	bootstrap.Environment()

	var (
		db         *gorm.DB
		manager    *migrations.Manager
		needDB     = opts.requiresDatabase()
		operations bool
	)

	if needDB {
		db = postgres.InitPostgres()
		defer postgres.Close()
	}

	if opts.Seed {
		if manager == nil {
			manager = migrations.NewManager(db)
		}
		if err := manager.ApplySeed(); err != nil {
			return fmt.Errorf("falha ao aplicar migrations de seed: %w", err)
		}
		log.Println("Migrations de seed aplicadas com sucesso.")
		operations = true
	}

	if opts.Update {
		if manager == nil {
			manager = migrations.NewManager(db)
		}
		if err := manager.ApplyUpdate(); err != nil {
			return fmt.Errorf("falha ao aplicar migrations de atualização: %w", err)
		}
		log.Println("Migrations de atualização aplicadas com sucesso.")
		operations = true
	}

	if opts.DBCheck {
		if db == nil {
			return fmt.Errorf("conexão com o banco de dados não inicializada")
		}
		status, err := admin.Check(db)
		if err != nil {
			return fmt.Errorf("falha ao checar banco de dados: %w", err)
		}
		log.Printf("Banco de dados ativo. Tabelas encontradas (%d): %v", len(status.Tables), status.Tables)
		operations = true
	}

	if opts.DBDelete {
		if db == nil {
			return fmt.Errorf("conexão com o banco de dados não inicializada")
		}
		if err := admin.DeleteAll(db); err != nil {
			return fmt.Errorf("falha ao deletar tabelas do banco: %w", err)
		}
		log.Println("Todas as tabelas foram removidas com sucesso.")
		operations = true
	}

	if opts.DBBackup {
		if opts.BackupDestination == "" {
			return fmt.Errorf("para executar o backup informe o destino com --local=<caminho>")
		}

		dest := opts.BackupDestination
		if !filepath.IsAbs(dest) {
			if abs, err := filepath.Abs(dest); err == nil {
				dest = abs
			}
		}

		if err := admin.Backup(admin.BackupOptions{Destination: dest}); err != nil {
			return fmt.Errorf("falha ao executar backup: %w", err)
		}
		log.Printf("Backup gerado em %s", dest)
		operations = true
	}

	if opts.Start {
		if err := startServer(); err != nil {
			return fmt.Errorf("falha ao iniciar servidor: %w", err)
		}
		operations = true
	}

	if !operations {
		log.Println("Nenhuma operação executada. Use --help para listar as opções disponíveis.")
	}

	return nil
}

func parseOptions(args []string) (options, error) {
	var opts options

	fs := pflag.NewFlagSet("tenant-crud", pflag.ContinueOnError)
	fs.BoolVar(&opts.Start, "start", false, "Inicia o servidor HTTP")
	fs.BoolVar(&opts.Stop, "stop", false, "Finaliza o servidor HTTP")
	fs.BoolVar(&opts.Seed, "migration-seed", false, "Aplica migrations de seed")
	fs.BoolVar(&opts.Update, "migration-update", false, "Aplica migrations de atualização")
	fs.BoolVar(&opts.DBCheck, "db-check", false, "Checa status do banco de dados")
	fs.BoolVar(&opts.DBDelete, "db-delete", false, "Remove todas as tabelas do banco de dados")
	fs.BoolVar(&opts.DBBackup, "db-backup", false, "Realiza backup do banco de dados")
	fs.StringVar(&opts.BackupDestination, "local", "", "Diretório de destino para o backup do banco")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	return opts, nil
}

func (o options) anyOperation() bool {
	return o.Start || o.Stop || o.Seed || o.Update || o.DBCheck || o.DBDelete || o.DBBackup
}

func (o options) requiresDatabase() bool {
	return o.Seed || o.Update || o.DBCheck || o.DBDelete
}
