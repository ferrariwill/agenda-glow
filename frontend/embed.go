package frontend

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
)

//go:embed templates/*
var templatesFS embed.FS

// LoadDashboardTemplates carrega o painel gerencial e partials HTMX embutidos.
func LoadDashboardTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"money": func(v float64) string {
			return fmt.Sprintf("R$ %.2f", v)
		},
		"pctOf": func(part, total float64) int {
			if total <= 0 {
				return 0
			}
			pct := int(part / total * 100)
			if pct > 100 {
				return 100
			}
			return pct
		},
		"initial": func(s string) string {
			s = strings.TrimSpace(s)
			for _, r := range s {
				return strings.ToUpper(string(r))
			}
			return "?"
		},
	}

	tmpl, err := template.New("dashboard_dona.html").Funcs(funcMap).ParseFS(templatesFS, "templates/dashboard_dona.html")
	if err != nil {
		return nil, fmt.Errorf("parse dashboard template: %w", err)
	}
	return tmpl, nil
}

// LoadProfessionalDashboardTemplates carrega o painel mobile da profissional parceira.
func LoadProfessionalDashboardTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"money": func(v float64) string {
			return fmt.Sprintf("R$ %.2f", v)
		},
		"initial": func(s string) string {
			s = strings.TrimSpace(s)
			for _, r := range s {
				return strings.ToUpper(string(r))
			}
			return "?"
		},
		"statusBadge": func(status string) string {
			switch strings.ToUpper(status) {
			case "AGENDADO":
				return "bg-sky-500/15 text-sky-300 border-sky-500/30"
			case "CONFIRMADO":
				return "bg-violet-500/15 text-violet-300 border-violet-500/30"
			case "CONCLUIDO":
				return "bg-emerald-500/15 text-emerald-300 border-emerald-500/30"
			case "CANCELADO":
				return "bg-rose-500/15 text-rose-300 border-rose-500/30"
			case "EM_APROVACAO":
				return "bg-amber-500/15 text-amber-300 border-amber-500/30"
			default:
				return "bg-slate-500/15 text-slate-300 border-slate-500/30"
			}
		},
		"statusLabel": func(status string) string {
			switch strings.ToUpper(status) {
			case "AGENDADO":
				return "Agendado"
			case "CONFIRMADO":
				return "Confirmado"
			case "CONCLUIDO":
				return "Concluído"
			case "CANCELADO":
				return "Cancelado"
			case "EM_APROVACAO":
				return "Em aprovação"
			default:
				return status
			}
		},
	}

	tmpl, err := template.New("dashboard_profissional.html").Funcs(funcMap).ParseFS(templatesFS, "templates/dashboard_profissional.html")
	if err != nil {
		return nil, fmt.Errorf("parse professional dashboard template: %w", err)
	}
	return tmpl, nil
}
