package bootstrap

import (
	"context"
	"fmt"
	"log"
	"tenant-crud-simply/internal/iam/application/auth"
	"tenant-crud-simply/internal/iam/domain/tenant"
	"tenant-crud-simply/internal/iam/domain/user"
	"tenant-crud-simply/internal/iam/middleware"
	"tenant-crud-simply/internal/infra/jwt"
	"tenant-crud-simply/internal/pkg/mailer"
	"time"

	"tenant-crud-simply/cmd/server"
	"tenant-crud-simply/internal/infra/database/postgres"

	"github.com/spf13/viper"
	"golang.ngrok.com/ngrok/v2"
	"gorm.io/gorm"
)

// Application armazena as dependências centrais da aplicação.
type Application struct {
	server *server.HTTPServer
}

// Environment configura e lê o arquivo de configuração (configs.json)
func Environment() {
	viper.SetConfigName("configs")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/") // Para ambientes de produção
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error in configuration file: %w", err))
	}
}

func initIamDomain(db *gorm.DB) {
	middleware.New(db)
	tenant.New(db)
	user.New(db)
	auth.New(db)

}

// New prepara a aplicação (config, db, di) e retorna a instância.
func New() (*Application, error) {
	Environment()
	log.Println("[BOOTSTRAP-ENV] Configuração de ambiente carregada.")
	jwtConfig := jwt.Config{
		AccessSecret:  viper.GetString("security.jwt_access_secret"),
		RefreshSecret: viper.GetString("security.jwt_refresh_secret"),
		Issuer:        viper.GetString("app.name"),
		AccessExpiry:  time.Duration(viper.GetInt64("security.jwt_access_expiry_min")) * time.Minute,
	}

	err := jwt.Init(jwtConfig)
	if err != nil {
		// Erro fatal, a aplicação não pode subir sem o gerador de token
		return nil, fmt.Errorf("[BOOTSTRAP-TOKEN] Falha ao criar gerador de token: %w", err)
	}
	tokenInterface := jwt.Use()
	if tokenInterface == nil {
		return nil, fmt.Errorf("[BOOTSTRAP-TOKEN] Falha ao criar gerador de token")
	}
	log.Println("[BOOTSTRAP-TOKEN] Gerador de token inicializado.")
	mailerCfg := mailer.SMTPConfig{Host: viper.GetString("smtp.host"), Port: viper.GetString("smtp.port"), Username: viper.GetString("smtp.username"), Password: viper.GetString("smtp.password"), Encryption: viper.GetString("smtp.encryption"), Address: viper.GetString("smtp.address")}
	_, err = mailer.New(mailerCfg)
	if err != nil {
		log.Println("[BOOTSTRAP-MAILER] Falha ao iniciar sistema de emails")
	} else {
		log.Println("[BOOTSTRAP-MAILER] Sucesso ao iniciar sistema de emails")
	}
	db := postgres.InitPostgres()
	log.Println("[BOOTSTRAP-DATABASE] Conexão com o banco de dados inicializada.")
	initIamDomain(db)
	log.Println("[BOOTSTRAP-DI] Contêiner de dependências inicializado.")

	return &Application{
		server: server.NewHTTPServer(),
	}, nil
}

func startNgrokForward(ctx context.Context, token string, port int) error {
	// Agent com authtoken da config
	agent, err := ngrok.NewAgent(
		ngrok.WithAuthtoken(token),
		ngrok.WithAutoConnect(true), // garante que ele conecta sozinho
	)
	if err != nil {
		return fmt.Errorf("erro criando ngrok Agent: %w", err)
	}

	// IMPORTANTE: usar URL completa do upstream
	// Se teu servidor local for HTTPS, troca pra "https://127.0.0.1:%d"
	upstreamURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	upstream := ngrok.WithUpstream(upstreamURL)

	endpoint, err := agent.Forward(ctx, upstream)
	if err != nil {
		// Se for erro do ngrok, loga o código (tipo ERR_NGROK_3004)
		if ngErr, ok := err.(ngrok.Error); ok {
			log.Printf("[NGROK] erro ao criar forward (code=%s): %v\n", ngErr.Code(), ngErr)
		}
		return fmt.Errorf("erro iniciando ngrok Forward: %w", err)
	}

	log.Println("[NGROK] Endpoint online:", endpoint.URL())

	// Fica vivo até o ctx da aplicação ser cancelado
	<-ctx.Done()

	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := endpoint.CloseWithContext(closeCtx); err != nil {
		return fmt.Errorf("erro ao fechar endpoint ngrok: %w", err)
	}

	if err := agent.Disconnect(); err != nil {
		return fmt.Errorf("erro ao desconectar ngrok Agent: %w", err)
	}

	return nil
}

func (a *Application) Start(ctx context.Context) error {
	log.Println("[BOOTSTRAP] Iniciando servidor no ambiente:", viper.GetString("app.env"))

	errCh := make(chan error, 1)

	// --- sobe o servidor normalmente ---
	go func() {
		errCh <- a.server.Start()

	}()

	// --- inicia ngrok se test.ngrok.live == true ---
	if viper.GetBool("test.ngrok.live") {
		token := viper.GetString("test.ngrok.token")
		if token == "" {
			log.Println("[NGROK] test.ngrok.live=true mas test.ngrok.token está vazio; ngrok NÃO será iniciado")
		} else {
			// ATENÇÃO: aqui assumo que você tem algo como a.server.Port
			// Se não tiver, troca por uma porta fixa (ex: 8080).
			port := viper.GetInt("server.http.port")

			go func() {
				if err := startNgrokForward(ctx, token, port); err != nil {
					log.Println("[NGROK] erro:", err)
				}
			}()
		}
	}

	// --- controle de shutdown do servidor (mantive igual ao teu) ---
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("falha ao encerrar servidor: %w", err)
		}
		return <-errCh

	case err := <-errCh:
		return err
	}
}
