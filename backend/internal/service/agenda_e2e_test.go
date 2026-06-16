//go:build integration

package service_test

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agendaglow/agendaglow/internal/service"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// captureMailer registra e-mails enviados pelo fluxo de aprovação de encaixe.
type captureMailer struct {
	mu   sync.Mutex
	sent []capturedEmail
}

type capturedEmail struct {
	To      string
	Subject string
	Body    string
}

func (m *captureMailer) SendHTML(_ context.Context, to, subject, htmlBody string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, capturedEmail{To: to, Subject: subject, Body: htmlBody})
	return nil
}

func (m *captureMailer) last() (capturedEmail, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return capturedEmail{}, false
	}
	return m.sent[len(m.sent)-1], true
}

func (m *captureMailer) waitFor(count int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		n := len(m.sent)
		m.mu.Unlock()
		if n >= count {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

type agendamentoDB struct {
	ID             string    `db:"id"`
	Status         string    `db:"status"`
	DataHoraInicio time.Time `db:"data_hora_inicio"`
	DataHoraFim    time.Time `db:"data_hora_fim"`
}

type lancamentoDB struct {
	Tipo  string  `db:"tipo"`
	Valor float64 `db:"valor"`
}

func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("defina TEST_DATABASE_URL (ex.: postgres://postgres:glow_secure_pwd_2026@localhost:5435/agenda_glow_prod?sslmode=disable)")
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		if strings.Contains(err.Error(), "connect") || strings.Contains(err.Error(), "refused") {
			t.Skipf("banco de testes indisponível (subir: docker compose up postgres-glow -d): %v", err)
		}
		t.Fatalf("conectar ao banco de testes: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func uniqueSlug(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("e2e-%d", time.Now().UnixNano())
}

func assertMoneyEqual(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.009 {
		t.Fatalf("%s: valor %.2f, esperado %.2f", label, got, want)
	}
}

func loadAgendamento(t *testing.T, db *sqlx.DB, id string) agendamentoDB {
	t.Helper()
	var row agendamentoDB
	const q = `SELECT id, status, data_hora_inicio, data_hora_fim FROM agendamentos WHERE id = $1`
	if err := db.Get(&row, q, id); err != nil {
		t.Fatalf("buscar agendamento %s: %v", id, err)
	}
	return row
}

func setProfissionalEmail(t *testing.T, db *sqlx.DB, profissionalID, email string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE profissionais SET email = $1 WHERE id = $2`, email, profissionalID); err != nil {
		t.Fatalf("atualizar e-mail da profissional: %v", err)
	}
}

// TestAgendaGlowJornadaCompleta executa a jornada SaaS → agendamento → encaixe → financeiro.
func TestAgendaGlowJornadaCompleta(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	planoSvc := service.NewPlanoSaasService(db)
	estabSvc := service.NewEstabelecimentoService(db)
	profSvc := service.NewProfissionalService(db)
	procSvc := service.NewProcedimentoService(db)
	finSvc := service.NewFinanceiroService(db)

	mailer := &captureMailer{}
	agendaSvc := service.NewAgendaService(db, service.AgendaOptions{
		BaseURL: "http://localhost:8081",
		Mailer:  mailer,
	})

	dia := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	as14h := func() time.Time { return dia.Add(14 * time.Hour) }
	as15h := func() time.Time { return dia.Add(15 * time.Hour) }
	as1545 := func() time.Time { return dia.Add(15*time.Hour + 45*time.Minute) }
	as16h := func() time.Time { return dia.Add(16 * time.Hour) }

	// -------------------------------------------------------------------------
	// ETAPA 1 — Configuração SaaS
	// -------------------------------------------------------------------------
	t.Log("ETAPA 1: plano SaaS + estabelecimento + assinatura")

	planoID, err := planoSvc.CreateSaasPlan(ctx, "Plano Solo", 97.00, 1)
	if err != nil {
		t.Fatalf("criar plano SaaS: %v", err)
	}

	estID, slug, err := estabSvc.RegisterEstablishment(ctx, "Estúdio Glow Salto", uniqueSlug(t))
	if err != nil {
		t.Fatalf("cadastrar estabelecimento: %v", err)
	}

	if err := planoSvc.AssignPlanToEstablishment(ctx, estID, planoID, 12); err != nil {
		t.Fatalf("atribuir plano ao estabelecimento: %v", err)
	}

	var dataVencimento time.Time
	if err := db.GetContext(ctx, &dataVencimento, `
SELECT data_vencimento FROM assinaturas_estabelecimentos WHERE estabelecimento_id = $1
`, estID); err != nil {
		t.Fatalf("consultar vencimento da assinatura: %v", err)
	}

	esperadoVencimento := time.Now().UTC().AddDate(0, 12, 0).Truncate(24 * time.Hour)
	gotVenc := dataVencimento.UTC().Truncate(24 * time.Hour)
	if !gotVenc.Equal(esperadoVencimento) {
		t.Fatalf("data de vencimento: got %s, want %s", gotVenc.Format("2006-01-02"), esperadoVencimento.Format("2006-01-02"))
	}

	plano, err := planoSvc.BuscarPlanoSaasPorID(ctx, planoID)
	if err != nil {
		t.Fatalf("buscar plano: %v", err)
	}
	if plano.LimiteProfissionais != 1 {
		t.Fatalf("limite de profissionais: got %d, want 1", plano.LimiteProfissionais)
	}
	_ = slug

	// -------------------------------------------------------------------------
	// ETAPA 2 — Configuração do salão
	// -------------------------------------------------------------------------
	t.Log("ETAPA 2: profissional, serviço base e adicional")

	profID, err := profSvc.CreateProfessional(ctx, estID, "Cláudia - Manicure", "Manicure", 40)
	if err != nil {
		t.Fatalf("cadastrar profissional: %v", err)
	}
	setProfissionalEmail(t, db, profID, "claudia.encaixe@test.local")

	servicoID, err := procSvc.CreateService(ctx, estID, "Fazer Unhas", 50, 45)
	if err != nil {
		t.Fatalf("cadastrar serviço: %v", err)
	}

	adicionalID, err := procSvc.CreateServiceAdditional(ctx, estID, servicoID, "Alongamento em Gel", 80, 60)
	if err != nil {
		t.Fatalf("cadastrar adicional: %v", err)
	}

	// -------------------------------------------------------------------------
	// ETAPA 3 — Agendamento de sucesso (serviço + adicional → 105 min)
	// -------------------------------------------------------------------------
	t.Log("ETAPA 3: agendamento completo das 14:00 às 15:45")

	res1, err := agendaSvc.CriarAgendamento(
		ctx,
		estID,
		"Ana Paula",
		"5515999000001",
		profID,
		servicoID,
		[]string{adicionalID},
		as14h(),
		service.OrigemExterno,
	)
	if err != nil {
		t.Fatalf("criar primeiro agendamento: %v", err)
	}
	if res1.Status != "AGENDADO" {
		t.Fatalf("status do 1º agendamento: got %q, want AGENDADO", res1.Status)
	}

	ag1 := loadAgendamento(t, db, res1.ID)
	if !ag1.DataHoraInicio.UTC().Equal(as14h()) {
		t.Fatalf("início: got %v, want 14:00 UTC", ag1.DataHoraInicio.UTC())
	}
	if !ag1.DataHoraFim.UTC().Equal(as1545()) {
		t.Fatalf("término: got %v, want 15:45 UTC (45+60 min)", ag1.DataHoraFim.UTC())
	}

	// -------------------------------------------------------------------------
	// ETAPA 4 — Colisão parcial + e-mail + aprovação
	// -------------------------------------------------------------------------
	t.Log("ETAPA 4: colisão parcial, e-mail de encaixe e approve")

	// Início às 15:00 cai dentro do 1º atendimento (14:00–15:45) → colisão total.
	_, err = agendaSvc.CriarAgendamento(
		ctx, estID, "Beatriz", "5515999000002", profID, servicoID, nil, as15h(), service.OrigemExterno,
	)
	if err == nil {
		t.Fatal("esperava erro de colisão total ao agendar às 15:00 sobre intervalo 14:00–15:45")
	}
	if !strings.Contains(err.Error(), "já possui um atendimento") {
		t.Fatalf("erro inesperado para colisão às 15:00: %v", err)
	}

	// Próxima cliente às 16:00 — ancora a sobreposição parcial de término.
	resAnchor, err := agendaSvc.CriarAgendamento(
		ctx, estID, "Carla", "5515999000003", profID, servicoID, nil, as16h(), service.OrigemExterno,
	)
	if err != nil {
		t.Fatalf("criar agendamento ancora 16:00: %v", err)
	}
	if resAnchor.Status != "AGENDADO" {
		t.Fatalf("status ancora: got %q, want AGENDADO", resAnchor.Status)
	}

	// Encaixe às 15:45 (45 min) invade 30 min do horário das 16:00 → EM_APROVACAO.
	res2, err := agendaSvc.CriarAgendamento(
		ctx, estID, "Beatriz", "5515999000002", profID, servicoID, nil, as1545(), service.OrigemExterno,
	)
	if err != nil {
		t.Fatalf("criar encaixe parcial: %v", err)
	}
	if res2.Status != "EM_APROVACAO" {
		t.Fatalf("status encaixe: got %q, want EM_APROVACAO", res2.Status)
	}
	if res2.MinutosInvadidos < 1 {
		t.Fatalf("minutos invadidos: got %d, want > 0", res2.MinutosInvadidos)
	}

	if !mailer.waitFor(1, 3*time.Second) {
		t.Fatal("e-mail de aprovação de encaixe não foi disparado dentro do timeout")
	}
	email, ok := mailer.last()
	if !ok {
		t.Fatal("captureMailer vazio após encaixe EM_APROVACAO")
	}
	if email.To != "claudia.encaixe@test.local" {
		t.Fatalf("e-mail enviado para %q, want claudia.encaixe@test.local", email.To)
	}
	if !strings.Contains(email.Subject, "Encaixe pendente") {
		t.Fatalf("assunto inesperado: %q", email.Subject)
	}
	if !strings.Contains(email.Body, "Beatriz") || !strings.Contains(email.Body, "approve") {
		t.Fatalf("corpo do e-mail não contém dados esperados do encaixe")
	}

	if err := agendaSvc.ApproveAppointment(ctx, res2.ID); err != nil {
		t.Fatalf("aprovar encaixe: %v", err)
	}
	ag2 := loadAgendamento(t, db, res2.ID)
	if ag2.Status != "AGENDADO" {
		t.Fatalf("status após approve: got %q, want AGENDADO", ag2.Status)
	}

	// -------------------------------------------------------------------------
	// ETAPA 5 — Conclusão e fechamento financeiro
	// -------------------------------------------------------------------------
	t.Log("ETAPA 5: ConcluirAtendimento e lançamentos no fluxo_caixa")

	if err := finSvc.ConcluirAtendimento(ctx, res1.ID); err != nil {
		t.Fatalf("concluir primeiro atendimento: %v", err)
	}

	ag1Final := loadAgendamento(t, db, res1.ID)
	if ag1Final.Status != "CONCLUIDO" {
		t.Fatalf("status final: got %q, want CONCLUIDO", ag1Final.Status)
	}

	var lancamentos []lancamentoDB
	if err := db.SelectContext(ctx, &lancamentos, `
SELECT tipo, valor::float8 AS valor
FROM fluxo_caixa
WHERE estabelecimento_id = $1
ORDER BY data_transacao ASC
`, estID); err != nil {
		t.Fatalf("listar fluxo_caixa: %v", err)
	}
	if len(lancamentos) != 2 {
		t.Fatalf("lançamentos no caixa: got %d, want 2 (ENTRADA + CUSTO_VARIAVEL)", len(lancamentos))
	}

	var entrada, comissao *lancamentoDB
	for i := range lancamentos {
		switch lancamentos[i].Tipo {
		case "ENTRADA":
			entrada = &lancamentos[i]
		case "CUSTO_VARIAVEL":
			comissao = &lancamentos[i]
		}
	}
	if entrada == nil {
		t.Fatal("lançamento ENTRADA não encontrado")
	}
	if comissao == nil {
		t.Fatal("lançamento CUSTO_VARIAVEL (comissão) não encontrado")
	}

	assertMoneyEqual(t, "ENTRADA", entrada.Valor, 130.00)
	assertMoneyEqual(t, "comissão Cláudia (40%)", comissao.Valor, 52.00)

	t.Log("jornada E2E concluída com sucesso")
}
