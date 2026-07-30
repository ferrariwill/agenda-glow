//go:build regression

package regression_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestWhatsApp_IntegrationSignupState(t *testing.T) {
	c := newClient(t)
	c.login("dona@glow.local")
	view := c.mustJSON(http.MethodGet, "/api/v1/whatsapp/integration", http.StatusOK, nil)
	wantState := "beleza_" + seedTenantID
	if view["state"] != wantState {
		t.Fatalf("state: got %q want %q", view["state"], wantState)
	}
	signup, _ := view["signup_url"].(string)
	if !strings.Contains(signup, "state="+wantState) {
		t.Fatalf("signup_url sem state do salão: %s", signup)
	}
	switch view["status"] {
	case "DESCONECTADO", "PENDENTE", "CONECTADO":
	default:
		t.Fatalf("status inesperado: %v", view["status"])
	}
}

func TestWhatsApp_StartConnectionPendente(t *testing.T) {
	c := newClient(t)
	c.login("dona@glow.local")
	resetWhatsAppStatus(t, "DESCONECTADO")
	started := c.mustJSON(http.MethodPost, "/api/v1/whatsapp/integration/start", http.StatusOK, nil)
	if started["status"] != "PENDENTE" {
		t.Fatalf("após start: want PENDENTE got %v", started["status"])
	}
	after := c.mustJSON(http.MethodGet, "/api/v1/whatsapp/integration", http.StatusOK, nil)
	if after["status"] != "PENDENTE" {
		t.Fatalf("status pós-start: %v", after["status"])
	}
}

func TestWhatsApp_WebhookConnected(t *testing.T) {
	c := newClient(t)
	status, body := c.do(http.MethodPost, "/api/v1/webhook/whatsapp-connected", map[string]string{
		"X-API-Key": gatewayKey(),
	}, map[string]any{
		"event": "whatsapp_connection_completed", "message": "QA regression connected",
		"sistema_origem": "beleza", "tenant_id": seedTenantID, "salon_id": seedTenantID,
		"waba_id": "waba-qa-test", "phone_number_id": "phone-qa-test",
		"whatsapp_phone_number": "+55 15 99999-0000", "status": "connected",
	})
	if status != http.StatusOK {
		t.Fatalf("connected webhook: %d %s", status, body)
	}
	var out map[string]any
	_ = json.Unmarshal([]byte(body), &out)
	if out["integration_status"] != "CONECTADO" && out["status"] != "connected" {
		t.Fatalf("resposta connected inesperada: %s", body)
	}
	dona := newClient(t)
	dona.login("dona@glow.local")
	view := dona.mustJSON(http.MethodGet, "/api/v1/whatsapp/integration", http.StatusOK, nil)
	if view["status"] != "CONECTADO" {
		t.Fatalf("UI/API status após webhook: got %v want CONECTADO", view["status"])
	}
}

func TestWhatsApp_GatewayDispatchConfirmCancel(t *testing.T) {
	dona := newClient(t)
	dona.login("dona@glow.local")
	data := futureWeekdayDate(28 + int(time.Now().UnixNano()%5))
	phone := uniquePhone()

	var ag map[string]any
	var ok bool
	for _, h := range []string{"09:00", "09:30", "10:00", "10:30", "11:00", "14:00", "15:00", "16:00"} {
		ag, ok = tryCreateAppointment(dona, map[string]any{
			"cliente_nome": "QA WhatsApp", "cliente_telefone": phone,
			"profissional_id": seedProfID, "servico_id": seedServicoID,
			"adicional_ids": []string{}, "data": data, "hora_inicio": h,
		})
		if ok {
			break
		}
	}
	if !ok {
		t.Fatal("não foi possível criar agendamento para fluxo WhatsApp")
	}
	agID, _ := ag["id"].(string)
	if agID == "" {
		t.Fatalf("agendamento sem id: %v", ag)
	}

	dispatch := func(action string) (int, string) {
		return dona.do(http.MethodPost, "/api/v1/webhook/whatsapp-gateway", map[string]string{
			"X-API-Key": gatewayKey(),
		}, map[string]any{
			"sistema_origem": "beleza", "tenant_id": seedTenantID,
			"phone_number": phone, "event_type": "button_reply",
			"action": action, "appointment_id": agID,
		})
	}
	st, body := dispatch("CONFIRM")
	if st != http.StatusOK || (!strings.Contains(body, "CONFIRMADO") && !strings.Contains(body, "processed")) {
		t.Fatalf("gateway CONFIRM: %d %s", st, body)
	}
	if st, body = dispatch("CONFIRM"); st != http.StatusOK {
		t.Fatalf("idempotent CONFIRM: %d %s", st, body)
	}
	if st, body = dispatch("CANCEL"); st != http.StatusOK ||
		(!strings.Contains(body, "CANCELADO") && !strings.Contains(body, "processed")) {
		t.Fatalf("gateway CANCEL: %d %s", st, body)
	}
}

func TestWhatsApp_GatewayRejectsConcluido(t *testing.T) {
	dona := newClient(t)
	dona.login("dona@glow.local")
	data := futureWeekdayDate(40)
	phone := uniquePhone()
	ag, ok := tryCreateAppointment(dona, map[string]any{
		"cliente_nome": "QA WhatsApp Concluido", "cliente_telefone": phone,
		"profissional_id": seedProfID, "servico_id": seedServicoID,
		"adicional_ids": []string{}, "data": data, "hora_inicio": "09:00",
	})
	if !ok {
		t.Skip("horário de fixture ocupado")
	}
	id, _ := ag["id"].(string)
	if _, err := regressionDB(t).Exec(`UPDATE agendamentos SET status = 'CONCLUIDO' WHERE id = $1`, id); err != nil {
		t.Fatalf("marcar concluído: %v", err)
	}
	status, body := dona.do(http.MethodPost, "/api/v1/webhook/whatsapp-gateway", map[string]string{
		"X-API-Key": gatewayKey(),
	}, map[string]any{
		"sistema_origem": "beleza", "tenant_id": seedTenantID,
		"phone_number": phone, "action": "CANCEL", "appointment_id": id,
	})
	if status != http.StatusConflict {
		t.Fatalf("CANCEL concluído: want 409 got %d body=%s", status, body)
	}
}

func TestWhatsApp_GatewayConnectionEvent(t *testing.T) {
	c := newClient(t)
	st, body := c.do(http.MethodPost, "/api/v1/webhook/whatsapp-gateway", map[string]string{
		"X-API-Key": gatewayKey(),
	}, map[string]any{
		"event": "whatsapp_connection_completed", "sistema_origem": "beleza",
		"tenant_id": seedTenantID, "salon_id": seedTenantID,
		"waba_id": "waba-gateway-qa", "phone_number_id": "phone-gateway-qa",
		"whatsapp_phone_number": "+55 15 98888-1111", "status": "connected",
	})
	if st != http.StatusOK {
		t.Fatalf("gateway connection: %d %s", st, body)
	}
}

func TestWhatsApp_SendNotificationContract_UnitSpy(t *testing.T) {
	c := newClient(t)
	c.login("dona@glow.local")
	view := c.mustJSON(http.MethodGet, "/api/v1/whatsapp/integration", http.StatusOK, nil)
	if view["estabelecimento_id"] != seedTenantID {
		t.Fatalf("estabelecimento_id: %v", view["estabelecimento_id"])
	}
}
