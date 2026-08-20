//go:build regression

// Suíte DEV-214 — SuperAdmin JSON create-dona + upload logo.
package regression

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestStage4CreateDonaJSON(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	planoID, _ := criarPlano(t, sa.Token, 97, 1)
	estID, _ := criarSalao(t, sa.Token)
	if status, body := requestJSON(t, http.MethodPost,
		"/api/v1/admin/establishments/"+estID+"/assign-plan", sa.Token,
		map[string]any{"plano_id": planoID, "meses": 12}); status != http.StatusOK {
		t.Fatalf("atribuir plano: status %d (%s)", status, body)
	}

	email := fmt.Sprintf("qa.dev214.%s@glow.local", uniqueSuffix())
	nome := "Maria QA DEV214"

	status, body := requestJSON(t, http.MethodPost,
		"/api/v1/admin/establishments/"+estID+"/create-dona", sa.Token,
		map[string]string{"nome": nome, "email": email})
	if status != http.StatusCreated {
		t.Fatalf("create-dona JSON: status %d (%s)", status, body)
	}

	var created struct {
		UserID       string `json:"user_id"`
		Email        string `json:"email"`
		Nome         string `json:"nome"`
		SenhaInicial string `json:"senha_inicial"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decodificar create-dona: %v (%s)", err, body)
	}
	if created.UserID == "" || created.Email != email || created.Nome != nome {
		t.Fatalf("resposta create-dona inesperada: %+v", created)
	}
	if created.SenhaInicial != devPassword {
		t.Fatalf("senha_inicial esperada %q, obtida %q", devPassword, created.SenhaInicial)
	}

	dona := login(t, email, created.SenhaInicial)
	if dona.Role != "DONA" {
		t.Fatalf("role esperada DONA, obtida %s", dona.Role)
	}
	if dona.User.EstabelecimentoID == nil || *dona.User.EstabelecimentoID != estID {
		t.Fatalf("estabelecimento_id esperado %s, obtido %v", estID, dona.User.EstabelecimentoID)
	}

	var donaNome, donaEmail, userNome string
	if err := db(t).QueryRow(
		`SELECT e.dona_nome, e.dona_email, u.nome
		 FROM estabelecimentos e
		 JOIN users u ON u.estabelecimento_id = e.id AND LOWER(u.email) = LOWER($2)
		 WHERE e.id = $1`,
		estID, email,
	).Scan(&donaNome, &donaEmail, &userNome); err != nil {
		t.Fatalf("ler denormalizado dona: %v", err)
	}
	if donaNome != nome || donaEmail != email || userNome != nome {
		t.Fatalf("persistência nome/email: dona_nome=%q dona_email=%q users.nome=%q", donaNome, donaEmail, userNome)
	}

	t.Run("missing_nome", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/create-dona", sa.Token,
			map[string]string{"nome": "  ", "email": "x@glow.local"})
		assertJSONError(t, status, body, http.StatusBadRequest, "missing_nome")
	})

	t.Run("missing_email", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/create-dona", sa.Token,
			map[string]string{"nome": "Alguém", "email": ""})
		assertJSONError(t, status, body, http.StatusBadRequest, "missing_email")
	})

	t.Run("email_already_exists", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/create-dona", sa.Token,
			map[string]string{"nome": "Outra", "email": strings.ToUpper(email)})
		assertJSONError(t, status, body, http.StatusConflict, "email_already_exists")
	})

	t.Run("not_found", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/44444444-5555-4666-8777-888888888888/create-dona", sa.Token,
			map[string]string{"nome": "Ghost", "email": "ghost-" + uniqueSuffix() + "@glow.local"})
		assertJSONError(t, status, body, http.StatusNotFound, "not_found")
	})

	t.Run("sem_super_admin", func(t *testing.T) {
		status, _ := requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/create-dona", "",
			map[string]string{"nome": "X", "email": "noauth-" + uniqueSuffix() + "@glow.local"})
		if status != http.StatusUnauthorized && status != http.StatusForbidden {
			t.Fatalf("esperado 401/403, obtido %d", status)
		}

		status, _ = requestJSON(t, http.MethodPost,
			"/api/v1/admin/establishments/"+estID+"/create-dona", dona.Token,
			map[string]string{"nome": "X", "email": "forbidden-" + uniqueSuffix() + "@glow.local"})
		if status != http.StatusForbidden && status != http.StatusUnauthorized {
			t.Fatalf("DONA não deve criar dona via admin: status %d", status)
		}
	})
}

func TestStage4UploadLogoJSON(t *testing.T) {
	sa := login(t, superAdminEmail, devPassword)
	estID, _ := criarSalao(t, sa.Token)

	t.Run("missing_logo", func(t *testing.T) {
		status, body := requestMultipart(t,
			"/api/v1/admin/establishments/"+estID+"/logo", sa.Token, nil)
		assertJSONError(t, status, body, http.StatusBadRequest, "missing_logo")
	})

	t.Run("invalid_image_type", func(t *testing.T) {
		status, body := requestMultipart(t,
			"/api/v1/admin/establishments/"+estID+"/logo", sa.Token,
			&multipartFile{field: "logo", filename: "x.gif", content: []byte("GIF89a")})
		assertJSONError(t, status, body, http.StatusBadRequest, "invalid_image_type")
	})

	t.Run("file_too_large", func(t *testing.T) {
		big := bytes.Repeat([]byte("a"), (2<<20)+1)
		status, body := requestMultipart(t,
			"/api/v1/admin/establishments/"+estID+"/logo", sa.Token,
			&multipartFile{field: "logo", filename: "big.png", content: big})
		if status != http.StatusBadRequest {
			t.Fatalf("esperado 400, obtido %d (%s)", status, body)
		}
		code := jsonErrorCode(t, body)
		if code != "file_too_large" && code != "invalid_multipart" {
			t.Fatalf("esperado file_too_large|invalid_multipart, obtido %q (%s)", code, body)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		status, body := requestMultipart(t,
			"/api/v1/admin/establishments/44444444-5555-4666-8777-888888888888/logo", sa.Token,
			&multipartFile{field: "logo", filename: "ok.png", content: tinyPNG()})
		assertJSONError(t, status, body, http.StatusNotFound, "not_found")
	})

	t.Run("sem_super_admin", func(t *testing.T) {
		status, _ := requestMultipart(t,
			"/api/v1/admin/establishments/"+estID+"/logo", "",
			&multipartFile{field: "logo", filename: "ok.png", content: tinyPNG()})
		if status != http.StatusUnauthorized && status != http.StatusForbidden {
			t.Fatalf("esperado 401/403, obtido %d", status)
		}
	})

	t.Run("upload_ok", func(t *testing.T) {
		if strings.TrimSpace(os.Getenv("SUPABASE_URL")) == "" ||
			strings.TrimSpace(os.Getenv("SUPABASE_SERVICE_ROLE_KEY")) == "" {
			t.Skip("SUPABASE_URL/SUPABASE_SERVICE_ROLE_KEY ausentes — skip upload feliz (documentado DEV-214)")
		}
		status, body := requestMultipart(t,
			"/api/v1/admin/establishments/"+estID+"/logo", sa.Token,
			&multipartFile{field: "logo", filename: "logo.png", content: tinyPNG()})
		if status != http.StatusOK {
			t.Fatalf("upload logo: status %d (%s)", status, body)
		}
		var out struct {
			LogoURL string `json:"logo_url"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("decodificar logo: %v (%s)", err, body)
		}
		if strings.TrimSpace(out.LogoURL) == "" {
			t.Fatalf("logo_url vazio: %s", body)
		}
	})
}

type multipartFile struct {
	field    string
	filename string
	content  []byte
}

func requestMultipart(t *testing.T, path, token string, file *multipartFile) (int, []byte) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if file != nil {
		part, err := w.CreateFormFile(file.field, file.filename)
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := part.Write(file.content); err != nil {
			t.Fatalf("escrever arquivo multipart: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("fechar multipart: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL()+path, &buf)
	if err != nil {
		t.Fatalf("montar multipart %s: %v", path, err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return do(t, req)
}

func assertJSONError(t *testing.T, status int, body []byte, wantStatus int, wantCode string) {
	t.Helper()
	if status != wantStatus {
		t.Fatalf("status esperado %d, obtido %d (%s)", wantStatus, status, body)
	}
	if got := jsonErrorCode(t, body); got != wantCode {
		t.Fatalf("error code esperado %q, obtido %q (%s)", wantCode, got, body)
	}
}

func jsonErrorCode(t *testing.T, body []byte) string {
	t.Helper()
	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decodificar erro JSON: %v (%s)", err, body)
	}
	return out.Error
}

// tinyPNG é um PNG 1x1 mínimo válido para testes de upload.
func tinyPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe,
		0xd4, 0xef, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
		0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
