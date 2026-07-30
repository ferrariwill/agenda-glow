//go:build regression

package regression_test

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"
)

func TestAgenda_CriarColisaoEFinanceiro(t *testing.T) {
	db := regressionDB(t)
	dona := newClient(t)
	dona.login("dona@glow.local")
	boot := dona.mustJSON(http.MethodGet, "/api/v1/bootstrap", http.StatusOK, nil)
	profs, _ := boot["profissionais"].([]any)
	servicos, _ := boot["servicos"].([]any)
	if len(profs) == 0 || len(servicos) == 0 {
		t.Fatal("bootstrap sem profissional/serviço")
	}
	prof := profs[0].(map[string]any)
	serv := servicos[0].(map[string]any)
	profID, _ := prof["id"].(string)
	servicoID, _ := serv["id"].(string)
	comissaoPct, _ := prof["comissao_porcentagem"].(float64)
	precoBase, _ := serv["preco_base"].(float64)

	var adicionalID string
	var precoAdd float64
	if adds, ok := serv["adicionais"].([]any); ok && len(adds) > 0 {
		ad := adds[0].(map[string]any)
		adicionalID, _ = ad["id"].(string)
		precoAdd, _ = ad["preco_adicional"].(float64)
	}

	data := futureWeekdayDate(120 + int(time.Now().UnixNano()%60))
	ag1Body := map[string]any{
		"cliente_nome": "QA Ana", "cliente_telefone": uniquePhone(),
		"profissional_id": profID, "servico_id": servicoID,
		"adicional_ids": []string{}, "data": data, "hora_inicio": "09:00",
		"aceita_adiantar": false,
	}
	if adicionalID != "" {
		ag1Body["adicional_ids"] = []string{adicionalID}
	}

	var ag1 map[string]any
	var horaBase string
	for _, h := range []string{"09:00", "09:30", "10:00", "10:30", "11:00", "11:30", "13:00", "13:30"} {
		ag1Body["hora_inicio"] = h
		st, body := dona.do(http.MethodPost, "/api/v1/appointments", nil, ag1Body)
		if st == http.StatusCreated {
			if err := json.Unmarshal([]byte(body), &ag1); err != nil {
				t.Fatalf("decode ag1: %v", err)
			}
			horaBase = h
			break
		}
		if st != http.StatusConflict {
			t.Fatalf("criar ag1: %d %s", st, body)
		}
	}
	ag1ID, _ := ag1["id"].(string)
	if ag1ID == "" {
		t.Fatalf("não encontrou horário livre em %s", data)
	}
	createdIDs := []string{ag1ID}
	t.Cleanup(func() {
		for _, id := range createdIDs {
			_, _ = db.Exec(`DELETE FROM agendamento_adicionais WHERE agendamento_id=$1`, id)
			_, _ = db.Exec(`DELETE FROM agendamentos WHERE id=$1`, id)
		}
	})

	statusCol, bodyCol := dona.do(http.MethodPost, "/api/v1/appointments", nil, map[string]any{
		"cliente_nome": "QA Beatriz", "cliente_telefone": uniquePhone(),
		"profissional_id": profID, "servico_id": servicoID,
		"adicional_ids": []string{}, "data": data, "hora_inicio": horaBase,
	})
	if statusCol != http.StatusConflict {
		t.Fatalf("colisão total: want 409 got %d body=%s", statusCol, bodyCol)
	}

	baseMin := parseHHMM(horaBase)
	agAnchor, ok := tryCreateAppointment(dona, map[string]any{
		"cliente_nome": "QA Carla", "cliente_telefone": uniquePhone(),
		"profissional_id": profID, "servico_id": servicoID,
		"adicional_ids": []string{}, "data": data, "hora_inicio": formatHHMM(baseMin + 120),
	})
	if !ok {
		t.Skip("horário âncora ocupado; colisão total já validada")
	}
	if agAnchor["status"] != "AGENDADO" {
		t.Fatalf("âncora: want AGENDADO got %v", agAnchor["status"])
	}
	if id, _ := agAnchor["id"].(string); id != "" {
		createdIDs = append(createdIDs, id)
	}
	agEncaixe, ok := tryCreateAppointment(dona, map[string]any{
		"cliente_nome": "QA Diana", "cliente_telefone": uniquePhone(),
		"profissional_id": profID, "servico_id": servicoID,
		"adicional_ids": []string{}, "data": data, "hora_inicio": formatHHMM(baseMin + 105),
	})
	if !ok || agEncaixe["status"] != "EM_APROVACAO" {
		t.Fatalf("encaixe parcial inválido: %v", agEncaixe)
	}
	if minInv, _ := agEncaixe["minutos_invadidos"].(float64); minInv < 1 {
		t.Fatalf("minutos_invadidos: got %v want >=1", agEncaixe["minutos_invadidos"])
	}
	encaixeID, _ := agEncaixe["id"].(string)
	createdIDs = append(createdIDs, encaixeID)
	if status, body := dona.do(http.MethodPost, "/api/v1/public/appointments/"+encaixeID+"/approve", nil, nil); status != http.StatusOK {
		t.Fatalf("approve: %d %s", status, body)
	}

	if status, body := dona.do(http.MethodPost, "/api/v1/webhook/whatsapp-callback", map[string]string{
		"X-API-Key": gatewayKey(),
	}, map[string]any{
		"sistema_origem": "beleza", "tenant_id": seedTenantID,
		"phone_number": ag1Body["cliente_telefone"], "text": "APPT_CONFIRM",
		"event_type": "button_reply", "action": "CONFIRM", "appointment_id": ag1ID,
	}); status != http.StatusOK {
		t.Fatalf("whatsapp confirm: %d %s", status, body)
	}

	charge := dona.mustJSON(http.MethodPost, "/api/v1/appointments/"+ag1ID+"/charge", http.StatusOK, map[string]any{
		"metodo": "DINHEIRO", "concluir": true,
	})
	if charge["status"] != "charged" {
		t.Fatalf("charge: %v", charge)
	}
	wantEntrada := precoBase + precoAdd
	wantComissao := math.Round(wantEntrada*(comissaoPct/100)*100) / 100

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	end := now.Format("2006-01-02")
	if status, body := dona.do(http.MethodGet,
		fmt.Sprintf("/api/v1/finance/professionals/%s/pending?start_date=%s&end_date=%s", profID, start, end),
		nil, nil); status != http.StatusOK {
		t.Fatalf("pending: %d %s", status, body)
	}
	if status, body := dona.do(http.MethodGet,
		fmt.Sprintf("/api/v1/finance/report?start_date=%s&end_date=%s", start, end),
		nil, nil); status != http.StatusOK {
		t.Fatalf("finance report: %d %s", status, body)
	}

	boot2 := dona.mustJSON(http.MethodGet, "/api/v1/bootstrap", http.StatusOK, nil)
	lancs, _ := boot2["lancamentos"].([]any)
	var foundEntrada, foundComissao bool
	for _, l := range lancs {
		lm, _ := l.(map[string]any)
		tipo, _ := lm["tipo"].(string)
		valor, _ := lm["valor"].(float64)
		foundEntrada = foundEntrada || tipo == "ENTRADA" && math.Abs(valor-wantEntrada) <= 0.009
		foundComissao = foundComissao || tipo == "CUSTO_VARIAVEL" && math.Abs(valor-wantComissao) <= 0.009
	}
	if !foundEntrada || !foundComissao {
		t.Fatalf("lançamentos incompletos: entrada=%v comissão=%v want %.2f/%.2f",
			foundEntrada, foundComissao, wantEntrada, wantComissao)
	}
	if status, body := dona.do(http.MethodPost, "/api/v1/appointments/"+ag1ID+"/charge", nil,
		map[string]any{"metodo": "DINHEIRO", "concluir": true}); status != http.StatusConflict {
		t.Fatalf("re-charge: want 409 got %d %s", status, body)
	}
}

func tryCreateAppointment(c *apiClient, body map[string]any) (map[string]any, bool) {
	c.t.Helper()
	st, raw := c.do(http.MethodPost, "/api/v1/appointments", nil, body)
	if st != http.StatusCreated {
		return nil, false
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		c.t.Fatalf("decode appointment: %v", err)
	}
	return out, true
}

func parseHHMM(s string) int {
	var h, m int
	fmt.Sscanf(s, "%d:%d", &h, &m)
	return h*60 + m
}

func formatHHMM(mins int) string {
	return fmt.Sprintf("%02d:%02d", mins/60, mins%60)
}

func TestAgenda_SlotsPublicos(t *testing.T) {
	c := newClient(t)
	path := fmt.Sprintf("/api/v1/public/%s/slots?data=%s&profissional_id=%s&procedimento_id=%s",
		seedSlug, futureWeekdayDate(5), seedProfID, seedServicoID)
	status, body := c.do(http.MethodGet, path, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("slots: %d %s", status, body)
	}
}

func TestFinanceiro_LancamentoManual(t *testing.T) {
	c := newClient(t)
	c.login("dona@glow.local")
	out := c.mustJSON(http.MethodPost, "/api/v1/cash-flow", http.StatusCreated, map[string]any{
		"tipo": "CUSTO_FIXO", "descricao": "QA aluguel regressivo", "valor": 10.50,
	})
	if out["status"] != "ok" {
		t.Fatalf("lancamento: %v", out)
	}
	if st, body := c.do(http.MethodPost, "/api/v1/cash-flow", nil, map[string]any{
		"tipo": "ENTRADA", "descricao": "", "valor": 5,
	}); st == http.StatusCreated {
		t.Fatalf("descrição vazia deveria falhar: %s", body)
	}
}
