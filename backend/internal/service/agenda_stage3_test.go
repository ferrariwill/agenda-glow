//go:build integration

package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/agendaglow/agendaglow/internal/service"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func hora(s string) *string { return &s }

// cenarioStage3 monta um salão isolado com jornada 09:00–18:00 e almoço 12:00–13:00.
type cenarioStage3 struct {
	estID     string
	profID    string
	servicoID string
	agenda    *service.AgendaService
	dia       time.Time
}

func montarCenarioStage3(t *testing.T, db *sqlx.DB, duracaoServico int) cenarioStage3 {
	t.Helper()
	ctx := context.Background()

	planoSvc := service.NewPlanoSaasService(db)
	estabSvc := service.NewEstabelecimentoService(db)
	profSvc := service.NewProfissionalService(db)
	espSvc := service.NewEspecialidadeService(db)
	procSvc := service.NewProcedimentoService(db)

	planoID, err := planoSvc.CreateSaasPlan(ctx, "Plano QA Stage3", 97.00, 2)
	if err != nil {
		t.Fatalf("criar plano SaaS: %v", err)
	}
	// Cleanup é LIFO: o plano cai depois do estabelecimento que o referencia.
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM planos_saas WHERE id = $1`, planoID); err != nil {
			t.Logf("limpar plano %s: %v", planoID, err)
		}
	})

	estID, _, err := estabSvc.RegisterEstablishment(ctx, "QA Stage3 Salao", uniqueSlug(t))
	if err != nil {
		t.Fatalf("cadastrar estabelecimento: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM estabelecimentos WHERE id = $1`, estID); err != nil {
			t.Logf("limpar estabelecimento %s: %v", estID, err)
		}
	})

	if err := planoSvc.AssignPlanToEstablishment(ctx, estID, planoID, 12); err != nil {
		t.Fatalf("atribuir plano ao estabelecimento: %v", err)
	}

	espID, err := espSvc.CreateEspecialidade(ctx, estID, "Manicure")
	if err != nil {
		t.Fatalf("cadastrar especialidade: %v", err)
	}
	profID, err := profSvc.CreateProfessional(ctx, estID, "Cláudia Stage3", espID, 40)
	if err != nil {
		t.Fatalf("cadastrar profissional: %v", err)
	}
	servicoID, err := procSvc.CreateService(ctx, estID, "Fazer Unhas", 50, duracaoServico)
	if err != nil {
		t.Fatalf("cadastrar serviço: %v", err)
	}

	// Dia útil arbitrário no futuro, no fuso local — igual ao que o handler produz.
	dia := time.Date(2026, time.September, 15, 0, 0, 0, 0, time.Local)

	if err := profSvc.SetProfessionalHours(ctx, profID, []service.ExpedienteInput{{
		DiaSemana:      int(dia.Weekday()),
		HorarioEntrada: "09:00",
		InicioAlmoco:   hora("12:00"),
		FimAlmoco:      hora("13:00"),
		HorarioSaida:   "18:00",
	}}); err != nil {
		t.Fatalf("definir expediente: %v", err)
	}

	return cenarioStage3{
		estID:     estID,
		profID:    profID,
		servicoID: servicoID,
		agenda:    service.NewAgendaService(db, service.AgendaOptions{BaseURL: "http://localhost:8081"}),
		dia:       dia,
	}
}

// DEF-S3-01: colunas TIME do expediente quebravam o scan e derrubavam o endpoint público.
func TestGetAvailableSlotsComExpediente(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	c := montarCenarioStage3(t, db, 45)

	res, err := c.agenda.CriarAgendamento(
		ctx, c.estID, "Ana Paula", "5515999100001", c.profID, c.servicoID, nil,
		c.dia.Add(11*time.Hour), service.OrigemInterno, false,
	)
	if err != nil {
		t.Fatalf("criar agendamento das 11:00: %v", err)
	}
	if res.Status != "AGENDADO" {
		t.Fatalf("status do agendamento base: got %q, want AGENDADO", res.Status)
	}

	slots, err := c.agenda.GetAvailableSlots(ctx, c.estID, c.profID, c.dia, c.servicoID, nil)
	if err != nil {
		t.Fatalf("buscar slots com expediente cadastrado: %v", err)
	}

	want := []string{
		"09:00", "09:30", "10:00",
		"13:00", "13:30", "14:00", "14:30",
		"15:00", "15:30", "16:00", "16:30", "17:00",
	}
	if len(slots) != len(want) {
		t.Fatalf("slots: got %v, want %v", slots, want)
	}
	for i := range want {
		if slots[i] != want[i] {
			t.Fatalf("slots: got %v, want %v", slots, want)
		}
	}
}

// DEF-S3-02: minutos_invadidos somava o offset do fuso (195 em vez de 15).
func TestCriarAgendamentoMinutosInvadidosSemOffset(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	c := montarCenarioStage3(t, db, 45)

	if _, err := c.agenda.CriarAgendamento(
		ctx, c.estID, "Ana Paula", "5515999100001", c.profID, c.servicoID, nil,
		c.dia.Add(11*time.Hour), service.OrigemInterno, false,
	); err != nil {
		t.Fatalf("criar agendamento das 11:00: %v", err)
	}

	// 10:30 + 45 min termina 11:15 → invade 15 min do atendimento das 11:00.
	res, err := c.agenda.CriarAgendamento(
		ctx, c.estID, "Beatriz", "5515999100002", c.profID, c.servicoID, nil,
		c.dia.Add(10*time.Hour+30*time.Minute), service.OrigemInterno, false,
	)
	if err != nil {
		t.Fatalf("criar encaixe das 10:30: %v", err)
	}
	if res.Status != "EM_APROVACAO" {
		t.Fatalf("status do encaixe: got %q, want EM_APROVACAO", res.Status)
	}
	if res.MinutosInvadidos != 15 {
		t.Fatalf("minutos_invadidos: got %d, want 15", res.MinutosInvadidos)
	}

	var persistido int
	if err := db.Get(&persistido, `SELECT minutos_invadidos FROM agendamentos WHERE id = $1`, res.ID); err != nil {
		t.Fatalf("ler minutos_invadidos persistido: %v", err)
	}
	if persistido != 15 {
		t.Fatalf("minutos_invadidos no banco: got %d, want 15", persistido)
	}
}
