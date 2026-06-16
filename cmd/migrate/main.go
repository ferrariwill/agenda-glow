// Aplica migrações SQL em ordem (migrations/*.up.sql) no Postgres configurado via env.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/agendaglow/agendaglow/internal/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("migrate: %v", err)
	}
}

func run() error {
	direction := strings.ToLower(strings.TrimSpace(envOr("MIGRATE_DIRECTION", "up")))
	if direction != "up" {
		return fmt.Errorf("direção %q não suportada (use MIGRATE_DIRECTION=up)", direction)
	}

	databaseURL, err := config.DatabaseURL()
	if err != nil {
		return err
	}

	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("conectar ao banco: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := ensureSchemaTable(ctx, db); err != nil {
		return err
	}

	dir := envOr("MIGRATIONS_DIR", "migrations")
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("listar migrações: %w", err)
	}
	sort.Strings(files)

	if len(files) == 0 {
		log.Printf("nenhuma migração encontrada em %s", dir)
		return nil
	}

	applied := 0
	for _, path := range files {
		name := filepath.Base(path)
		ok, err := isApplied(ctx, db, name)
		if err != nil {
			return err
		}
		if ok {
			log.Printf("skip %s (já aplicada)", name)
			continue
		}

		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("ler %s: %w", name, err)
		}

		log.Printf("apply %s ...", name)
		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("executar %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, `
INSERT INTO schema_migrations (filename, applied_at) VALUES ($1, NOW())
`, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("registrar %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}

		applied++
	}

	log.Printf("migrações concluídas: %d nova(s), %d arquivo(s) verificados", applied, len(files))
	return nil
}

func ensureSchemaTable(ctx context.Context, db *sqlx.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    filename   VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
)
`
	_, err := db.ExecContext(ctx, ddl)
	return err
}

func isApplied(ctx context.Context, db *sqlx.DB, filename string) (bool, error) {
	var exists bool
	err := db.GetContext(ctx, &exists, `
SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)
`, filename)
	return exists, err
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
