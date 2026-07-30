//go:build regression

package regression_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	c := newClient(t)
	status, body := c.do(http.MethodGet, "/health", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("health: %d %s", status, body)
	}
	if !strings.Contains(body, `"ok"`) && !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("health body inesperado: %s", body)
	}
}

func TestAuth_LoginPorPerfil(t *testing.T) {
	cases := []struct {
		email string
		role  string
	}{
		{"ferrariwill@gmail.com", "SUPER_ADMIN"},
		{"dona@glow.local", "DONA"},
		{"claudia@glow.local", "PROFISSIONAL"},
	}
	for _, tc := range cases {
		t.Run(tc.role, func(t *testing.T) {
			c := newClient(t)
			out := c.login(tc.email)
			if out["role"] != tc.role {
				t.Fatalf("role: got %v want %s", out["role"], tc.role)
			}
			user, _ := out["user"].(map[string]any)
			switch tc.role {
			case "DONA":
				if user["estabelecimento_id"] == nil || user["estabelecimento_id"] == "" {
					t.Fatal("DONA sem estabelecimento_id")
				}
			case "PROFISSIONAL":
				if user["estabelecimento_id"] == nil || user["profissional_id"] == nil {
					t.Fatal("PROFISSIONAL sem escopo completo")
				}
			case "SUPER_ADMIN":
				if user["estabelecimento_id"] != nil {
					t.Fatalf("SUPER_ADMIN não deve ter estabelecimento_id: %v", user["estabelecimento_id"])
				}
			}
		})
	}
}

func TestAuth_CredenciaisInvalidas(t *testing.T) {
	c := newClient(t)
	status, body := c.do(http.MethodPost, "/api/v1/auth/login", nil, map[string]string{
		"email": "dona@glow.local", "password": "senha-errada",
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d body=%s", status, body)
	}
	if strings.Contains(strings.ToLower(body), "não existe") || strings.Contains(strings.ToLower(body), "not found") {
		t.Fatalf("erro revelou existência do e-mail: %s", body)
	}
}

func TestAuth_EmailCaseInsensitive(t *testing.T) {
	c := newClient(t)
	out := c.login("DONA@GLOW.LOCAL")
	if out["role"] != "DONA" {
		t.Fatalf("case-insensitive login falhou: %v", out)
	}
}

func TestAuth_ProfissionalIgnoraGuardaSaaS_KNOWN(t *testing.T) {
	c := newClient(t)
	c.login("claudia@glow.local")
	status, body := c.do(http.MethodGet, "/api/v1/professional/dashboard", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("dashboard profissional: %d %s", status, body)
	}
}

func TestMultiTenant_PublicSlugAtivo(t *testing.T) {
	c := newClient(t)
	status, body := c.do(http.MethodGet, "/api/v1/public/"+seedSlug+"/catalog", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("catalog público: %d %s", status, body)
	}
	if !strings.Contains(body, seedServicoID) && !strings.Contains(body, "Fazer Unhas") {
		t.Fatalf("catálogo sem serviço seed: %s", body)
	}
}

func TestMultiTenant_SlugInexistente404(t *testing.T) {
	c := newClient(t)
	status, _ := c.do(http.MethodGet, "/api/v1/public/slug-que-nao-existe-xyz/catalog", nil, nil)
	if status != http.StatusNotFound {
		t.Fatalf("want 404 got %d", status)
	}
}

func TestSaaS_LimiteProfissionais(t *testing.T) {
	c := newClient(t)
	c.login("dona@glow.local")
	boot := c.mustJSON(http.MethodGet, "/api/v1/bootstrap", http.StatusOK, nil)
	espList, _ := boot["especialidades"].([]any)
	if len(espList) == 0 {
		t.Fatal("sem especialidades no seed")
	}
	espID, _ := espList[0].(map[string]any)["id"].(string)

	plano, _ := boot["plano"].(map[string]any)
	limite := int(plano["limite_profissionais"].(float64))
	profs, _ := boot["profissionais"].([]any)
	ativas := 0
	for _, p := range profs {
		if p.(map[string]any)["ativo"] == true {
			ativas++
		}
	}
	if ativas < limite {
		t.Skipf("seed não está no limite (%d/%d)", ativas, limite)
	}

	status, body := c.do(http.MethodPost, "/api/v1/professionals", nil, map[string]any{
		"nome": "Extra QA Limit", "especialidade_id": espID, "comissao_porcentagem": 30,
	})
	if status != http.StatusForbidden {
		t.Fatalf("limite plano: want 403 got %d body=%s", status, body)
	}
	var errBody map[string]any
	if err := json.Unmarshal([]byte(body), &errBody); err != nil || errBody["error"] != "limit_reached" {
		t.Fatalf("limite plano: want error=limit_reached body=%s", body)
	}
}

func TestSuperAdmin_ListEstablishments(t *testing.T) {
	c := newClient(t)
	c.login("ferrariwill@gmail.com")
	status, body := c.do(http.MethodGet, "/api/v1/admin/establishments", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("list establishments: %d %s", status, body)
	}
	if !strings.Contains(body, seedSlug) && !strings.Contains(body, seedTenantID) {
		t.Fatalf("lista sem tenant seed: %s", body)
	}
}

func TestDona_AcessoTenantRotas(t *testing.T) {
	c := newClient(t)
	c.login("dona@glow.local")
	for _, path := range []string{
		"/api/v1/bootstrap",
		"/api/v1/services",
		"/api/v1/professionals",
		"/api/v1/specialties",
		"/api/v1/dashboard/gerencial",
		"/api/v1/whatsapp/integration",
	} {
		status, body := c.do(http.MethodGet, path, nil, nil)
		if status != http.StatusOK {
			t.Fatalf("%s: %d %s", path, status, body)
		}
	}
}
