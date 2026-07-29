package handler

import (
	"html/template"
	"net/http"
	"strings"
)

type loginPageData struct {
	Title       string
	Subtitle    string
	Role        string
	PostAction  string
	ShowError   bool
	DevHint     string
}

var loginPageTmpl = template.Must(template.New("login").Parse(loginPageHTML))

// LoginPage GET /login, /login/superadmin, /login/dona, /login/profissional
func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "/login" {
		h.renderLoginHub(w)
		return
	}

	role := roleFromLoginPath(path)
	data := loginPageData{
		ShowError:  r.URL.Query().Get("error") == "1",
		PostAction: path,
	}

	switch role {
	case "SUPER_ADMIN":
		data.Title = "Super Admin"
		data.Subtitle = "Gestão da plataforma AgendaGlow"
		data.Role = "SUPER_ADMIN"
		data.DevHint = "Dev: ferrariwill@gmail.com"
	case "DONA":
		data.Title = "Dona do Salão"
		data.Subtitle = "Painel gerencial, finanças e configuração"
		data.Role = "DONA"
		data.DevHint = "Dev: dona@glow.local"
	case "PROFISSIONAL":
		data.Title = "Profissional Parceira"
		data.Subtitle = "Sua agenda e atendimentos do dia"
		data.Role = "PROFISSIONAL"
		data.DevHint = "Dev: claudia@glow.local"
	default:
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := loginPageTmpl.Execute(w, data); err != nil {
		http.Error(w, "Erro ao renderizar login", http.StatusInternalServerError)
	}
}

func (h *AuthHandler) renderLoginHub(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(loginHubHTML))
}

const loginHubHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>AgendaGlow · Entrar</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center p-6">
  <div class="w-full max-w-md space-y-4">
    <div class="text-center mb-8">
      <p class="text-violet-400 text-xs uppercase tracking-widest font-semibold">AgendaGlow</p>
      <h1 class="text-3xl font-bold mt-2">Escolha seu perfil</h1>
    </div>
    <a href="/login/superadmin" class="block rounded-2xl border border-slate-700 bg-slate-900 px-5 py-4 hover:border-violet-500 transition-colors">
      <span class="font-semibold text-white">Super Admin</span>
      <span class="block text-sm text-slate-400 mt-1">Planos, salões e assinaturas SaaS</span>
    </a>
    <a href="/login/dona" class="block rounded-2xl border border-slate-700 bg-slate-900 px-5 py-4 hover:border-violet-500 transition-colors">
      <span class="font-semibold text-white">Dona do Salão</span>
      <span class="block text-sm text-slate-400 mt-1">Finanças, equipe e procedimentos</span>
    </a>
    <a href="/login/profissional" class="block rounded-2xl border border-slate-700 bg-slate-900 px-5 py-4 hover:border-violet-500 transition-colors">
      <span class="font-semibold text-white">Profissional Parceira</span>
      <span class="block text-sm text-slate-400 mt-1">Agenda individual do dia</span>
    </a>
  </div>
</body>
</html>`

const loginPageHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>AgendaGlow · {{.Title}}</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center p-6">
  <div class="w-full max-w-sm">
    <a href="/login" class="text-sm text-slate-500 hover:text-violet-400">← Voltar</a>
    <div class="mt-6 mb-8">
      <p class="text-violet-400 text-xs uppercase tracking-widest font-semibold">AgendaGlow</p>
      <h1 class="text-2xl font-bold mt-2">{{.Title}}</h1>
      <p class="text-slate-400 text-sm mt-1">{{.Subtitle}}</p>
    </div>
    {{if .ShowError}}
    <div class="mb-4 rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-3 text-sm text-rose-300">
      E-mail ou senha inválidos. Tente novamente.
    </div>
    {{end}}
    <form method="POST" action="{{.PostAction}}" class="space-y-4">
      <input type="hidden" name="role" value="{{.Role}}">
      <div>
        <label class="block text-sm text-slate-400 mb-1.5" for="email">E-mail</label>
        <input id="email" name="email" type="email" required autocomplete="username"
          class="w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-white focus:border-violet-500 focus:outline-none">
      </div>
      <div>
        <label class="block text-sm text-slate-400 mb-1.5" for="password">Senha</label>
        <input id="password" name="password" type="password" required autocomplete="current-password"
          class="w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-white focus:border-violet-500 focus:outline-none">
      </div>
      <button type="submit"
        class="w-full rounded-xl bg-violet-600 hover:bg-violet-500 py-3 font-semibold text-white transition-colors">
        Entrar
      </button>
    </form>
    {{if .DevHint}}
    <p class="text-xs text-slate-600 mt-6 text-center">{{.DevHint}} · senha: AgendaGlow@2026</p>
    {{end}}
  </div>
</body>
</html>`
