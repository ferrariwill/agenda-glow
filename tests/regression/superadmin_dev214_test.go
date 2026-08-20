//go:build regression

// Suíte de regressão QA — DEV-214 (create-dona JSON + upload logo SuperAdmin).
// Requer a API rodando em http://localhost:8081 e Postgres em localhost:5435.
//
//	go test -tags=regression ./tests/regression/ -run DEV214 -v
package regression

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

func TestDEV214CreateDonaJSON(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	estID, _ := criarSalao(t, sa.Token)

	email := fmt.Sprintf("qa.dev214.%s@glow.local", uniqueSuffix())
	nome := "Maria QA DEV214"

	status, body := requestJSON(t, http.MethodPost,
		"/api/v1/admin/establishments/"+estID+"/create-dona", sa.Token,
		map[string]string{"nome": nome, "email": email})
	if status != http.StatusCreated {
		t.Fatalf("create-dona: status %d body %s", status, body)
	}

	var out struct {
		UserID       string `json:"user_id"`
		Email        string `json:"email"`
		Nome         string `json:"nome"`
		SenhaInicial string `json:"senha_inicial"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decodificar create-dona: %v (%s)", err, body)
	}
	if out.UserID == "" || out.Email != email || out.Nome != nome || out.SenhaInicial != devPassword {
		t.Fatalf("resposta inesperada: %+v", out)
	}

	dona := login(t, email, out.SenhaInicial)
	if dona.Role != "DONA" {
		t.Fatalf("role esperada DONA, obtida %s", dona.Role)
	}
	if dona.User.EstabelecimentoID == nil || *dona.User.EstabelecimentoID != estID {
		t.Fatalf("estabelecimento_id esperado %s, obtido %v", estID, dona.User.EstabelecimentoID)
	}

	var dbNome, dbEmail string
	if err := db(t).QueryRow(
		`SELECT COALESCE(dona_nome,''), COALESCE(dona_email,'') FROM estabelecimentos WHERE id = $1`,
		estID,
	).Scan(&dbNome, &dbEmail); err != nil {
		t.Fatalf("ler dona_* do salão: %v", err)
	}
	if dbNome != nome || dbEmail != email {
		t.Fatalf("denormalizado no salão: nome=%q email=%q", dbNome, dbEmail)
	}

	var userNome string
	if err := db(t).QueryRow(`SELECT COALESCE(nome,'') FROM users WHERE id = $1`, out.UserID).Scan(&userNome); err != nil {
		t.Fatalf("ler users.nome: %v", err)
	}
	if userNome != nome {
		t.Fatalf("users.nome = %q, esperado %q", userNome, nome)
	}

	t.Run("email duplicado retorna 409", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/create-dona", sa.Token,
			map[string]string{"nome": "Outra", "email": email})
		if status != http.StatusConflict {
			t.Fatalf("esperado 409, obtido %d (%s)", status, body)
		}
		assertErrorCode(t, body, "email_already_exists")
	})

	t.Run("salao inexistente retorna 404", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/44444444-5555-4666-8777-888888888888/create-dona", sa.Token,
			map[string]string{"nome": "X", "email": "ghost-" + uniqueSuffix() + "@glow.local"})
		if status != http.StatusNotFound {
			t.Fatalf("esperado 404, obtido %d (%s)", status, body)
		}
		assertErrorCode(t, body, "not_found")
	})

	t.Run("sem nome retorna 400", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/create-dona", sa.Token,
			map[string]string{"nome": "  ", "email": "x-" + uniqueSuffix() + "@glow.local"})
		if status != http.StatusBadRequest {
			t.Fatalf("esperado 400, obtido %d (%s)", status, body)
		}
		assertErrorCode(t, body, "missing_nome")
	})

	t.Run("sem email retorna 400", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/create-dona", sa.Token,
			map[string]string{"nome": "Maria", "email": ""})
		if status != http.StatusBadRequest {
			t.Fatalf("esperado 400, obtido %d (%s)", status, body)
		}
		assertErrorCode(t, body, "missing_email")
	})

	t.Run("sem superadmin retorna 401", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/create-dona", "",
			map[string]string{"nome": "Maria", "email": "auth-" + uniqueSuffix() + "@glow.local"})
		if status != http.StatusUnauthorized && status != http.StatusForbidden {
			t.Fatalf("esperado 401/403, obtido %d (%s)", status, body)
		}
	})
}

func TestDEV214UploadLogo(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	estID, _ := criarSalao(t, sa.Token)

	t.Run("tipo invalido retorna 400", func(t *testing.T) {
		status, body := requestMultipartLogo(t, sa.Token, estID, "logo.gif", []byte("GIF89a"))
		if status != http.StatusBadRequest {
			t.Fatalf("esperado 400, obtido %d (%s)", status, body)
		}
		assertErrorCode(t, body, "invalid_image_type")
	})

	t.Run("sem arquivo retorna 400", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		_ = writer.WriteField("other", "x")
		_ = writer.Close()
		req, err := http.NewRequest(http.MethodPost, baseURL()+"/api/v1/admin/establishments/"+estID+"/logo", &buf)
		if err != nil {
			t.Fatalf("montar request: %v", err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+sa.Token)
		status, body := do(t, req)
		if status != http.StatusBadRequest {
			t.Fatalf("esperado 400, obtido %d (%s)", status, body)
		}
		assertErrorCode(t, body, "missing_logo")
	})

	t.Run("salao inexistente retorna 404", func(t *testing.T) {
		status, body := requestMultipartLogo(t, sa.Token,
			"44444444-5555-4666-8777-888888888888", "logo.png", minimalPNG())
		if status != http.StatusNotFound {
			t.Fatalf("esperado 404, obtido %d (%s)", status, body)
		}
		assertErrorCode(t, body, "not_found")
	})

	t.Run("sem superadmin retorna 401", func(t *testing.T) {
		status, body := requestMultipartLogo(t, "", estID, "logo.png", minimalPNG())
		if status != http.StatusUnauthorized && status != http.StatusForbidden {
			t.Fatalf("esperado 401/403, obtido %d (%s)", status, body)
		}
	})

	t.Run("upload valido quando storage configurado", func(t *testing.T) {
		status, body := requestMultipartLogo(t, sa.Token, estID, "logo.png", minimalPNG())
		switch status {
		case http.StatusOK:
			var out struct {
				LogoURL string `json:"logo_url"`
			}
			if err := json.Unmarshal(body, &out); err != nil {
				t.Fatalf("decodificar logo: %v (%s)", err, body)
			}
			if strings.TrimSpace(out.LogoURL) == "" {
				t.Fatalf("logo_url vazio: %s", body)
			}
			var dbURL sql.NullString
			if err := db(t).QueryRow(`SELECT logo_url FROM estabelecimentos WHERE id = $1`, estID).Scan(&dbURL); err != nil {
				t.Fatalf("ler logo_url: %v", err)
			}
			if !dbURL.Valid || dbURL.String != out.LogoURL {
				t.Fatalf("logo_url no banco = %v, esperado %q", dbURL, out.LogoURL)
			}
		case http.StatusBadGateway:
			var errBody struct {
				Error string `json:"error"`
			}
			_ = json.Unmarshal(body, &errBody)
			if errBody.Error != "storage_upload_failed" {
				t.Fatalf("esperado storage_upload_failed, obtido %s", body)
			}
			t.Skip("Supabase não configurado neste ambiente — validação de erro 502 ok")
		default:
			t.Fatalf("status inesperado %d (%s)", status, body)
		}
	})
}

func assertErrorCode(t *testing.T, body []byte, code string) {
	t.Helper()
	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decodificar erro: %v (%s)", err, body)
	}
	if out.Error != code {
		t.Fatalf("error = %q, esperado %q (%s)", out.Error, code, body)
	}
}

func requestMultipartLogo(t *testing.T, token, establishmentID, filename string, content []byte) (int, []byte) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("logo", filename)
	if err != nil {
		t.Fatalf("criar form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("escrever logo: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("fechar multipart: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, baseURL()+"/api/v1/admin/establishments/"+establishmentID+"/logo", &buf)
	if err != nil {
		t.Fatalf("montar request logo: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return do(t, req)
}

// minimalPNG is a 1x1 transparent PNG.
func minimalPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
}