package frontend

import (
	"bytes"
	"testing"

	"github.com/agendaglow/agendaglow/internal/service"
)

func TestDashboardPageRenders(t *testing.T) {
	tmpl, err := LoadDashboardTemplates()
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}

	data := &service.DashboardGerencial{
		EstabelecimentoNome: "Estúdio Glow",
		PeriodoLabel:        "Junho 2026",
		StartDate:           "2026-06-01",
		EndDate:             "2026-06-30",
		Relatorio: &service.RelatorioFinanceiro{
			TotalEntradas:           1000,
			TotalComissoesPendentes: 200,
			TotalCustosFixos:        150,
			LucroLiquidoEstimado:    650,
		},
		Equipe: []service.DesempenhoProfissional{
			{ID: "p1", Nome: "Cláudia", Especialidade: "Manicure", ServicosMes: 3, ComissaoPendente: 50},
		},
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "dashboard_page", data); err != nil {
		t.Fatalf("render dashboard_page: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("empty output")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Estúdio Glow")) {
		t.Fatalf("missing establishment name in output")
	}
}
