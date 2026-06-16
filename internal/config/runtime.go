package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	// Portas reservadas do AgendaGlow (isoladas do WhatsApp Gateway e Postgres local).
	DefaultAPIPort      = "8081"
	DefaultFrontendPort = "8082"
	DefaultDBHostPort   = "5435"
	DefaultDBInternalPort = "5432"

	// Portas ocupadas por outros sistemas — nunca usar no AgendaGlow.
	GatewayAPIPort     = "8080"
	GatewayDBPort      = "5433"
	LocalPostgresPort  = "5432"
)

var forbiddenAPIPorts = map[string]string{
	GatewayAPIPort:    "WhatsApp Gateway API",
	DefaultFrontendPort: "front-end AgendaGlow (reservado para container futuro)",
}

var forbiddenDBHostPorts = map[string]string{
	LocalPostgresPort: "PostgreSQL local da máquina",
	GatewayDBPort:     "WhatsApp Gateway PostgreSQL",
}

// ValidateRuntime impede conflito de portas com Gateway (8080/5433) ou Postgres local (5432).
func ValidateRuntime() error {
	apiPort := APIPort()
	if err := assertAPIPort(apiPort); err != nil {
		return err
	}

	if frontendPort := strings.TrimSpace(os.Getenv("FRONTEND_PORT")); frontendPort != "" {
		if frontendPort != DefaultFrontendPort {
			return fmt.Errorf(
				"FRONTEND_PORT=%s inválida: AgendaGlow reserva %s para o front-end containerizado",
				frontendPort, DefaultFrontendPort,
			)
		}
		if frontendPort == apiPort {
			return fmt.Errorf("FRONTEND_PORT não pode ser igual à PORT da API (%s)", apiPort)
		}
	}

	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	if host == "" || host == "localhost" || host == "127.0.0.1" {
		if err := assertDBHostPort(DBHostPort()); err != nil {
			return err
		}
	}

	return nil
}

// APIPort retorna a porta HTTP da API (PORT tem precedência sobre HTTP_ADDR legado).
func APIPort() string {
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		return port
	}
	if addr := strings.TrimSpace(os.Getenv("HTTP_ADDR")); addr != "" {
		addr = strings.TrimPrefix(addr, ":")
		if _, err := strconv.Atoi(addr); err == nil {
			return addr
		}
	}
	return DefaultAPIPort
}

// ListenAddr devolve endereço de bind no formato ":8081".
func ListenAddr() string {
	return ":" + APIPort()
}

// DBHostPort é a porta do Postgres exposta no host (5435).
func DBHostPort() string {
	if port := strings.TrimSpace(os.Getenv("DB_PORT")); port != "" {
		return port
	}
	return DefaultDBHostPort
}

// DatabaseURL monta a connection string ou usa DATABASE_URL quando definida.
func DatabaseURL() (string, error) {
	if direct := strings.TrimSpace(os.Getenv("DATABASE_URL")); direct != "" {
		return direct, nil
	}

	host := envOrDefault("DB_HOST", "postgres-glow")
	user := envOrDefault("DB_USER", "postgres")
	password := os.Getenv("DB_PASSWORD")
	dbName := envOrDefault("DB_NAME", "agenda_glow_prod")

	port := DBConnectPort(host)
	if password == "" {
		return "", fmt.Errorf("DB_PASSWORD não configurada")
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   dbName,
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// DBConnectPort escolhe 5432 dentro da rede Docker ou DB_PORT no host.
func DBConnectPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "postgres-glow" {
		return envOrDefault("DB_INTERNAL_PORT", DefaultDBInternalPort)
	}
	return DBHostPort()
}

func assertAPIPort(port string) error {
	if port != DefaultAPIPort {
		return fmt.Errorf(
			"PORT=%s inválida: AgendaGlow deve usar exclusivamente %s (Gateway ocupa %s)",
			port, DefaultAPIPort, GatewayAPIPort,
		)
	}
	if reason, blocked := forbiddenAPIPorts[port]; blocked && port != DefaultAPIPort {
		return fmt.Errorf("PORT=%s conflita com %s", port, reason)
	}
	if port == GatewayAPIPort {
		return fmt.Errorf("PORT=%s conflita com WhatsApp Gateway API (%s)", port, GatewayAPIPort)
	}
	return nil
}

func assertDBHostPort(port string) error {
	if port != DefaultDBHostPort {
		return fmt.Errorf(
			"DB_PORT=%s inválida: AgendaGlow deve usar %s no host (Gateway=%s, Postgres local=%s)",
			port, DefaultDBHostPort, GatewayDBPort, LocalPostgresPort,
		)
	}
	if reason, blocked := forbiddenDBHostPorts[port]; blocked {
		return fmt.Errorf("DB_PORT=%s conflita com %s", port, reason)
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
