//go:build integration

package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Exercita ListInbox/MarkRead no Postgres real: o bug de tipagem
// ("could not determine data type of parameter $n") só aparece fora do sqlmock.
func TestAvisoInbox_PostgresParamTyping(t *testing.T) {
	db := newAvisosInboxPostgres(t)
	svc := NewAvisoService(db)
	ctx := context.Background()

	superID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	donaID := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	estID := "11111111-1111-4111-8111-111111111111"

	avisosMustExec(t, db, `
INSERT INTO users (id, email, nome, role) VALUES
  ($1, 'super@glow.local', 'Super', 'SUPER_ADMIN'),
  ($2, 'dona@glow.local', 'Dona', 'DONA')
`, superID, donaID)
	avisosMustExec(t, db, `
INSERT INTO estabelecimentos (id, nome_comercial) VALUES ($1, 'Salao Teste')
`, estID)

	allTenants, err := svc.Create(ctx, CreateAvisoInput{
		Titulo:       "Broadcast donas",
		AudienceTipo: AvisoAudienceAllTenants,
		CreatedBy:    superID,
	})
	if err != nil {
		t.Fatalf("Create ALL_TENANTS: %v", err)
	}
	superOnly, err := svc.Create(ctx, CreateAvisoInput{
		Titulo:       "Só super",
		AudienceTipo: AvisoAudienceSuperAdmins,
		CreatedBy:    superID,
	})
	if err != nil {
		t.Fatalf("Create SUPER_ADMINS: %v", err)
	}
	estOnly, err := svc.Create(ctx, CreateAvisoInput{
		Titulo:             "Só salão",
		AudienceTipo:       AvisoAudienceEstabelecimentos,
		EstabelecimentoIDs: []string{estID},
		CreatedBy:          superID,
	})
	if err != nil {
		t.Fatalf("Create ESTABELECIMENTOS: %v", err)
	}

	// SUPER_ADMIN sem estabelecimento: caminho que passava $n IS NOT NULL sem cast.
	superInbox, err := svc.ListInbox(ctx, superID, "SUPER_ADMIN", nil, 20)
	if err != nil {
		t.Fatalf("ListInbox SUPER_ADMIN (nil est): %v", err)
	}
	if !inboxHas(superInbox, superOnly.ID) || inboxHas(superInbox, allTenants.ID) {
		t.Fatalf("inbox SUPER_ADMIN inesperado: %+v", superInbox.Items)
	}

	// DONA com estabelecimento: $n::uuid tipado nas duas ocorrências.
	donaInbox, err := svc.ListInbox(ctx, donaID, "DONA", &estID, 20)
	if err != nil {
		t.Fatalf("ListInbox DONA: %v", err)
	}
	if !inboxHas(donaInbox, allTenants.ID) || !inboxHas(donaInbox, estOnly.ID) || inboxHas(donaInbox, superOnly.ID) {
		t.Fatalf("inbox DONA inesperado: %+v", donaInbox.Items)
	}
	if donaInbox.UnreadCount < 2 {
		t.Fatalf("unread_count=%d; want >= 2", donaInbox.UnreadCount)
	}

	marked, err := svc.MarkRead(ctx, donaID, "DONA", allTenants.ID, &estID)
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if marked.ReadAt == nil {
		t.Fatal("MarkRead sem read_at")
	}
	after, err := svc.ListInbox(ctx, donaID, "DONA", &estID, 20)
	if err != nil {
		t.Fatalf("ListInbox após MarkRead: %v", err)
	}
	if after.UnreadCount != donaInbox.UnreadCount-1 {
		t.Fatalf("unread_count após read: got %d want %d", after.UnreadCount, donaInbox.UnreadCount-1)
	}

	// Cross-audience: DONA não marca aviso SUPER_ADMINS.
	if _, err := svc.MarkRead(ctx, donaID, "DONA", superOnly.ID, &estID); err != ErrAvisoNaoEncontrado {
		t.Fatalf("MarkRead cross-audience: err=%v want ErrAvisoNaoEncontrado", err)
	}
}

func newAvisosInboxPostgres(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		dsn = "postgresql://postgres:glow_secure_pwd_2026@localhost:5435/agenda_glow_prod?sslmode=disable"
	}

	admin, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Postgres indisponível (%v)", err)
	}
	t.Cleanup(func() { _ = admin.Close() })

	schema := fmt.Sprintf("avisos_inbox_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatalf("criar schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`); err != nil {
			t.Errorf("drop schema: %v", err)
		}
	})

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()

	db, err := sqlx.Connect("postgres", parsed.String())
	if err != nil {
		t.Fatalf("conectar schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	avisosMustExec(t, db, avisosInboxTestDDL)
	return db
}

func avisosMustExec(t *testing.T, db *sqlx.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec: %v\nSQL: %s", err, q)
	}
}

func inboxHas(in *InboxResult, id string) bool {
	if in == nil {
		return false
	}
	for _, item := range in.Items {
		if item.ID == id {
			return true
		}
	}
	return false
}

// Recorte mínimo das tabelas tocadas pelo inbox (espelha 000025 + FKs).
const avisosInboxTestDDL = `
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    nome TEXT NOT NULL,
    role TEXT NOT NULL
);

CREATE TABLE estabelecimentos (
    id UUID PRIMARY KEY,
    nome_comercial TEXT NOT NULL
);

CREATE TABLE avisos_globais (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo VARCHAR(200) NOT NULL,
    corpo TEXT,
    severidade VARCHAR(20) NOT NULL DEFAULT 'INFO'
        CHECK (severidade IN ('INFO', 'WARNING', 'CRITICAL')),
    audience_tipo VARCHAR(30) NOT NULL
        CHECK (audience_tipo IN ('SUPER_ADMINS', 'ALL_TENANTS', 'ESTABELECIMENTOS')),
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL REFERENCES users (id),
    criado_em TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP
);

CREATE TABLE aviso_estabelecimentos (
    aviso_id UUID NOT NULL REFERENCES avisos_globais (id) ON DELETE CASCADE,
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    PRIMARY KEY (aviso_id, estabelecimento_id)
);

CREATE TABLE aviso_leituras (
    aviso_id UUID NOT NULL REFERENCES avisos_globais (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    read_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (aviso_id, user_id)
);
`
