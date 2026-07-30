//go:build regression

package regression_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

const (
	defaultAPIBase = "http://localhost:8081"
	seedTenantID   = "a1000002-0002-4002-8002-000000000002"
	seedProfID     = "a1000003-0003-4003-8003-000000000003"
	seedServicoID  = "a1000004-0004-4004-8004-000000000004"
	seedAdicional  = "a1000005-0005-4005-8005-000000000005"
	seedSlug       = "estudio-glow-salto"
	devPassword    = "AgendaGlow@2026"
)

func apiBase() string {
	if v := os.Getenv("AGENDA_GLOW_API"); v != "" {
		return v
	}
	return defaultAPIBase
}

func gatewayKey() string {
	if v := os.Getenv("WHATSAPP_GATEWAY_KEY"); v != "" {
		return v
	}
	return "test-key-regression"
}

type apiClient struct {
	t     *testing.T
	token string
	http  *http.Client
}

func newClient(t *testing.T) *apiClient {
	t.Helper()
	return &apiClient{t: t, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *apiClient) login(email string) map[string]any {
	c.t.Helper()
	status, body := c.do(http.MethodPost, "/api/v1/auth/login", nil, map[string]string{
		"email": email, "password": devPassword,
	})
	if status != http.StatusOK {
		c.t.Fatalf("login %s: status %d body %s", email, status, body)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		c.t.Fatalf("decode login: %v", err)
	}
	token, _ := out["token"].(string)
	if token == "" {
		c.t.Fatalf("login sem token: %s", body)
	}
	c.token = token
	return out
}

func (c *apiClient) do(method, path string, headers map[string]string, payload any) (int, string) {
	c.t.Helper()
	var reader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			c.t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, apiBase()+path, reader)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v (API em %s está no ar?)", method, path, err, apiBase())
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func (c *apiClient) mustJSON(method, path string, wantStatus int, payload any) map[string]any {
	c.t.Helper()
	status, body := c.do(method, path, nil, payload)
	if status != wantStatus {
		c.t.Fatalf("%s %s: got %d want %d body=%s", method, path, status, wantStatus, body)
	}
	if body == "" || body == "null" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		var arr []any
		if err2 := json.Unmarshal([]byte(body), &arr); err2 == nil {
			return map[string]any{"_array": arr}
		}
		c.t.Fatalf("decode %s: %v body=%s", path, err, body)
	}
	return out
}

func assertMoney(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.009 {
		t.Fatalf("%s: got %.4f want %.2f", label, got, want)
	}
}

func futureWeekdayDate(daysAhead int) string {
	d := time.Now().AddDate(0, 0, daysAhead)
	for d.Weekday() == time.Sunday || d.Weekday() == time.Saturday {
		d = d.AddDate(0, 0, 1)
	}
	return d.Format("2006-01-02")
}

func uniquePhone() string {
	return fmt.Sprintf("55159%08d", time.Now().UnixNano()%100000000)
}

func regressionDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://postgres:glow_secure_pwd_2026@localhost:5435/agenda_glow_prod?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("ping db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func resetWhatsAppStatus(t *testing.T, status string) {
	t.Helper()
	if _, err := regressionDB(t).Exec(
		`UPDATE estabelecimentos SET whatsapp_status = $1 WHERE id = $2`,
		status, seedTenantID,
	); err != nil {
		t.Fatalf("reset whatsapp_status: %v", err)
	}
}
