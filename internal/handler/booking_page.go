package handler

import (
	"errors"
	"html/template"
	"net/http"

	"github.com/agendaglow/agendaglow/internal/service"
)

type BookingPageHandler struct {
	estabelecimentos *service.EstabelecimentoService
	tmpl             *template.Template
}

func NewBookingPageHandler(estabelecimentos *service.EstabelecimentoService) *BookingPageHandler {
	return &BookingPageHandler{
		estabelecimentos: estabelecimentos,
		tmpl:             template.Must(template.New("booking").Parse(bookingPageHTML)),
	}
}

type bookingPageData struct {
	Catalogo *service.CatalogoAutoatendimento
}

// ServeHTTP atende GET /{slug} — página mobile de autoatendimento da cliente final.
func (h *BookingPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	est, err := h.estabelecimentos.BuscarPorSlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) || errors.Is(err, service.ErrSlugInvalido) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

	catalogo, err := h.estabelecimentos.BuscarCatalogoAutoatendimento(r.Context(), est.ID)
	if err != nil {
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.Execute(w, bookingPageData{Catalogo: catalogo}); err != nil {
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
	}
}

const bookingPageHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Catalogo.Estabelecimento.NomeComercial}} · AgendaGlow</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background: #faf7fb;
      color: #2d2a32;
      line-height: 1.5;
    }
    .header {
      background: #fff;
      padding: 1.5rem 1rem 1rem;
      text-align: center;
      border-bottom: 1px solid #ece6f0;
      box-shadow: 0 2px 12px rgba(80, 40, 120, 0.06);
    }
    .logo {
      max-width: 120px;
      max-height: 120px;
      object-fit: contain;
      margin-bottom: 0.75rem;
      border-radius: 16px;
    }
    h1 {
      font-size: 1.35rem;
      font-weight: 700;
      color: #5b2d82;
    }
    .subtitle {
      font-size: 0.9rem;
      color: #7a7085;
      margin-top: 0.25rem;
    }
    main { padding: 1rem; max-width: 480px; margin: 0 auto; }
    section { margin-bottom: 1.5rem; }
    h2 {
      font-size: 0.85rem;
      text-transform: uppercase;
      letter-spacing: 0.06em;
      color: #9a8fad;
      margin-bottom: 0.75rem;
    }
    .card {
      background: #fff;
      border-radius: 14px;
      padding: 1rem;
      margin-bottom: 0.75rem;
      border: 1px solid #ece6f0;
    }
    .card strong { display: block; font-size: 1rem; }
    .card span { font-size: 0.875rem; color: #6b6375; }
    .price { color: #5b2d82; font-weight: 600; margin-top: 0.35rem; display: block; }
    .tag {
      display: inline-block;
      background: #f3ebfa;
      color: #5b2d82;
      font-size: 0.75rem;
      padding: 0.15rem 0.5rem;
      border-radius: 999px;
      margin-top: 0.35rem;
      margin-right: 0.25rem;
    }
    footer {
      text-align: center;
      font-size: 0.75rem;
      color: #b0a8bc;
      padding: 2rem 1rem;
    }
  </style>
</head>
<body>
  <header class="header">
    {{if .Catalogo.Estabelecimento.LogoURL}}
      <img class="logo" src="{{.Catalogo.Estabelecimento.LogoURL}}" alt="Logo {{.Catalogo.Estabelecimento.NomeComercial}}">
    {{end}}
    <h1>{{.Catalogo.Estabelecimento.NomeComercial}}</h1>
    <p class="subtitle">Agende seu horário online</p>
  </header>

  <main>
    <section>
      <h2>Profissionais</h2>
      {{range .Catalogo.Profissionais}}
      <div class="card">
        <strong>{{.Nome}}</strong>
        <span>{{.Especialidade}}</span>
      </div>
      {{else}}
      <div class="card"><span>Nenhuma profissional disponível no momento.</span></div>
      {{end}}
    </section>

    <section>
      <h2>Serviços</h2>
      {{range .Catalogo.Servicos}}
      <div class="card">
        <strong>{{.Nome}}</strong>
        <span>{{.DuracaoBaseMinutos}} min</span>
        <span class="price">R$ {{printf "%.2f" .PrecoBase}}</span>
        {{range .Adicionais}}
          <span class="tag">+ {{.Nome}}</span>
        {{end}}
      </div>
      {{else}}
      <div class="card"><span>Nenhum serviço disponível no momento.</span></div>
      {{end}}
    </section>
  </main>

  <footer>AgendaGlow · Agendamento online</footer>
</body>
</html>`
