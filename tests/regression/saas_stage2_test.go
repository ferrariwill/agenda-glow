//go:build regression

package regression_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSaaSStage2_31Cases(t *testing.T) {
	db := regressionDB(t)
	sa := newClient(t)
	sa.login("ferrariwill@gmail.com")
	stamp := time.Now().UnixNano()
	planName := fmt.Sprintf("QA Stage2 Plan %d", stamp)

	var planID, estID, espID, inactiveEspID, activeProfID, reactivationProfID string
	var serviceID, additionalID, appointmentID string
	var owner *apiClient
	t.Cleanup(func() {
		if appointmentID != "" {
			_, _ = db.Exec(`DELETE FROM agendamento_adicionais WHERE agendamento_id=$1`, appointmentID)
			_, _ = db.Exec(`DELETE FROM agendamentos WHERE id=$1`, appointmentID)
		}
		if estID != "" {
			_, _ = db.Exec(`DELETE FROM estabelecimentos WHERE id=$1`, estID)
		}
		if planID != "" {
			_, _ = db.Exec(`DELETE FROM planos_saas WHERE id=$1`, planID)
		}
	})

	run := func(id, name string, fn func(t *testing.T)) {
		t.Run(id+"_"+name, fn)
	}
	assertStatus := func(t *testing.T, got, want int, body string) {
		t.Helper()
		if got != want {
			t.Fatalf("status=%d want=%d body=%s", got, want, body)
		}
	}
	decodeID := func(t *testing.T, body string) string {
		t.Helper()
		var out struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(body), &out); err != nil || out.ID == "" {
			t.Fatalf("resposta sem id: %s (%v)", body, err)
		}
		return out.ID
	}

	run("S2-00", "login_dona_e_super_admin", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/api/v1/auth/login", nil, map[string]string{
			"email": "dona@glow.local", "password": devPassword,
		})
		assertStatus(t, status, http.StatusOK, body)
	})
	run("S2-01", "criar_plano", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/api/v1/admin/plans", nil, map[string]any{
			"nome": planName, "preco_mensal": 97, "limite_profissionais": 3,
		})
		assertStatus(t, status, http.StatusCreated, body)
		planID = decodeID(t, body)
	})
	run("S2-02", "plano_nome_unico_case_insensitive", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/api/v1/admin/plans", nil, map[string]any{
			"nome": strings.ToUpper(planName), "preco_mensal": 10, "limite_profissionais": 2,
		})
		assertStatus(t, status, http.StatusConflict, body)
	})
	run("S2-03", "plano_limite_invalido", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/api/v1/admin/plans", nil, map[string]any{
			"nome": fmt.Sprintf("QA Bad Limit %d", stamp), "preco_mensal": 10, "limite_profissionais": 0,
		})
		assertStatus(t, status, http.StatusBadRequest, body)
	})
	run("S2-04", "plano_preco_negativo", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/api/v1/admin/plans", nil, map[string]any{
			"nome": fmt.Sprintf("QA Bad Price %d", stamp), "preco_mensal": -1, "limite_profissionais": 1,
		})
		if status == http.StatusCreated || status == 0 {
			t.Fatalf("preço negativo foi aceito: status=%d body=%s", status, body)
		}
	})
	run("S2-05", "criar_salao", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/api/v1/admin/establishments", nil, map[string]any{
			"nome_comercial": fmt.Sprintf("QA Stage2 Salão %d", stamp),
			"slug":           fmt.Sprintf("qa-stage2-%d", stamp),
		})
		assertStatus(t, status, http.StatusCreated, body)
		estID = decodeID(t, body)
	})
	run("S2-06", "atribuir_plano_meses_invalidos", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/api/v1/admin/establishments/"+estID+"/assign-plan", nil,
			map[string]any{"plano_id": planID, "meses": 0})
		assertStatus(t, status, http.StatusBadRequest, body)
	})
	run("S2-07", "atribuir_plano_ativo", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/api/v1/admin/establishments/"+estID+"/assign-plan", nil,
			map[string]any{"plano_id": planID, "meses": 1})
		assertStatus(t, status, http.StatusOK, body)
		var got string
		if err := db.QueryRow(`SELECT status FROM assinaturas_estabelecimentos WHERE estabelecimento_id=$1`, estID).Scan(&got); err != nil || got != "ATIVO" {
			t.Fatalf("assinatura: status=%q err=%v", got, err)
		}
	})
	var firstExpiry time.Time
	run("S2-08", "renovacao_soma_vencimento", func(t *testing.T) {
		if err := db.QueryRow(`SELECT data_vencimento FROM assinaturas_estabelecimentos WHERE estabelecimento_id=$1`, estID).Scan(&firstExpiry); err != nil {
			t.Fatal(err)
		}
		status, body := sa.do(http.MethodPost, "/api/v1/admin/establishments/"+estID+"/assign-plan", nil,
			map[string]any{"plano_id": planID, "meses": 1})
		assertStatus(t, status, http.StatusOK, body)
		var second time.Time
		if err := db.QueryRow(`SELECT data_vencimento FROM assinaturas_estabelecimentos WHERE estabelecimento_id=$1`, estID).Scan(&second); err != nil ||
			second.Before(firstExpiry.AddDate(0, 1, -1)) {
			t.Fatalf("vencimento inicial=%s renovado=%s err=%v", firstExpiry, second, err)
		}
	})
	run("S2-09", "suspender", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/superadmin/establishments/"+estID+"/suspend", nil, nil)
		assertStatus(t, status, http.StatusOK, body)
		var ativo bool
		var assinatura string
		err := db.QueryRow(`SELECT e.ativo, a.status FROM estabelecimentos e
			JOIN assinaturas_estabelecimentos a ON a.estabelecimento_id=e.id WHERE e.id=$1`, estID).
			Scan(&ativo, &assinatura)
		if err != nil || ativo || assinatura != "SUSPENSO" {
			t.Fatalf("ativo=%v assinatura=%s err=%v", ativo, assinatura, err)
		}
	})
	run("S2-10", "ativar", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/superadmin/establishments/"+estID+"/activate", nil, nil)
		assertStatus(t, status, http.StatusOK, body)
		ownerEmail := fmt.Sprintf("qa.stage2.%d@glow.local", stamp)
		_, err := db.Exec(`INSERT INTO users (email,password_hash,role,estabelecimento_id,ativo)
			SELECT $1,password_hash,'DONA',$2,true FROM users WHERE email='dona@glow.local'`, ownerEmail, estID)
		if err != nil {
			t.Fatalf("criar dona isolada: %v", err)
		}
		owner = newClient(t)
		owner.login(ownerEmail)
	})
	run("S2-11", "pagamento_pendente_nao_bloqueia", func(t *testing.T) {
		_, _ = sa.do(http.MethodPost, "/superadmin/establishments/"+estID+"/suspend", nil, nil)
		if _, err := db.Exec(`UPDATE estabelecimentos SET ativo=true WHERE id=$1`, estID); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE assinaturas_estabelecimentos SET status='PAGAMENTO_PENDENTE',
			data_vencimento=CURRENT_DATE+60 WHERE estabelecimento_id=$1`, estID); err != nil {
			t.Fatal(err)
		}
		status, body := owner.do(http.MethodGet, "/api/v1/services", nil, nil)
		assertStatus(t, status, http.StatusOK, body)
	})
	run("S2-12", "vencido_bloqueia_402", func(t *testing.T) {
		_, _ = sa.do(http.MethodPost, "/superadmin/establishments/"+estID+"/activate", nil, nil)
		if _, err := db.Exec(`UPDATE assinaturas_estabelecimentos SET status='ATIVO',
			data_vencimento=CURRENT_DATE-2 WHERE estabelecimento_id=$1`, estID); err != nil {
			t.Fatal(err)
		}
		status, body := owner.do(http.MethodGet, "/api/v1/services", nil, nil)
		assertStatus(t, status, http.StatusPaymentRequired, body)
	})
	run("S2-13", "suspenso_bloqueia_402", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/superadmin/establishments/"+estID+"/suspend", nil, nil)
		assertStatus(t, status, http.StatusOK, body)
		status, body = owner.do(http.MethodGet, "/api/v1/services", nil, nil)
		assertStatus(t, status, http.StatusPaymentRequired, body)
	})
	run("S2-14", "restore_acesso", func(t *testing.T) {
		status, body := sa.do(http.MethodPost, "/superadmin/establishments/"+estID+"/activate", nil, nil)
		assertStatus(t, status, http.StatusOK, body)
		if _, err := db.Exec(`UPDATE assinaturas_estabelecimentos SET status='ATIVO',
			data_vencimento=CURRENT_DATE+60 WHERE estabelecimento_id=$1`, estID); err != nil {
			t.Fatal(err)
		}
		status, body = owner.do(http.MethodGet, "/api/v1/services", nil, nil)
		assertStatus(t, status, http.StatusOK, body)
	})
	run("S2-15", "post_acima_limite_403", func(t *testing.T) {
		if _, err := db.Exec(`UPDATE planos_saas SET limite_profissionais=1 WHERE id=$1`, planID); err != nil {
			t.Fatal(err)
		}
		status, body := owner.do(http.MethodPost, "/api/v1/specialties", nil, map[string]any{"nome": fmt.Sprintf("QA Esp %d", stamp)})
		assertStatus(t, status, http.StatusCreated, body)
		espID = decodeID(t, body)
		status, body = owner.do(http.MethodPost, "/api/v1/professionals", nil, map[string]any{
			"nome": "QA Prof Um", "especialidade_id": espID, "comissao_porcentagem": 40,
		})
		assertStatus(t, status, http.StatusCreated, body)
		activeProfID = decodeID(t, body)
		status, body = owner.do(http.MethodPost, "/api/v1/professionals", nil, map[string]any{
			"nome": "QA Prof Acima Limite", "especialidade_id": espID, "comissao_porcentagem": 40,
		})
		assertLimitReached(t, status, body)
	})
	run("S2-16", "criar_especialidade", func(t *testing.T) {
		status, body := owner.do(http.MethodPost, "/api/v1/specialties", nil, map[string]any{"nome": fmt.Sprintf("QA Outra %d", stamp)})
		assertStatus(t, status, http.StatusCreated, body)
	})
	run("S2-17", "especialidade_unica_case_insensitive", func(t *testing.T) {
		name := fmt.Sprintf("QA Única %d", stamp)
		status, body := owner.do(http.MethodPost, "/api/v1/specialties", nil, map[string]any{"nome": name})
		assertStatus(t, status, http.StatusCreated, body)
		status, body = owner.do(http.MethodPost, "/api/v1/specialties", nil, map[string]any{"nome": strings.ToUpper(name)})
		if status != http.StatusBadRequest && status != http.StatusConflict {
			t.Fatalf("status=%d want=400/409 body=%s", status, body)
		}
	})
	run("S2-18", "profissional_exige_especialidade_ativa", func(t *testing.T) {
		if _, err := db.Exec(`UPDATE planos_saas SET limite_profissionais=5 WHERE id=$1`, planID); err != nil {
			t.Fatal(err)
		}
		status, body := owner.do(http.MethodPost, "/api/v1/specialties", nil,
			map[string]any{"nome": fmt.Sprintf("QA Inativa %d", stamp)})
		assertStatus(t, status, http.StatusCreated, body)
		inactiveEspID = decodeID(t, body)
		status, body = owner.do(http.MethodPut, "/api/v1/specialties/"+inactiveEspID, nil,
			map[string]any{"nome": fmt.Sprintf("QA Inativa %d", stamp), "ativo": false})
		assertStatus(t, status, http.StatusOK, body)
		status, body = owner.do(http.MethodPost, "/api/v1/professionals", nil, map[string]any{
			"nome": "QA Prof Inativa", "especialidade_id": inactiveEspID, "comissao_porcentagem": 30,
		})
		assertStatus(t, status, http.StatusBadRequest, body)
	})
	run("S2-19", "comissao_negativa", func(t *testing.T) {
		status, body := owner.do(http.MethodPost, "/api/v1/professionals", nil, map[string]any{
			"nome": "QA Comissão Negativa", "especialidade_id": espID, "comissao_porcentagem": -1,
		})
		assertStatus(t, status, http.StatusBadRequest, body)
	})
	run("S2-20", "comissao_acima_100", func(t *testing.T) {
		status, body := owner.do(http.MethodPost, "/api/v1/professionals", nil, map[string]any{
			"nome": "QA Comissão Alta", "especialidade_id": espID, "comissao_porcentagem": 101,
		})
		assertStatus(t, status, http.StatusBadRequest, body)
	})
	run("S2-21", "comissao_55_aceita", func(t *testing.T) {
		if _, err := db.Exec(`UPDATE planos_saas SET limite_profissionais=2 WHERE id=$1`, planID); err != nil {
			t.Fatal(err)
		}
		status, body := owner.do(http.MethodPost, "/api/v1/professionals", nil, map[string]any{
			"nome": "QA Comissão 55", "especialidade_id": espID, "comissao_porcentagem": 55,
		})
		assertStatus(t, status, http.StatusCreated, body)
		reactivationProfID = decodeID(t, body)
	})
	run("S2-22", "put_reativacao_acima_limite_403", func(t *testing.T) {
		status, body := owner.do(http.MethodPut, "/api/v1/professionals/"+reactivationProfID, nil, map[string]any{
			"nome": "QA Comissão 55", "especialidade_id": espID, "comissao_porcentagem": 55, "ativo": false,
		})
		assertStatus(t, status, http.StatusOK, body)
		if _, err := db.Exec(`UPDATE planos_saas SET limite_profissionais=1 WHERE id=$1`, planID); err != nil {
			t.Fatal(err)
		}
		status, body = owner.do(http.MethodPut, "/api/v1/professionals/"+reactivationProfID, nil, map[string]any{
			"nome": "QA Comissão 55", "especialidade_id": espID, "comissao_porcentagem": 55, "ativo": true,
		})
		assertLimitReached(t, status, body)
	})
	run("S2-23", "criar_servico", func(t *testing.T) {
		status, body := owner.do(http.MethodPost, "/api/v1/services", nil, map[string]any{
			"nome": fmt.Sprintf("QA Serviço %d", stamp), "preco_base": 80.5, "duracao_base_minutos": 40,
		})
		assertStatus(t, status, http.StatusCreated, body)
		serviceID = decodeID(t, body)
	})
	run("S2-24", "servico_duracao_invalida", func(t *testing.T) {
		status, body := owner.do(http.MethodPost, "/api/v1/services", nil, map[string]any{
			"nome": "QA Duração Inválida", "preco_base": 10, "duracao_base_minutos": 0,
		})
		assertStatus(t, status, http.StatusBadRequest, body)
	})
	run("S2-25", "servico_preco_negativo", func(t *testing.T) {
		status, body := owner.do(http.MethodPost, "/api/v1/services", nil, map[string]any{
			"nome": "QA Preço Inválido", "preco_base": -5, "duracao_base_minutos": 30,
		})
		assertStatus(t, status, http.StatusBadRequest, body)
	})
	run("S2-26", "criar_adicional", func(t *testing.T) {
		status, body := owner.do(http.MethodPost, "/api/v1/services/"+serviceID+"/additionals", nil, map[string]any{
			"nome": "QA Adicional", "preco_adicional": 19.5, "duracao_adicional_minutos": 15,
		})
		assertStatus(t, status, http.StatusCreated, body)
		additionalID = decodeID(t, body)
	})
	run("S2-27", "adicional_servico_inexistente", func(t *testing.T) {
		status, body := owner.do(http.MethodPost,
			"/api/v1/services/00000000-0000-4000-8000-000000000099/additionals", nil, map[string]any{
				"nome": "QA Órfão", "preco_adicional": 1, "duracao_adicional_minutos": 1,
			})
		if status != http.StatusNotFound && status != http.StatusBadRequest {
			t.Fatalf("status=%d want=400/404 body=%s", status, body)
		}
	})
	run("S2-28", "agendamento_soma_duracao", func(t *testing.T) {
		status, body := owner.do(http.MethodPost, "/api/v1/appointments", nil, map[string]any{
			"cliente_nome": "QA Cliente", "cliente_telefone": uniquePhone(),
			"profissional_id": activeProfID, "servico_id": serviceID,
			"adicional_ids": []string{additionalID}, "data": futureWeekdayDate(14), "hora_inicio": "10:00",
		})
		assertStatus(t, status, http.StatusCreated, body)
		appointmentID = decodeID(t, body)
		var duration float64
		if err := db.QueryRow(`SELECT EXTRACT(EPOCH FROM (data_hora_fim-data_hora_inicio))/60
			FROM agendamentos WHERE id=$1`, appointmentID).Scan(&duration); err != nil || duration != 55 {
			t.Fatalf("duração=%.2f want=55 err=%v", duration, err)
		}
	})
	run("S2-29", "preco_base_mais_adicional", func(t *testing.T) {
		var total float64
		err := db.QueryRow(`SELECT s.preco_base+a.preco_adicional FROM servicos s
			JOIN servico_adicionais a ON a.servico_id=s.id WHERE s.id=$1 AND a.id=$2`,
			serviceID, additionalID).Scan(&total)
		if err != nil || total != 100 {
			t.Fatalf("total=%.2f want=100.00 err=%v", total, err)
		}
	})
	run("S2-30", "adicional_de_outro_servico", func(t *testing.T) {
		status, body := owner.do(http.MethodPost, "/api/v1/appointments", nil, map[string]any{
			"cliente_nome": "QA Adicional Errado", "cliente_telefone": uniquePhone(),
			"profissional_id": activeProfID, "servico_id": serviceID,
			"adicional_ids": []string{seedAdicional}, "data": futureWeekdayDate(15), "hora_inicio": "14:00",
		})
		assertStatus(t, status, http.StatusBadRequest, body)
	})
}

func assertLimitReached(t *testing.T, status int, body string) {
	t.Helper()
	if status != http.StatusForbidden {
		t.Fatalf("status=%d want=403 body=%s", status, body)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(body), &out); err != nil || out["error"] != "limit_reached" {
		t.Fatalf("body sem error=limit_reached: %s", body)
	}
}
