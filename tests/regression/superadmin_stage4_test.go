//go:build regression

// Suíte de regressão QA — Stage 4 (Super Admin, §9 do brief DEV-10).
// Requer a API rodando em http://localhost:8081 e Postgres em localhost:5435.
//
//	go test -tags=regression ./tests/regression/ -run Stage4 -v
package regression

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	superAdminEmail = "ferrariwill@gmail.com"
	donaSeedEmail   = "dona@glow.local"
	devPassword     = "AgendaGlow@2026"
	defaultBaseURL  = "http://localhost:8081"
	defaultDBURL    = "postgresql://postgres:glow_secure_pwd_2026@localhost:5435/agenda_glow_prod?sslmode=disable"
)

var (
	dbOnce sync.Once
	dbRef  *sqlx.DB
	dbErr  error

	httpClient = &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

func baseURL() string {
	if v := strings.TrimSpace(os.Getenv("AGENDAGLOW_API_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultBaseURL
}

func db(t *testing.T) *sqlx.DB {
	t.Helper()
	dbOnce.Do(func() {
		dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
		if dsn == "" {
			dsn = defaultDBURL
		}
		dbRef, dbErr = sqlx.Connect("postgres", dsn)
	})
	if dbErr != nil {
		t.Fatalf("conectar ao Postgres: %v", dbErr)
	}
	return dbRef
}

type loginResult struct {
	Token string `json:"token"`
	Role  string `json:"role"`
	User  struct {
		ID                string  `json:"id"`
		Email             string  `json:"email"`
		EstabelecimentoID *string `json:"estabelecimento_id"`
		ProfissionalID    *string `json:"profissional_id"`
	} `json:"user"`
}

func login(t *testing.T, email, password string) loginResult {
	t.Helper()
	status, body := requestJSON(t, http.MethodPost, "/api/v1/auth/login", "",
		map[string]string{"email": email, "password": password})
	if status != http.StatusOK {
		t.Fatalf("login %s: status %d body %s", email, status, body)
	}
	var out loginResult
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decodificar login: %v (%s)", err, body)
	}
	if strings.TrimSpace(out.Token) == "" {
		t.Fatalf("login %s sem token: %s", email, body)
	}
	return out
}

func requestJSON(t *testing.T, method, path, token string, payload any) (int, []byte) {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("serializar payload: %v", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, baseURL()+path, body)
	if err != nil {
		t.Fatalf("montar request %s %s: %v", method, path, err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return do(t, req)
}

func requestForm(t *testing.T, path, token string, form url.Values) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL()+path, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("montar form %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return do(t, req)
}

func requestHTML(t *testing.T, path, token string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL()+path, nil)
	if err != nil {
		t.Fatalf("montar GET %s: %v", path, err)
	}
	req.Header.Set("Accept", "text/html")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return do(t, req)
}

func do(t *testing.T, req *http.Request) (int, []byte) {
	t.Helper()
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ler corpo de %s: %v", req.URL.Path, err)
	}
	return resp.StatusCode, raw
}

// ---------------------------------------------------------------- fixtures

func uniqueSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
}

// criarPlano cadastra um plano SaaS ativo e agenda a limpeza.
func criarPlano(t *testing.T, token string, preco float64, limite int) (id, nome string) {
	t.Helper()
	nome = "QA Stage4 Plano " + uniqueSuffix()
	status, body := requestJSON(t, http.MethodPost, "/api/v1/admin/plans", token, map[string]any{
		"nome":                 nome,
		"preco_mensal":         preco,
		"limite_profissionais": limite,
	})
	if status != http.StatusCreated {
		t.Fatalf("criar plano: status %d body %s", status, body)
	}
	var out struct{ ID string }
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decodificar plano: %v (%s)", err, body)
	}
	t.Cleanup(func() { removerPlano(t, out.ID) })
	return out.ID, nome
}

// criarSalao cadastra um salão via API JSON e agenda a limpeza.
func criarSalao(t *testing.T, token string) (id, slug string) {
	t.Helper()
	slug = "qa-stage4-" + uniqueSuffix()
	status, body := requestJSON(t, http.MethodPost, "/api/v1/admin/establishments", token, map[string]string{
		"nome_comercial": "QA Stage4 Salão " + slug,
		"slug":           slug,
	})
	if status != http.StatusCreated {
		t.Fatalf("criar salão: status %d body %s", status, body)
	}
	var out struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decodificar salão: %v (%s)", err, body)
	}
	t.Cleanup(func() { removerSalao(t, out.ID) })
	return out.ID, out.Slug
}

func removerSalao(t *testing.T, id string) {
	t.Helper()
	conn := db(t)
	if _, err := conn.Exec(`DELETE FROM users WHERE estabelecimento_id = $1`, id); err != nil {
		t.Logf("limpeza users do salão %s: %v", id, err)
	}
	if _, err := conn.Exec(`DELETE FROM assinaturas_estabelecimentos WHERE estabelecimento_id = $1`, id); err != nil {
		t.Logf("limpeza assinatura do salão %s: %v", id, err)
	}
	if _, err := conn.Exec(`DELETE FROM estabelecimentos WHERE id = $1`, id); err != nil {
		t.Logf("limpeza do salão %s: %v", id, err)
	}
}

func removerPlano(t *testing.T, id string) {
	t.Helper()
	if _, err := db(t).Exec(`DELETE FROM planos_saas WHERE id = $1`, id); err != nil {
		t.Logf("limpeza do plano %s: %v", id, err)
	}
}

type assinaturaRow struct {
	PlanoID        string    `db:"plano_id"`
	Status         string    `db:"status"`
	DataVencimento time.Time `db:"data_vencimento"`
}

func lerAssinatura(t *testing.T, estID string) assinaturaRow {
	t.Helper()
	var row assinaturaRow
	err := db(t).Get(&row,
		`SELECT plano_id, status, data_vencimento FROM assinaturas_estabelecimentos WHERE estabelecimento_id = $1`,
		estID)
	if err != nil {
		t.Fatalf("ler assinatura de %s: %v", estID, err)
	}
	return row
}

func lerAtivo(t *testing.T, estID string) bool {
	t.Helper()
	var ativo bool
	if err := db(t).Get(&ativo, `SELECT ativo FROM estabelecimentos WHERE id = $1`, estID); err != nil {
		t.Fatalf("ler ativo de %s: %v", estID, err)
	}
	return ativo
}

func dataISO(ts time.Time) string { return ts.Format("2006-01-02") }

func hojeUTC() time.Time { return time.Now().UTC().Truncate(24 * time.Hour) }

type tenantBootstrap struct {
	ID               string  `json:"id"`
	Slug             string  `json:"slug"`
	Ativo            bool    `json:"ativo"`
	StatusAssinatura string  `json:"status_assinatura"`
	PlanoNome        *string `json:"plano_nome"`
	DataVencimento   *string `json:"data_vencimento"`
}

func lerBootstrapAdmin(t *testing.T, token string) []tenantBootstrap {
	t.Helper()
	status, body := requestJSON(t, http.MethodGet, "/api/v1/admin/bootstrap", token, nil)
	if status != http.StatusOK {
		t.Fatalf("bootstrap admin: status %d body %s", status, body)
	}
	var out struct {
		Tenants []tenantBootstrap `json:"tenants"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decodificar bootstrap admin: %v", err)
	}
	return out.Tenants
}

func acharTenant(t *testing.T, token, estID string) tenantBootstrap {
	t.Helper()
	for _, tn := range lerBootstrapAdmin(t, token) {
		if tn.ID == estID {
			return tn
		}
	}
	t.Fatalf("salão %s ausente do bootstrap do Super Admin", estID)
	return tenantBootstrap{}
}

// ---------------------------------------------------------------- casos §9

func TestStage4SuperAdminEscopoEGuarda(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	if sa.Role != "SUPER_ADMIN" {
		t.Fatalf("role esperada SUPER_ADMIN, obtida %q", sa.Role)
	}
	if sa.User.EstabelecimentoID != nil || sa.User.ProfissionalID != nil {
		t.Fatalf("Super Admin não deve ter escopo de tenant: %+v", sa.User)
	}

	t.Run("sem token retorna 401", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodGet, "/api/v1/admin/establishments", "", nil)
		if status != http.StatusUnauthorized {
			t.Fatalf("esperado 401, obtido %d (%s)", status, body)
		}
	})

	t.Run("token de dona retorna 403", func(t *testing.T) {
		dona := login(t, donaSeedEmail, devPassword)
		for _, path := range []string{
			"/api/v1/admin/establishments",
			"/api/v1/admin/plans",
			"/api/v1/admin/bootstrap",
		} {
			status, body := requestJSON(t, http.MethodGet, path, dona.Token, nil)
			if status != http.StatusForbidden {
				t.Fatalf("%s: esperado 403 para DONA, obtido %d (%s)", path, status, body)
			}
		}
	})
}

func TestStage4CrudPlanosSaaS(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)

	planoID, nome := criarPlano(t, sa.Token, 149.9, 3)

	t.Run("plano aparece na listagem", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodGet, "/api/v1/admin/plans", sa.Token, nil)
		if status != http.StatusOK {
			t.Fatalf("listar planos: status %d", status)
		}
		if !strings.Contains(string(body), nome) {
			t.Fatalf("plano %q ausente da listagem", nome)
		}
	})

	t.Run("nome duplicado case-insensitive retorna 409", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost, "/api/v1/admin/plans", sa.Token, map[string]any{
			"nome":                 strings.ToUpper(nome),
			"preco_mensal":         10,
			"limite_profissionais": 1,
		})
		if status != http.StatusConflict {
			t.Fatalf("esperado 409 duplicate_plan_name, obtido %d (%s)", status, body)
		}
	})

	t.Run("limite zero retorna 400", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost, "/api/v1/admin/plans", sa.Token, map[string]any{
			"nome":                 "QA Stage4 Limite " + uniqueSuffix(),
			"preco_mensal":         10,
			"limite_profissionais": 0,
		})
		if status != http.StatusBadRequest {
			t.Fatalf("esperado 400 invalid_professional_limit, obtido %d (%s)", status, body)
		}
	})

	t.Run("update altera nome preco limite e ativo", func(t *testing.T) {
		novoNome := nome + " editado"
		status, body := requestJSON(t, http.MethodPut, "/api/v1/admin/plans/"+planoID, sa.Token, map[string]any{
			"nome":                 novoNome,
			"preco_mensal":         199.5,
			"limite_profissionais": 5,
			"ativo":                true,
		})
		if status != http.StatusOK {
			t.Fatalf("update plano: status %d (%s)", status, body)
		}
		var row struct {
			Nome                string  `db:"nome"`
			PrecoMensal         float64 `db:"preco_mensal"`
			LimiteProfissionais int     `db:"limite_profissionais"`
			Ativo               bool    `db:"ativo"`
		}
		if err := db(t).Get(&row,
			`SELECT nome, preco_mensal, limite_profissionais, ativo FROM planos_saas WHERE id = $1`, planoID); err != nil {
			t.Fatalf("ler plano: %v", err)
		}
		if row.Nome != novoNome || row.PrecoMensal != 199.5 || row.LimiteProfissionais != 5 || !row.Ativo {
			t.Fatalf("plano não refletiu a edição: %+v", row)
		}
	})

	t.Run("update com limite invalido retorna 400", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPut, "/api/v1/admin/plans/"+planoID, sa.Token, map[string]any{
			"nome":                 nome,
			"preco_mensal":         50,
			"limite_profissionais": 0,
			"ativo":                true,
		})
		if status != http.StatusBadRequest {
			t.Fatalf("esperado 400, obtido %d (%s)", status, body)
		}
	})

	t.Run("update de plano inexistente retorna 404", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPut,
			"/api/v1/admin/plans/11111111-2222-4333-8444-555555555555", sa.Token, map[string]any{
				"nome":                 "QA Stage4 Fantasma " + uniqueSuffix(),
				"preco_mensal":         10,
				"limite_profissionais": 1,
				"ativo":                true,
			})
		if status != http.StatusNotFound {
			t.Fatalf("esperado 404, obtido %d (%s)", status, body)
		}
	})

	t.Run("plano inativo nao pode ser atribuido", func(t *testing.T) {
		inativoID, inativoNome := criarPlano(t, sa.Token, 10, 1)
		status, body := requestJSON(t, http.MethodPut, "/api/v1/admin/plans/"+inativoID, sa.Token, map[string]any{
			"nome":                 inativoNome,
			"preco_mensal":         10,
			"limite_profissionais": 1,
			"ativo":                false,
		})
		if status != http.StatusOK {
			t.Fatalf("desativar plano: status %d (%s)", status, body)
		}

		estID, _ := criarSalao(t, sa.Token)
		status, body = requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token, map[string]any{
				"plano_id": inativoID,
				"meses":    12,
			})
		if status != http.StatusNotFound {
			t.Fatalf("esperado 404 plan_not_found para plano inativo, obtido %d (%s)", status, body)
		}
	})
}

func TestStage4CriarSalaoESlug(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	estID, slug := criarSalao(t, sa.Token)

	if !lerAtivo(t, estID) {
		t.Fatalf("salão recém-criado deve nascer ativo")
	}

	t.Run("slug duplicado retorna 409", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost, "/api/v1/admin/establishments", sa.Token,
			map[string]string{"nome_comercial": "Outro salão", "slug": slug})
		if status != http.StatusConflict {
			t.Fatalf("esperado 409 slug_already_exists, obtido %d (%s)", status, body)
		}
		if !strings.Contains(string(body), "slug_already_exists") {
			t.Fatalf("corpo sem código slug_already_exists: %s", body)
		}
	})

	t.Run("slug invalido retorna 400", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost, "/api/v1/admin/establishments", sa.Token,
			map[string]string{"nome_comercial": "!!!", "slug": "!!!"})
		if status != http.StatusBadRequest {
			t.Fatalf("esperado 400 invalid_slug, obtido %d (%s)", status, body)
		}
	})

	t.Run("pagina publica do slug responde 200", func(t *testing.T) {
		status, _ := requestHTML(t, "/"+slug, "")
		if status != http.StatusOK {
			t.Fatalf("GET /%s: esperado 200, obtido %d", slug, status)
		}
	})

	t.Run("salao aparece na listagem do super admin", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodGet, "/api/v1/admin/establishments", sa.Token, nil)
		if status != http.StatusOK {
			t.Fatalf("listar salões: status %d", status)
		}
		if !strings.Contains(string(body), slug) {
			t.Fatalf("slug %s ausente da listagem", slug)
		}
	})
}

func TestStage4AtribuirPlanoEVencimento(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	planoID, planoNome := criarPlano(t, sa.Token, 97, 2)
	estID, _ := criarSalao(t, sa.Token)

	status, body := requestJSON(t, http.MethodPost,
		"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
		map[string]any{"plano_id": planoID, "meses": 12})
	if status != http.StatusOK {
		t.Fatalf("atribuir plano: status %d (%s)", status, body)
	}

	assinatura := lerAssinatura(t, estID)
	if assinatura.PlanoID != planoID {
		t.Fatalf("plano_id esperado %s, obtido %s", planoID, assinatura.PlanoID)
	}
	if assinatura.Status != "ATIVO" {
		t.Fatalf("status da assinatura esperado ATIVO, obtido %s", assinatura.Status)
	}
	esperado := hojeUTC().AddDate(0, 12, 0)
	if got := dataISO(assinatura.DataVencimento); got != dataISO(esperado) {
		t.Fatalf("vencimento esperado %s, obtido %s", dataISO(esperado), got)
	}

	t.Run("nova atribuicao soma a partir do vencimento vigente", func(t *testing.T) {
		anterior := lerAssinatura(t, estID).DataVencimento
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
			map[string]any{"plano_id": planoID, "meses": 6})
		if status != http.StatusOK {
			t.Fatalf("reatribuir plano: status %d (%s)", status, body)
		}
		esperado := anterior.AddDate(0, 6, 0)
		if got := dataISO(lerAssinatura(t, estID).DataVencimento); got != dataISO(esperado) {
			t.Fatalf("vencimento esperado %s (base %s + 6m), obtido %s",
				dataISO(esperado), dataISO(anterior), got)
		}
	})

	t.Run("vencimento expirado usa hoje como base", func(t *testing.T) {
		if _, err := db(t).Exec(
			`UPDATE assinaturas_estabelecimentos SET data_vencimento = CURRENT_DATE - INTERVAL '40 days'
			 WHERE estabelecimento_id = $1`, estID); err != nil {
			t.Fatalf("forçar vencimento passado: %v", err)
		}
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
			map[string]any{"plano_id": planoID, "meses": 1})
		if status != http.StatusOK {
			t.Fatalf("atribuir plano após vencimento: status %d (%s)", status, body)
		}
		esperado := hojeUTC().AddDate(0, 1, 0)
		if got := dataISO(lerAssinatura(t, estID).DataVencimento); got != dataISO(esperado) {
			t.Fatalf("vencimento esperado %s, obtido %s", dataISO(esperado), got)
		}
	})

	t.Run("meses invalidos retorna 400", func(t *testing.T) {
		for _, meses := range []int{0, -3} {
			status, body := requestJSON(t, http.MethodPost,
				"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
				map[string]any{"plano_id": planoID, "meses": meses})
			if status != http.StatusBadRequest {
				t.Fatalf("meses=%d: esperado 400 invalid_months, obtido %d (%s)", meses, status, body)
			}
		}
	})

	t.Run("plano inexistente retorna 404", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
			map[string]any{"plano_id": "99999999-8888-4777-8666-555555555555", "meses": 12})
		if status != http.StatusNotFound {
			t.Fatalf("esperado 404 plan_not_found, obtido %d (%s)", status, body)
		}
	})

	t.Run("salao inexistente retorna 404", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/12121212-3434-4565-8787-909090909090/assign-plan", sa.Token,
			map[string]any{"plano_id": planoID, "meses": 12})
		if status != http.StatusNotFound {
			t.Fatalf("esperado 404 establishment_not_found, obtido %d (%s)", status, body)
		}
	})

	t.Run("bootstrap reflete plano e status ATIVO", func(t *testing.T) {
		tn := acharTenant(t, sa.Token, estID)
		if tn.StatusAssinatura != "ATIVO" {
			t.Fatalf("status_assinatura esperado ATIVO, obtido %s", tn.StatusAssinatura)
		}
		if tn.PlanoNome == nil || *tn.PlanoNome != planoNome {
			t.Fatalf("plano_nome esperado %q, obtido %v", planoNome, tn.PlanoNome)
		}
	})
}

func TestStage4RenovarDozeMeses(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	planoID, _ := criarPlano(t, sa.Token, 97, 1)
	estID, _ := criarSalao(t, sa.Token)

	t.Run("renovar sem plano atribuido retorna 400", func(t *testing.T) {
		status, body := requestForm(t, "/superadmin/establishments/"+estID+"/renew", sa.Token, url.Values{})
		if status != http.StatusBadRequest {
			t.Fatalf("esperado 400 (salão sem plano), obtido %d (%s)", status, body)
		}
	})

	if status, body := requestJSON(t, http.MethodPost,
		"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
		map[string]any{"plano_id": planoID, "meses": 12}); status != http.StatusOK {
		t.Fatalf("atribuir plano: status %d (%s)", status, body)
	}

	anterior := lerAssinatura(t, estID).DataVencimento
	status, body := requestForm(t, "/superadmin/establishments/"+estID+"/renew", sa.Token, url.Values{})
	if status != http.StatusOK {
		t.Fatalf("renovar: status %d (%s)", status, body)
	}
	if !strings.Contains(string(body), "+12 meses") {
		t.Fatalf("flash de renovação ausente no fragmento HTMX: %s", body)
	}

	depois := lerAssinatura(t, estID)
	esperado := anterior.AddDate(0, 12, 0)
	if got := dataISO(depois.DataVencimento); got != dataISO(esperado) {
		t.Fatalf("vencimento esperado %s (base %s +12m), obtido %s",
			dataISO(esperado), dataISO(anterior), got)
	}
	if depois.Status != "ATIVO" {
		t.Fatalf("status após renovação esperado ATIVO, obtido %s", depois.Status)
	}
}

func TestStage4SuspenderEAtivar(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	planoID, _ := criarPlano(t, sa.Token, 97, 1)
	estID, slug := criarSalao(t, sa.Token)

	if status, body := requestJSON(t, http.MethodPost,
		"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
		map[string]any{"plano_id": planoID, "meses": 12}); status != http.StatusOK {
		t.Fatalf("atribuir plano: status %d (%s)", status, body)
	}

	status, body := requestForm(t, "/superadmin/establishments/"+estID+"/suspend", sa.Token, url.Values{})
	if status != http.StatusOK {
		t.Fatalf("suspender: status %d (%s)", status, body)
	}
	if !strings.Contains(string(body), "SUSPENSO") {
		t.Fatalf("fragmento da linha deveria exibir SUSPENSO: %s", body)
	}
	if lerAtivo(t, estID) {
		t.Fatalf("suspender deve zerar estabelecimentos.ativo")
	}
	if got := lerAssinatura(t, estID).Status; got != "SUSPENSO" {
		t.Fatalf("assinatura esperada SUSPENSO, obtida %s", got)
	}
	if tn := acharTenant(t, sa.Token, estID); tn.StatusAssinatura != "SUSPENSO" {
		t.Fatalf("status_assinatura exibido esperado SUSPENSO, obtido %s", tn.StatusAssinatura)
	}
	if st, _ := requestHTML(t, "/"+slug, ""); st != http.StatusNotFound {
		t.Fatalf("slug de salão suspenso deveria retornar 404, obtido %d", st)
	}

	status, body = requestForm(t, "/superadmin/establishments/"+estID+"/activate", sa.Token, url.Values{})
	if status != http.StatusOK {
		t.Fatalf("ativar: status %d (%s)", status, body)
	}
	if !lerAtivo(t, estID) {
		t.Fatalf("ativar deve marcar estabelecimentos.ativo = true")
	}
	if got := lerAssinatura(t, estID).Status; got != "ATIVO" {
		t.Fatalf("assinatura esperada ATIVO, obtida %s", got)
	}
	if tn := acharTenant(t, sa.Token, estID); tn.StatusAssinatura != "ATIVO" {
		t.Fatalf("status_assinatura exibido esperado ATIVO, obtido %s", tn.StatusAssinatura)
	}
	if st, _ := requestHTML(t, "/"+slug, ""); st != http.StatusOK {
		t.Fatalf("slug de salão reativado deveria retornar 200, obtido %d", st)
	}

	t.Run("toggle JSON altera apenas ativo", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPut,
			"/api/v1/admin/establishments/"+estID+"/status", sa.Token, map[string]bool{"ativo": false})
		if status != http.StatusOK {
			t.Fatalf("toggle status: %d (%s)", status, body)
		}
		if lerAtivo(t, estID) {
			t.Fatalf("toggle deveria desativar o salão")
		}
		// Comportamento documentado (§13 REGRAS_NEGOCIO): o toggle JSON não mexe na assinatura.
		if got := lerAssinatura(t, estID).Status; got != "ATIVO" {
			t.Fatalf("assinatura deveria permanecer ATIVO no toggle JSON, obtida %s", got)
		}
		if status, _ := requestJSON(t, http.MethodPut,
			"/api/v1/admin/establishments/"+estID+"/status", sa.Token, map[string]bool{"ativo": true}); status != http.StatusOK {
			t.Fatalf("reativar via toggle: %d", status)
		}
	})

	t.Run("toggle de salao inexistente retorna 404", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPut,
			"/api/v1/admin/establishments/33333333-4444-4555-8666-777777777777/status", sa.Token,
			map[string]bool{"ativo": false})
		if status != http.StatusNotFound {
			t.Fatalf("esperado 404, obtido %d (%s)", status, body)
		}
	})
}

func TestStage4StatusVencidoExibido(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	planoID, _ := criarPlano(t, sa.Token, 97, 1)
	estID, slug := criarSalao(t, sa.Token)

	if status, body := requestJSON(t, http.MethodPost,
		"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
		map[string]any{"plano_id": planoID, "meses": 12}); status != http.StatusOK {
		t.Fatalf("atribuir plano: status %d (%s)", status, body)
	}

	t.Run("sem assinatura o status e VENCIDO", func(t *testing.T) {
		semPlanoID, _ := criarSalao(t, sa.Token)
		if tn := acharTenant(t, sa.Token, semPlanoID); tn.StatusAssinatura != "VENCIDO" {
			t.Fatalf("salão sem assinatura deveria exibir VENCIDO, obtido %s", tn.StatusAssinatura)
		}
	})

	if _, err := db(t).Exec(
		`UPDATE assinaturas_estabelecimentos SET data_vencimento = CURRENT_DATE - INTERVAL '5 days'
		 WHERE estabelecimento_id = $1`, estID); err != nil {
		t.Fatalf("forçar vencimento passado: %v", err)
	}

	if tn := acharTenant(t, sa.Token, estID); tn.StatusAssinatura != "VENCIDO" {
		t.Fatalf("status_assinatura esperado VENCIDO, obtido %s", tn.StatusAssinatura)
	}

	statusHTML, html := requestHTML(t, "/superadmin/dashboard", sa.Token)
	if statusHTML != http.StatusOK {
		t.Fatalf("dashboard super admin: status %d", statusHTML)
	}
	linha := extrairLinha(string(html), estID)
	if linha == "" {
		t.Fatalf("linha do salão %s (%s) ausente do dashboard", slug, estID)
	}
	if !strings.Contains(linha, "VENCIDO") {
		t.Fatalf("linha do salão deveria exibir badge VENCIDO: %s", linha)
	}
}

func TestStage4CriarDonaDoSalao(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	planoID, _ := criarPlano(t, sa.Token, 97, 1)
	estID, slug := criarSalao(t, sa.Token)
	if status, body := requestJSON(t, http.MethodPost,
		"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
		map[string]any{"plano_id": planoID, "meses": 12}); status != http.StatusOK {
		t.Fatalf("atribuir plano: status %d (%s)", status, body)
	}

	// Sem e-mail no formulário → padrão {slug}-dona@glow.local.
	status, body := requestForm(t, "/superadmin/establishments/"+estID+"/create-dona", sa.Token, url.Values{})
	if status != http.StatusOK {
		t.Fatalf("criar dona: status %d (%s)", status, body)
	}
	emailPadrao := slug + "-dona@glow.local"
	if !strings.Contains(string(body), emailPadrao) {
		t.Fatalf("flash deveria informar o e-mail %s: %s", emailPadrao, body)
	}

	dona := login(t, emailPadrao, devPassword)
	if dona.Role != "DONA" {
		t.Fatalf("role esperada DONA, obtida %s", dona.Role)
	}
	if dona.User.EstabelecimentoID == nil || *dona.User.EstabelecimentoID != estID {
		t.Fatalf("estabelecimento_id do token esperado %s, obtido %v", estID, dona.User.EstabelecimentoID)
	}

	t.Run("e-mail duplicado retorna 409", func(t *testing.T) {
		form := url.Values{}
		form.Set("email", emailPadrao)
		status, body := requestForm(t, "/superadmin/establishments/"+estID+"/create-dona", sa.Token, form)
		if status != http.StatusConflict {
			t.Fatalf("esperado 409, obtido %d (%s)", status, body)
		}
	})

	t.Run("e-mail informado e normalizado", func(t *testing.T) {
		email := "QA.Stage4." + uniqueSuffix() + "@Glow.Local"
		form := url.Values{}
		form.Set("email", email)
		status, body := requestForm(t, "/superadmin/establishments/"+estID+"/create-dona", sa.Token, form)
		if status != http.StatusOK {
			t.Fatalf("criar dona com e-mail informado: %d (%s)", status, body)
		}
		nova := login(t, strings.ToLower(email), devPassword)
		if nova.Role != "DONA" || nova.User.Email != strings.ToLower(email) {
			t.Fatalf("dona criada com escopo inesperado: %+v", nova.User)
		}
	})

	t.Run("salao inexistente retorna 404", func(t *testing.T) {
		status, body := requestForm(t,
			"/superadmin/establishments/44444444-5555-4666-8777-888888888888/create-dona", sa.Token, url.Values{})
		if status != http.StatusNotFound {
			t.Fatalf("esperado 404, obtido %d (%s)", status, body)
		}
	})
}

func TestStage4UISuperAdminAcoesDoTenant(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	planoID, planoNome := criarPlano(t, sa.Token, 97, 1)
	estID, slug := criarSalao(t, sa.Token)
	if status, body := requestJSON(t, http.MethodPost,
		"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
		map[string]any{"plano_id": planoID, "meses": 12}); status != http.StatusOK {
		t.Fatalf("atribuir plano: status %d (%s)", status, body)
	}

	status, html := requestHTML(t, "/superadmin/dashboard", sa.Token)
	if status != http.StatusOK {
		t.Fatalf("dashboard: status %d", status)
	}
	page := string(html)
	linha := extrairLinha(page, estID)
	if linha == "" {
		t.Fatalf("linha do salão %s ausente do dashboard", slug)
	}
	acoes := map[string]string{
		"atribuir/trocar plano": "/superadmin/establishments/" + estID + "/assign-plan",
		"criar dona":            "/superadmin/establishments/" + estID + "/create-dona",
		"suspender":             "/superadmin/establishments/" + estID + "/suspend",
		"renovar +12m":          "/superadmin/establishments/" + estID + "/renew",
	}
	for nome, alvo := range acoes {
		if !strings.Contains(linha, alvo) {
			t.Fatalf("ação %q (%s) ausente da linha do tenant", nome, alvo)
		}
	}
	if !strings.Contains(linha, planoNome) && !strings.Contains(linha, "Trocar") {
		t.Fatalf("linha do tenant sem plano/seletor de troca: %s", linha)
	}

	t.Run("pagina de planos lista o plano criado", func(t *testing.T) {
		status, html := requestHTML(t, "/superadmin/planos", sa.Token)
		if status != http.StatusOK {
			t.Fatalf("planos: status %d", status)
		}
		if !strings.Contains(string(html), planoNome) {
			t.Fatalf("plano %q ausente de /superadmin/planos", planoNome)
		}
	})

	t.Run("rota HTML sem sessao redireciona para login", func(t *testing.T) {
		status, _ := requestHTML(t, "/superadmin/dashboard", "")
		if status != http.StatusSeeOther {
			t.Fatalf("esperado 303 para /login, obtido %d", status)
		}
	})
}

// TestStage4MenuAcoesTenantReact protege o menu “…” do Super Admin contra
// regressão de overflow: ele precisa continuar em portal + position fixed.
func TestStage4MenuAcoesTenantReact(t *testing.T) {
	dropdown := lerArquivo(t, "../../frontend-react/src/components/ui/ActionsDropdown.tsx")
	for _, marca := range []string{"createPortal", "document.body", "fixed z-[200]", "overflow-y-auto", "maxHeight"} {
		if !strings.Contains(dropdown, marca) {
			t.Fatalf("ActionsDropdown perdeu %q — menu volta a ser cortado pelo container", marca)
		}
	}

	menu := lerArquivo(t, "../../frontend-react/src/components/superadmin/TenantActionsMenu.tsx")
	if !strings.Contains(menu, "ActionsDropdown") {
		t.Fatalf("TenantActionsMenu deveria usar ActionsDropdown (portal)")
	}
	for _, item := range []string{"Atribuir plano", "Renovar +12 meses", "Suspender", "Ativar", "Criar dona", "Ver catálogo"} {
		if !strings.Contains(menu, item) {
			t.Fatalf("ação %q ausente do menu do tenant", item)
		}
	}
}

func lerArquivo(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler %s: %v", path, err)
	}
	return string(raw)
}

// extrairLinha isola o <tr> do salão informado no HTML do dashboard.
func extrairLinha(page, estID string) string {
	marker := `id="est-row-` + estID + `"`
	start := strings.Index(page, marker)
	if start < 0 {
		return ""
	}
	rest := page[start:]
	if end := strings.Index(rest, "</tr>"); end > 0 {
		return rest[:end]
	}
	return rest
}
