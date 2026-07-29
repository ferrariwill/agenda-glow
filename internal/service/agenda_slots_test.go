package service

import (
	"testing"
	"time"
)

// fusoSalao reproduz o fuso da aplicação (UTC−3) sem depender do TZ da máquina.
var fusoSalao = time.FixedZone("BRT", -3*60*60)

// lidoDoBanco simula o que o driver devolve para uma coluna TIMESTAMP: hora de
// parede correta, mas localizada em UTC.
func lidoDoBanco(ano int, mes time.Month, dia, hora, minuto int) time.Time {
	return time.Date(ano, mes, dia, hora, minuto, 0, 0, time.UTC)
}

func ponteiro(s string) *string { return &s }

func TestNormalizarFusoDoBancoPreservaHoraDeParede(t *testing.T) {
	origem := lidoDoBanco(2026, time.September, 15, 11, 0)
	got := normalizarFusoDoBanco(origem, fusoSalao)

	if got.Format("2006-01-02 15:04") != "2026-09-15 11:00" {
		t.Fatalf("hora de parede alterada: got %s", got.Format("2006-01-02 15:04"))
	}
	if got.Location() != fusoSalao {
		t.Fatalf("fuso não realinhado: got %s", got.Location())
	}
}

// DEF-S3-02: AGENDADO às 11:00; novo atendimento 10:30 + 45 min invade 15 min.
func TestCalcularMinutosInvadidosIgnoraOffsetDoFuso(t *testing.T) {
	fim := time.Date(2026, time.September, 15, 11, 15, 0, 0, fusoSalao)
	proximoInicio := lidoDoBanco(2026, time.September, 15, 11, 0)

	if got := calcularMinutosInvadidos(fim, proximoInicio); got != 15 {
		t.Fatalf("minutos_invadidos: got %d, want 15", got)
	}
}

func TestCalcularMinutosInvadidosNuncaMenorQueUm(t *testing.T) {
	fim := time.Date(2026, time.September, 15, 11, 0, 0, 0, fusoSalao)
	proximoInicio := lidoDoBanco(2026, time.September, 15, 11, 0)

	if got := calcularMinutosInvadidos(fim, proximoInicio); got != 1 {
		t.Fatalf("minutos_invadidos: got %d, want 1", got)
	}
}

// DEF-S3-01: horários vêm de colunas TIME via ::text ("HH:MM:SS").
func TestFiltrarSlotsDisponiveisRespeitaAlmocoEAgendamentos(t *testing.T) {
	dia := time.Date(2026, time.September, 15, 0, 0, 0, 0, fusoSalao)
	expediente := &expedienteDia{
		HorarioEntrada: "09:00:00",
		InicioAlmoco:   ponteiro("12:00:00"),
		FimAlmoco:      ponteiro("13:00:00"),
		HorarioSaida:   "18:00:00",
	}
	ocupados := []intervaloAgendado{{
		Inicio: normalizarFusoDoBanco(lidoDoBanco(2026, time.September, 15, 11, 0), fusoSalao),
		Fim:    normalizarFusoDoBanco(lidoDoBanco(2026, time.September, 15, 11, 45), fusoSalao),
	}}

	got, err := filtrarSlotsDisponiveis(dia, 60, ocupados, expediente)
	if err != nil {
		t.Fatalf("filtrar slots: %v", err)
	}

	want := []string{
		"09:00", "09:30", "10:00",
		"13:00", "13:30", "14:00", "14:30",
		"15:00", "15:30", "16:00", "16:30", "17:00",
	}
	if len(got) != len(want) {
		t.Fatalf("slots: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slots: got %v, want %v", got, want)
		}
	}
}

func TestFiltrarSlotsDisponiveisSemAlmocoRespeitaSaida(t *testing.T) {
	dia := time.Date(2026, time.September, 15, 0, 0, 0, 0, fusoSalao)
	expediente := &expedienteDia{
		HorarioEntrada: "09:00",
		HorarioSaida:   "11:00",
	}

	got, err := filtrarSlotsDisponiveis(dia, 45, nil, expediente)
	if err != nil {
		t.Fatalf("filtrar slots: %v", err)
	}

	want := []string{"09:00", "09:30", "10:00"}
	if len(got) != len(want) {
		t.Fatalf("slots: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slots: got %v, want %v", got, want)
		}
	}
}
