package service

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Reproduz o bug de tipagem do Postgres em $n IS NOT NULL sem cast ::uuid.
// sqlmock não detecta esse erro — só o driver real.
func TestListInbox_PostgresEstabelecimentoParamTyping(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("defina TEST_DATABASE_URL ou DATABASE_URL para validar tipagem no Postgres")
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("conectar ao Postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var exists bool
	if err := db.Get(&exists, `
SELECT EXISTS (
  SELECT 1 FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = 'avisos_globais'
)`); err != nil {
		t.Fatalf("checar tabela avisos_globais: %v", err)
	}
	if !exists {
		t.Skip("migração 000025 ainda não aplicada neste DATABASE_URL")
	}

	svc := NewAvisoService(db)
	ctx := context.Background()

	if _, err := svc.ListInbox(ctx, "00000000-0000-4000-8000-000000000001", "SUPER_ADMIN", nil, 10); err != nil {
		t.Fatalf("ListInbox SUPER_ADMIN (est=nil): %v", err)
	}

	estID := "11111111-1111-4111-8111-111111111111"
	if _, err := svc.ListInbox(ctx, "00000000-0000-4000-8000-000000000002", "DONA", &estID, 10); err != nil {
		t.Fatalf("ListInbox DONA: %v", err)
	}

	_, err = svc.MarkRead(ctx, "00000000-0000-4000-8000-000000000002", "DONA", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", &estID)
	if err != nil && err != ErrAvisoNaoEncontrado {
		t.Fatalf("MarkRead tipagem: %v", err)
	}
}
