package frontend

import (
	"bytes"
	"testing"

	"github.com/agendaglow/agendaglow/internal/service"
)

func TestEquipePageRenders(t *testing.T) {
	tmpl, err := LoadAdminConfigTemplates()
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}

	data := struct {
		EstabelecimentoNome string
		NavActive           string
		Profissionais       []service.Profissional
		Especialidades      []service.Especialidade
		Limite              service.StatusLimiteEquipe
	}{
		EstabelecimentoNome: "Test",
		NavActive:           "equipe",
		Profissionais: []service.Profissional{
			{
				ID:                  "p1",
				Nome:                "Ana",
				EspecialidadeID:     "e1",
				Especialidade:       "Manicure",
				ComissaoPorcentagem: 40,
				Ativo:               true,
			},
		},
		Especialidades: []service.Especialidade{
			{ID: "e1", Nome: "Manicure", Ativo: true},
		},
		Limite: service.StatusLimiteEquipe{LimiteProfissionais: 5, TotalAtivos: 1},
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "config_equipe_page", data); err != nil {
		t.Fatalf("render equipe page: %v", err)
	}
}
