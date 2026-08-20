package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func TestListServicesIncluiProfissionalIDs(t *testing.T) {
	t.Parallel()

	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer raw.Close()
	svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, nome, preco_base, duracao_base_minutos, ativo`)).
		WithArgs("est-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "nome", "preco_base", "duracao_base_minutos", "ativo",
		}).
			AddRow("svc-1", "Corte", 80.0, 45, true).
			AddRow("svc-2", "Escova", 90.0, 60, true))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM servico_adicionais sa`)).
		WithArgs("est-1", pq.Array([]string{"svc-1", "svc-2"})).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "servico_id", "nome", "preco_adicional", "duracao_adicional_minutos",
		}))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM servico_profissionais`)).
		WithArgs("est-1", pq.Array([]string{"svc-1", "svc-2"})).
		WillReturnRows(sqlmock.NewRows([]string{"servico_id", "profissional_id"}).
			AddRow("svc-1", "prof-a").
			AddRow("svc-1", "prof-b"))

	lista, err := svc.ListServices(context.Background(), "est-1")
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if len(lista) != 2 {
		t.Fatalf("len=%d; want 2", len(lista))
	}
	if got := lista[0].ProfissionalIDs; len(got) != 2 || got[0] != "prof-a" || got[1] != "prof-b" {
		t.Fatalf("svc-1 profissional_ids=%v", got)
	}
	if lista[1].ProfissionalIDs == nil || len(lista[1].ProfissionalIDs) != 0 {
		t.Fatalf("svc-2 profissional_ids deve ser []; got %#v", lista[1].ProfissionalIDs)
	}

	rawJSON, err := json.Marshal(lista[1])
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`"profissional_ids":\s*\[\]`).Match(rawJSON) {
		t.Fatalf("JSON deve serializar profissional_ids como array vazio: %s", rawJSON)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateServiceComVinculosAtomico(t *testing.T) {
	t.Parallel()

	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer raw.Close()
	svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO servicos`)).
		WithArgs("est-1", "Escova", 90.0, 60).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("svc-new"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT dona_atua_como_profissional`)).
		WithArgs("est-1").
		WillReturnRows(sqlmock.NewRows([]string{"dona_atua_como_profissional"}).AddRow(false))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, ativo, eh_dona`)).
		WithArgs("est-1", pq.Array([]string{"prof-1", "prof-2"})).
		WillReturnRows(sqlmock.NewRows([]string{"id", "ativo", "eh_dona"}).
			AddRow("prof-1", true, false).
			AddRow("prof-2", true, false))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM servico_profissionais`)).
		WithArgs("est-1", "svc-new").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO servico_profissionais`)).
		WithArgs("est-1", "svc-new", "prof-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO servico_profissionais`)).
		WithArgs("est-1", "svc-new", "prof-2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	id, err := svc.CreateService(context.Background(), "est-1", "Escova", 90, 60, []string{"prof-1", "prof-2", "prof-1"})
	if err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	if id != "svc-new" {
		t.Fatalf("id=%s; want svc-new", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateServiceReplaceSetERejeitaCrossTenant(t *testing.T) {
	t.Parallel()

	t.Run("replace e limpa com array vazio", func(t *testing.T) {
		t.Parallel()
		raw, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New: %v", err)
		}
		defer raw.Close()
		svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM servicos`)).
			WithArgs("svc-1", "est-1").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("svc-1"))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE servicos`)).
			WithArgs("svc-1", "est-1", "Escova", 95.0, 60, true).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM servico_profissionais`)).
			WithArgs("est-1", "svc-1").
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		if err := svc.UpdateService(context.Background(), "est-1", "svc-1", "Escova", 95, 60, true, nil); err != nil {
			t.Fatalf("UpdateService: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("profissional de outro tenant", func(t *testing.T) {
		t.Parallel()
		raw, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New: %v", err)
		}
		defer raw.Close()
		svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM servicos`)).
			WithArgs("svc-1", "est-1").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("svc-1"))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE servicos`)).
			WithArgs("svc-1", "est-1", "Escova", 95.0, 60, true).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT dona_atua_como_profissional`)).
			WithArgs("est-1").
			WillReturnRows(sqlmock.NewRows([]string{"dona_atua_como_profissional"}).AddRow(true))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, ativo, eh_dona`)).
			WithArgs("est-1", pq.Array([]string{"prof-outro"})).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ativo", "eh_dona"}))
		mock.ExpectRollback()

		err = svc.UpdateService(context.Background(), "est-1", "svc-1", "Escova", 95, 60, true, []string{"prof-outro"})
		if !errors.Is(err, ErrInvalidProfessional) {
			t.Fatalf("err=%v; want ErrInvalidProfessional", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("dona sem flag", func(t *testing.T) {
		t.Parallel()
		raw, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New: %v", err)
		}
		defer raw.Close()
		svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM servicos`)).
			WithArgs("svc-1", "est-1").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("svc-1"))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE servicos`)).
			WithArgs("svc-1", "est-1", "Escova", 95.0, 60, true).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT dona_atua_como_profissional`)).
			WithArgs("est-1").
			WillReturnRows(sqlmock.NewRows([]string{"dona_atua_como_profissional"}).AddRow(false))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, ativo, eh_dona`)).
			WithArgs("est-1", pq.Array([]string{"dona-1"})).
			WillReturnRows(sqlmock.NewRows([]string{"id", "ativo", "eh_dona"}).
				AddRow("dona-1", true, true))
		mock.ExpectRollback()

		err = svc.UpdateService(context.Background(), "est-1", "svc-1", "Escova", 95, 60, true, []string{"dona-1"})
		if !errors.Is(err, ErrDonaNotActingAsProfessional) {
			t.Fatalf("err=%v; want ErrDonaNotActingAsProfessional", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("serviço inexistente", func(t *testing.T) {
		t.Parallel()
		raw, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New: %v", err)
		}
		defer raw.Close()
		svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM servicos`)).
			WithArgs("svc-missing", "est-1").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		err = svc.UpdateService(context.Background(), "est-1", "svc-missing", "X", 10, 30, true, nil)
		if !errors.Is(err, ErrServicoNaoEncontrado) {
			t.Fatalf("err=%v; want ErrServicoNaoEncontrado", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestListProfessionalsIncluiEhDona(t *testing.T) {
	t.Parallel()

	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer raw.Close()
	svc := NewProfissionalService(sqlx.NewDb(raw, "sqlmock"))

	rows := sqlmock.NewRows([]string{
		"id", "nome", "especialidade_id", "especialidade_nome",
		"comissao_porcentagem", "ativo", "pendente_aprovacao", "eh_dona",
		"foto_url", "data_nascimento",
	}).
		AddRow("dona-1", "Dona", "esp-1", "Geral", 0.0, true, false, true, nil, nil).
		AddRow("prof-1", "Ana", "esp-1", "Manicure", 40.0, true, false, false, nil, nil)

	mock.ExpectQuery(`SELECT p.id, p.nome`).
		WithArgs("est-1").
		WillReturnRows(rows)

	lista, err := svc.ListProfessionals(context.Background(), "est-1")
	if err != nil {
		t.Fatalf("ListProfessionals: %v", err)
	}
	if len(lista) != 2 {
		t.Fatalf("len=%d; want 2", len(lista))
	}
	if !lista[0].EhDona {
		t.Fatalf("dona deve ter eh_dona=true")
	}
	if lista[1].EhDona {
		t.Fatalf("Ana não deve ter eh_dona")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadTenantViewIncluiDonaAtuaComoProfissional(t *testing.T) {
	t.Parallel()

	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer raw.Close()
	svc := NewBootstrapService(sqlx.NewDb(raw, "sqlmock"))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM estabelecimentos e`)).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "nome_comercial", "slug", "logo_url", "dona_atua_como_profissional",
			"plano_id", "data_vencimento", "assinatura_status",
		}).AddRow("tenant-1", "Glow", "glow", nil, true, nil, nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(whatsapp_enabled, FALSE)`)).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(false))

	view, err := svc.loadTenantView(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("loadTenantView: %v", err)
	}
	if !view.DonaAtuaComoProfissional {
		t.Fatalf("dona_atua_como_profissional deve ser true")
	}
	rawJSON, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`"dona_atua_como_profissional":\s*true`).Match(rawJSON) {
		t.Fatalf("JSON sem flag: %s", rawJSON)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
