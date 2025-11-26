package mailer

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"
	"sync"
)

type impl struct {
	cfg SMTPConfig
}

type SMTPConfig struct {
	Host       string
	Port       string
	Username   string
	Password   string
	Encryption string // ex: "tls"
	Address    string // From:
}

var (
	instance                Service
	once                    sync.Once
	initErr                 error
	ErrMailerNotInitialized = errors.New("mailer not initialized")
)

// --- Implementação de Auth Customizada (LOGIN) para Office 365 ---
type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unknown from server")
		}
	}
	return nil, nil
}

// --- Funções de Inicialização ---

// Cria a instância do mailer
func New(cfg SMTPConfig) (Service, error) {
	once.Do(func() {
		if cfg.Host == "" ||
			cfg.Port == "" ||
			cfg.Username == "" ||
			cfg.Password == "" ||
			cfg.Encryption == "" ||
			cfg.Address == "" {

			initErr = errors.New("missing required SMTP configuration")
			return
		}
		instance = &impl{cfg: cfg}
	})

	return instance, initErr
}

// Retorna a instância já inicializada (pode ser nil)
func Use() Service { return instance }

// Valida inicialização do singleton
func validate() error {
	if instance == nil {
		return ErrMailerNotInitialized
	}
	return nil
}

// --- Métodos de Envio ---

// Envia email bruto (HTML ou texto)
func (m *impl) SendRaw(to, subject, body string) error {
	if err := validate(); err != nil {
		return err
	}

	msg := []byte(
		"From: " + m.cfg.Address + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-version: 1.0;\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n" +
			body,
	)

	addr := fmt.Sprintf("%s:%s", m.cfg.Host, m.cfg.Port)

	// Se estiver configurado como TLS (caso Office365 – porta 587 com STARTTLS)
	if m.cfg.Encryption == "tls" {
		err := m.sendWithStartTLS(addr, to, msg)
		if err != nil {
			fmt.Printf("erro ao enviar email via %s: %v\n", addr, err)
			return err
		}
		// CORREÇÃO: Removemos o segundo envio que existia aqui
		return nil
	}

	// Fallback: sem TLS (usando Auth Plain padrão)
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	if err := smtp.SendMail(addr, auth, m.cfg.Address, []string{to}, msg); err != nil {
		fmt.Printf("erro ao enviar email via %s: %v\n", addr, err)
		return fmt.Errorf("erro ao enviar email via %s: %w", addr, err)
	}

	return nil
}

// Implementa envio usando STARTTLS com Auth LOGIN (Office365 fix)
func (m *impl) sendWithStartTLS(addr, to string, msg []byte) error {
	// 1. Conecta
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial error: %w", err)
	}
	defer c.Close()

	// 2. EHLO
	if err = c.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello error: %w", err)
	}

	// 3. STARTTLS
	tlsConfig := &tls.Config{
		ServerName: m.cfg.Host,
	}
	if err = c.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("smtp starttls error: %w", err)
	}

	// 4. Auth após STARTTLS (USANDO LOGIN AUTH CUSTOMIZADO)
	// O Office365 muitas vezes rejeita PlainAuth e exige LoginAuth
	auth := LoginAuth(m.cfg.Username, m.cfg.Password)

	if err = c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth error: %w", err)
	}

	// 5. Envelope
	if err = c.Mail(m.cfg.Address); err != nil {
		return fmt.Errorf("smtp mail error: %w", err)
	}
	if err = c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt error: %w", err)
	}

	// 6. Corpo
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data error: %w", err)
	}
	if _, err = wc.Write(msg); err != nil {
		_ = wc.Close()
		return fmt.Errorf("smtp write error: %w", err)
	}
	if err = wc.Close(); err != nil {
		return fmt.Errorf("smtp close data error: %w", err)
	}

	// 7. QUIT
	if err = c.Quit(); err != nil {
		// Ignora erro no Quit, pois o email já foi enviado e a conexão pode ter fechado
		// return fmt.Errorf("smtp quit error: %w", err)
	}

	return nil
}

// Envia email usando template HTML
func (m *impl) SendTemplate(to, subject string, tpl string, data interface{}) error {
	if err := validate(); err != nil {
		return err
	}

	t, err := template.New("email").Parse(tpl)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return err
	}

	return m.SendRaw(to, subject, buf.String())
}
