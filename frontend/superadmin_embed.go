package frontend

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// LoadSuperAdminTemplates carrega o painel exclusivo do Super Admin.
func LoadSuperAdminTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"money": func(v float64) string {
			return fmt.Sprintf("R$ %.2f", v)
		},
		"moneyMonthly": func(v float64) string {
			return fmt.Sprintf("R$ %.2f/mês", v)
		},
		"shortID": func(id string) string {
			if len(id) > 8 {
				return id[:8] + "…"
			}
			return id
		},
		"formatDate": func(t time.Time) string {
			return t.Format("02/01/2006")
		},
		"formatDatePtr": func(t *time.Time) string {
			if t == nil {
				return "—"
			}
			return t.Format("02/01/2006")
		},
		"publicURL": func(baseURL, slug string) string {
			baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
			return baseURL + "/" + slug
		},
		"statusBadge": func(status string) string {
			switch strings.ToUpper(status) {
			case "ATIVO":
				return "bg-emerald-500/15 text-emerald-300 border-emerald-400/40 shadow-[0_0_12px_rgba(52,211,153,0.15)]"
			case "VENCIDO":
				return "bg-amber-500/15 text-amber-300 border-amber-400/40 shadow-[0_0_12px_rgba(251,191,36,0.15)]"
			case "SUSPENSO":
				return "bg-rose-500/15 text-rose-300 border-rose-400/40 shadow-[0_0_12px_rgba(251,113,133,0.15)]"
			default:
				return "bg-violet-500/15 text-violet-300 border-violet-400/40"
			}
		},
		"eq": func(a, b string) bool { return a == b },
		"derefStr": func(s *string) string {
			if s == nil {
				return "—"
			}
			return *s
		},
	}

	tmpl, err := template.New("superadmin").Funcs(funcMap).ParseFS(templatesFS,
		"templates/superadmin/common.html",
		"templates/superadmin/dashboard.html",
		"templates/superadmin/planos.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse superadmin templates: %w", err)
	}
	return tmpl, nil
}
