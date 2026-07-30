//go:build integration

package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Postgres de verdade, porque a exclusividade do slot é uma propriedade do
// `FOR UPDATE OF o, r, a` — um mock não tem lock de linha para exercer. Cada
// teste roda em schema próprio e o derruba no fim.
func newEarlySlotPostgres(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("defina TEST_DATABASE_URL para executar a corrida real de aceites")
	}

	admin, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("conectar ao Postgres: %v", err)
	}
	t.Cleanup(func() { admin.Close() }) //nolint:errcheck

	schema := fmt.Sprintf("early_slot_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatalf("criar schema isolado: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`); err != nil {
			t.Errorf("remover schema isolado: %v", err)
		}
	})

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("interpretar DSN: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()

	db, err := sqlx.Connect("postgres", parsed.String())
	if err != nil {
		t.Fatalf("conectar ao schema isolado: %v", err)
	}
	// Mais de uma conexão é o ponto do teste: com pool de 1 as transações
	// serializariam no cliente e o lock do banco nunca seria exercido.
	db.SetMaxOpenConns(8)
	t.Cleanup(func() { db.Close() }) //nolint:errcheck

	if _, err := db.Exec(earlySlotTestDDL); err != nil {
		t.Fatalf("criar schema de teste: %v", err)
	}
	return db
}

// Recorte das tabelas que o aceite toca, com as mesmas colunas e restrições das
// migrações 000021/000022 que importam para a corrida.
const earlySlotTestDDL = `
CREATE TABLE estabelecimentos (
    id UUID PRIMARY KEY,
    nome_comercial TEXT NOT NULL,
    whatsapp_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    whatsapp_status TEXT NOT NULL DEFAULT 'CONECTADO'
);
CREATE TABLE profissionais (
    id UUID PRIMARY KEY,
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos(id),
    nome TEXT NOT NULL
);
CREATE TABLE clientes (
    id UUID PRIMARY KEY,
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos(id),
    nome TEXT NOT NULL,
    telefone TEXT NOT NULL
);
CREATE TABLE servicos (
    id UUID PRIMARY KEY,
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos(id),
    nome TEXT NOT NULL
);
CREATE TABLE expedientes_profissionais (
    profissional_id UUID NOT NULL REFERENCES profissionais(id),
    dia_semana INTEGER NOT NULL,
    horario_entrada TIME NOT NULL,
    horario_saida TIME NOT NULL,
    inicio_almoco TIME,
    fim_almoco TIME
);
CREATE TABLE agendamentos (
    id UUID PRIMARY KEY,
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos(id),
    cliente_id UUID NOT NULL REFERENCES clientes(id),
    servico_id UUID NOT NULL REFERENCES servicos(id),
    profissional_id UUID NOT NULL REFERENCES profissionais(id),
    data_hora_inicio TIMESTAMPTZ NOT NULL,
    data_hora_fim TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    aceita_adiantar BOOLEAN NOT NULL DEFAULT FALSE,
    aceita_adiantar_em TIMESTAMPTZ
);
CREATE TABLE rodadas_antecipacao (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos(id),
    profissional_id UUID NOT NULL REFERENCES profissionais(id),
    slot_inicio TIMESTAMPTZ NOT NULL,
    slot_fim TIMESTAMPTZ NOT NULL,
    agendamento_cancelado_id UUID NOT NULL REFERENCES agendamentos(id),
    status VARCHAR(16) NOT NULL DEFAULT 'ATIVA'
        CHECK (status IN ('ATIVA', 'PREENCHIDA', 'ESGOTADA', 'CANCELADA')),
    motivo_encerramento VARCHAR(40),
    candidato_atual_agendamento_id UUID REFERENCES agendamentos(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    CHECK (slot_fim > slot_inicio)
);
CREATE TABLE ofertas_antecipacao (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos(id),
    rodada_id UUID NOT NULL REFERENCES rodadas_antecipacao(id) ON DELETE CASCADE,
    agendamento_candidato_id UUID NOT NULL REFERENCES agendamentos(id),
    cliente_id UUID REFERENCES clientes(id),
    profissional_snapshot_id UUID NOT NULL REFERENCES profissionais(id),
    inicio_snapshot TIMESTAMPTZ NOT NULL,
    fim_snapshot TIMESTAMPTZ NOT NULL,
    posicao INTEGER NOT NULL CHECK (posicao > 0),
    status VARCHAR(16) NOT NULL DEFAULT 'PENDENTE'
        CHECK (status IN ('PENDENTE','ACEITA','RECUSADA','EXPIRADA','INVALIDADA','FALHA_ENVIO')),
    token_hash BYTEA NOT NULL,
    enviada_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expira_em TIMESTAMPTZ NOT NULL,
    respondida_em TIMESTAMPTZ,
    origem_resposta VARCHAR(16),
    external_message_id TEXT,
    tentativas_envio INTEGER NOT NULL DEFAULT 0,
    proxima_tentativa_em TIMESTAMPTZ,
    UNIQUE (rodada_id, agendamento_candidato_id),
    UNIQUE (token_hash)
);
CREATE TABLE antecipacao_auditoria (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos(id),
    rodada_id UUID NOT NULL REFERENCES rodadas_antecipacao(id) ON DELETE CASCADE,
    oferta_id UUID REFERENCES ofertas_antecipacao(id),
    evento VARCHAR(40) NOT NULL,
    horario_anterior_inicio TIMESTAMPTZ,
    horario_anterior_fim TIMESTAMPTZ,
    horario_novo_inicio TIMESTAMPTZ,
    horario_novo_fim TIMESTAMPTZ,
    origem VARCHAR(16),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

const (
	tenantUUID = "11111111-1111-1111-1111-111111111111"
	profUUID   = "22222222-2222-2222-2222-222222222222"
	servUUID   = "33333333-3333-3333-3333-333333333333"
)

// seedEarlySlotTenant deixa o salão pronto: um profissional em expediente
// integral e o agendamento cancelado que dá origem ao slot livre.
func seedEarlySlotTenant(t *testing.T, db *sqlx.DB, slotStart, slotEnd time.Time) string {
	t.Helper()

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("seed (%s): %v", q, err)
		}
	}
	exec(`INSERT INTO estabelecimentos (id, nome_comercial) VALUES ($1,'Glow')`, tenantUUID)
	exec(`INSERT INTO profissionais (id, estabelecimento_id, nome) VALUES ($1,$2,'Ana')`, profUUID, tenantUUID)
	exec(`INSERT INTO servicos (id, estabelecimento_id, nome) VALUES ($1,$2,'Corte')`, servUUID, tenantUUID)
	for dia := 0; dia < 7; dia++ {
		exec(`INSERT INTO expedientes_profissionais
			(profissional_id, dia_semana, horario_entrada, horario_saida)
			VALUES ($1,$2,'00:00','23:59')`, profUUID, dia)
	}

	cancelledID := "44444444-4444-4444-4444-444444444444"
	clientID := "55555555-5555-5555-5555-555555555555"
	exec(`INSERT INTO clientes (id, estabelecimento_id, nome, telefone)
		VALUES ($1,$2,'Cancelada','5511000000000')`, clientID, tenantUUID)
	exec(`INSERT INTO agendamentos
		(id, estabelecimento_id, cliente_id, servico_id, profissional_id,
		 data_hora_inicio, data_hora_fim, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'CANCELADO')`,
		cancelledID, tenantUUID, clientID, servUUID, profUUID, slotStart, slotEnd)
	return cancelledID
}

// addCandidate cria um cliente que optou por adiantar, com o agendamento futuro
// que ele abre mão caso aceite.
func addCandidate(t *testing.T, db *sqlx.DB, suffix string, start, end time.Time) string {
	t.Helper()

	clientID := fmt.Sprintf("66666666-6666-6666-6666-%012s", suffix)
	appointmentID := fmt.Sprintf("77777777-7777-7777-7777-%012s", suffix)
	if _, err := db.Exec(`INSERT INTO clientes (id, estabelecimento_id, nome, telefone)
		VALUES ($1,$2,$3,$4)`, clientID, tenantUUID, "Cliente "+suffix, "55119"+suffix); err != nil {
		t.Fatalf("criar cliente candidato: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO agendamentos
		(id, estabelecimento_id, cliente_id, servico_id, profissional_id,
		 data_hora_inicio, data_hora_fim, status, aceita_adiantar, aceita_adiantar_em)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'AGENDADO',TRUE,NOW())`,
		appointmentID, tenantUUID, clientID, servUUID, profUUID, start, end); err != nil {
		t.Fatalf("criar agendamento candidato: %v", err)
	}
	return appointmentID
}

func insertRound(t *testing.T, db *sqlx.DB, cancelledID string, slotStart, slotEnd time.Time) string {
	t.Helper()
	var roundID string
	if err := db.Get(&roundID, `INSERT INTO rodadas_antecipacao
		(estabelecimento_id, profissional_id, slot_inicio, slot_fim, agendamento_cancelado_id)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		tenantUUID, profUUID, slotStart, slotEnd, cancelledID); err != nil {
		t.Fatalf("criar rodada: %v", err)
	}
	return roundID
}

func insertOffer(
	t *testing.T, db *sqlx.DB, roundID, appointmentID string,
	position int, start, end, expiresAt time.Time,
) string {
	t.Helper()
	token, hash, err := newEarlySlotToken()
	if err != nil {
		t.Fatalf("gerar token: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO ofertas_antecipacao
		(estabelecimento_id, rodada_id, agendamento_candidato_id, cliente_id,
		 profissional_snapshot_id, inicio_snapshot, fim_snapshot,
		 posicao, token_hash, expira_em, tentativas_envio)
		SELECT $1,$2,$3,a.cliente_id,$4,$5,$6,$7,$8,$9,1
		FROM agendamentos a WHERE a.id=$3`,
		tenantUUID, roundID, appointmentID, profUUID, start, end,
		position, hash[:], expiresAt); err != nil {
		t.Fatalf("criar oferta: %v", err)
	}
	return token
}

func newEarlySlotServiceForPostgres(db *sqlx.DB, now time.Time) *EarlySlotService {
	svc := NewEarlySlotService(db, "")
	svc.now = func() time.Time { return now }
	svc.channelReady = func(context.Context, sqlx.QueryerContext, string) (bool, error) {
		return true, nil
	}
	svc.send = func(context.Context, WhatsAppNotificationInput) error { return nil }
	return svc
}

// Duas rodadas disputando o mesmo horário — dois cancelamentos quase simultâneos
// do mesmo profissional. Os aceites partem juntos e batem no mesmo `FOR UPDATE`:
// um reagenda, o outro precisa perder. Se os dois passassem, o profissional
// ficaria com dois atendimentos sobrepostos.
func TestPostgresConcurrentAcceptsOnOverlappingSlotsKeepOneWinner(t *testing.T) {
	db := newEarlySlotPostgres(t)
	now := time.Now().UTC().Truncate(time.Second)
	slotStart := now.Add(24 * time.Hour)
	slotEnd := slotStart.Add(time.Hour)

	cancelledID := seedEarlySlotTenant(t, db, slotStart, slotEnd)

	// A segunda rodada começa 15 minutos depois: janelas diferentes (o índice
	// único só barra slot_inicio igual) que ainda assim se sobrepõem.
	roundA := insertRound(t, db, cancelledID, slotStart, slotEnd)
	roundB := insertRound(t, db, cancelledID, slotStart.Add(15*time.Minute), slotEnd)

	candStart := now.Add(72 * time.Hour)
	candEnd := candStart.Add(45 * time.Minute)
	candA := addCandidate(t, db, "00000000000a", candStart, candEnd)
	candB := addCandidate(t, db, "00000000000b", candStart, candEnd)

	expires := now.Add(5 * time.Minute)
	tokenA := insertOffer(t, db, roundA, candA, 1, candStart, candEnd, expires)
	tokenB := insertOffer(t, db, roundB, candB, 1, candStart, candEnd, expires)

	svc := newEarlySlotServiceForPostgres(db, now)

	type outcome struct {
		result *EarlySlotAcceptResult
		err    error
	}
	outcomes := make([]outcome, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, token := range []string{tokenA, tokenB} {
		wg.Add(1)
		go func(idx int, tok string) {
			defer wg.Done()
			<-start
			r, err := svc.Accept(context.Background(), tok, "WEB")
			outcomes[idx] = outcome{result: r, err: err}
		}(i, token)
	}
	close(start)
	wg.Wait()

	winners := 0
	for _, o := range outcomes {
		if o.err == nil && o.result != nil && o.result.Status == "ACEITA" {
			winners++
			continue
		}
		if o.err == nil {
			t.Fatalf("aceite sem erro e sem resultado ACEITA: %#v", o.result)
		}
	}
	if winners != 1 {
		t.Fatalf("aceites vencedores = %d, esperado exatamente 1 (erros: %v, %v)",
			winners, outcomes[0].err, outcomes[1].err)
	}

	// A prova que interessa ao salão: nenhuma sobreposição na agenda.
	var overlaps int
	if err := db.Get(&overlaps, `
SELECT COUNT(*) FROM agendamentos a JOIN agendamentos b
  ON a.profissional_id = b.profissional_id AND a.id < b.id
WHERE a.status IN ('AGENDADO','CONFIRMADO') AND b.status IN ('AGENDADO','CONFIRMADO')
  AND a.data_hora_inicio < b.data_hora_fim AND a.data_hora_fim > b.data_hora_inicio`); err != nil {
		t.Fatalf("checar sobreposição: %v", err)
	}
	if overlaps != 0 {
		t.Fatalf("agenda ficou com %d sobreposições após a corrida", overlaps)
	}

	var accepted int
	if err := db.Get(&accepted, `SELECT COUNT(*) FROM ofertas_antecipacao WHERE status='ACEITA'`); err != nil {
		t.Fatalf("contar ofertas aceitas: %v", err)
	}
	if accepted != 1 {
		t.Fatalf("ofertas ACEITA = %d, esperado 1", accepted)
	}
}

// Mesmo token clicado duas vezes ao mesmo tempo — o caso real do cliente que
// toca no link duas vezes, ou responde no WhatsApp e na web. O reagendamento
// pode ser relatado duas vezes, mas só pode ser aplicado uma.
func TestPostgresDoubleAcceptOfSameOfferAppliesRescheduleOnce(t *testing.T) {
	db := newEarlySlotPostgres(t)
	now := time.Now().UTC().Truncate(time.Second)
	slotStart := now.Add(24 * time.Hour)
	slotEnd := slotStart.Add(time.Hour)

	cancelledID := seedEarlySlotTenant(t, db, slotStart, slotEnd)
	roundID := insertRound(t, db, cancelledID, slotStart, slotEnd)

	candStart := now.Add(72 * time.Hour)
	candEnd := candStart.Add(45 * time.Minute)
	candidateID := addCandidate(t, db, "00000000000c", candStart, candEnd)
	token := insertOffer(t, db, roundID, candidateID, 1, candStart, candEnd, now.Add(5*time.Minute))

	svc := newEarlySlotServiceForPostgres(db, now)

	errs := make([]error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			_, err := svc.Accept(context.Background(), token, "WEB")
			errs[idx] = err
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("aceite %d falhou; clique repetido deve ser idempotente: %v", i, err)
		}
	}

	var auditados int
	if err := db.Get(&auditados, `SELECT COUNT(*) FROM antecipacao_auditoria WHERE evento='ACEITE'`); err != nil {
		t.Fatalf("contar auditoria: %v", err)
	}
	if auditados != 1 {
		t.Fatalf("eventos de ACEITE = %d, esperado 1 — o reagendamento foi aplicado duas vezes", auditados)
	}

	var start2, end2 time.Time
	if err := db.QueryRow(`SELECT data_hora_inicio, data_hora_fim FROM agendamentos WHERE id=$1`,
		candidateID).Scan(&start2, &end2); err != nil {
		t.Fatalf("reler candidato: %v", err)
	}
	if !start2.UTC().Equal(slotStart) {
		t.Fatalf("início = %s, esperado %s", start2.UTC(), slotStart)
	}
	if got := end2.Sub(start2); got != 45*time.Minute {
		t.Fatalf("duração = %s, esperado 45m (o slot não pode esticar o atendimento)", got)
	}
}
