package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

func TestAgendaNotificationsHandlerRejectsClaimPathMismatchBeforeDB(t *testing.T) {
	claimID := "tenant-a"
	claims := &security.Claims{
		UserID: "owner", Email: "owner@example.com", Role: security.RoleDona,
		EstabelecimentoID: &claimID,
	}
	ctx := security.WithSessionClaims(context.Background(), claims)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/estabelecimentos/tenant-b/notificacoes-agenda", nil).
		WithContext(ctx)
	request.SetPathValue("id", "tenant-b")
	response := httptest.NewRecorder()

	NewAgendaNotificationsHandler(service.NewAgendaService(nil)).Get(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403", response.Code)
	}
	if !strings.Contains(response.Body.String(), "establishment_scope_mismatch") {
		t.Fatalf("resposta inesperada: %s", response.Body.String())
	}
}
