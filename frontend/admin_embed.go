package frontend

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// LoadAdminConfigTemplates carrega telas de configuração interna do salão.
func LoadAdminConfigTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"money": func(v float64) string {
			return fmt.Sprintf("R$ %.2f", v)
		},
		"pct": func(v float64) string {
			return fmt.Sprintf("%.0f%%", v)
		},
		"initial": func(s string) string {
			s = strings.TrimSpace(s)
			for _, r := range s {
				return strings.ToUpper(string(r))
			}
			return "?"
		},
		"tipoLabel": func(tipo string) string {
			switch strings.ToUpper(tipo) {
			case "ENTRADA":
				return "Entrada"
			case "CUSTO_FIXO":
				return "Custo Fixo"
			case "CUSTO_VARIAVEL":
				return "Custo Variável"
			default:
				return tipo
			}
		},
		"tipoBadge": func(tipo string) string {
			switch strings.ToUpper(tipo) {
			case "ENTRADA":
				return "bg-emerald-500/15 text-emerald-300 border-emerald-500/30"
			case "CUSTO_FIXO":
				return "bg-rose-500/15 text-rose-300 border-rose-500/30"
			case "CUSTO_VARIAVEL":
				return "bg-amber-500/15 text-amber-300 border-amber-500/30"
			default:
				return "bg-slate-500/15 text-slate-300 border-slate-500/30"
			}
		},
		"valorClass": func(tipo string) string {
			if strings.ToUpper(tipo) == "ENTRADA" {
				return "text-emerald-400 font-semibold"
			}
			return "text-rose-400 font-semibold"
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("02/01/2006 15:04")
		},
		"eq": func(a, b string) bool { return a == b },
	}

	tmpl, err := template.New("admin").Funcs(funcMap).ParseFS(templatesFS,
		"templates/admin_common.html",
		"templates/config_servicos.html",
		"templates/config_equipe.html",
		"templates/caixa_fluxo.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse admin config templates: %w", err)
	}
	return tmpl, nil
}
