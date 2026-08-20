package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func TestListServicesIncluiProfissionalIDsEmBatch(t *testing.T) {
	t.Parallel()

	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer raw.Close()
	svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

	mock.ExpectQuery(`SELECT s.id, s.nome, s.preco_base`).
		WithArgs("est-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "nome", "preco_base", "duracao_base_minutos", "ativo", "categoria_id", "categoria_nome",
		}).
			AddRow("svc-1", "Corte", 80.0, 60, true, nil, nil).
			AddRow("svc-2", "Escova", 50.0, 30, true, nil, nil))

	mock.ExpectQuery(`SELECT sa.id, sa.servico_id`).
		WithArgs("est-1", pq.Array([]string{"svc-1", "svc-2"})).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "servico_id", "nome", "preco_adicional", "duracao_adicional_minutos",
		}))

	mock.ExpectQuery(`SELECT servico_id, profissional_id`).
		WithArgs("est-1", pq.Array([]string{"svc-1", "svc-2"})).
		WillReturnRows(sqlmock.NewRows([]string{"servico_id", "profissional_id"}).
			AddRow("svc-1", "prof-2").
			AddRow("svc-1", "prof-1"))

	lista, err := svc.ListServices(context.Background(), "est-1")
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if len(lista) != 2 {
		t.Fatalf("len=%d; want 2", len(lista))
	}
	if got := lista[0].ProfissionalIDs; len(got) != 2 || got[0] != "prof-2" || got[1] != "prof-1" {
		t.Fatalf("svc-1 profissional_ids=%v", got)
	}
	if lista[1].ProfissionalIDs == nil || len(lista[1].ProfissionalIDs) != 0 {
		t.Fatalf("svc-2 profissional_ids deve ser []; got %#v", lista[1].ProfissionalIDs)
	}

	payload, err := json.Marshal(lista[1])
	if err != nil {
		t.Fatal(err)
	}
	if !jsonContainsKey(payload, "profissional_ids") {
		t.Fatalf("profissional_ids deve sempre serializar: %s", payload)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateServicePersisteVinculosEmTransacao(t *testing.T) {
	t.Parallel()

	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer raw.Close()
	svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM profissionais`).
		WithArgs("est-1", pq.Array([]string{"prof-1", "prof-dona"})).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("prof-1").AddRow("prof-dona"))
	mock.ExpectQuery(`INSERT INTO servicos`).
		WithArgs("est-1", "Corte", 80.0, 60, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("svc-1"))
	mock.ExpectExec(`DELETE FROM servico_profissionais`).
		WithArgs("est-1", "svc-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO servico_profissionais`).
		WithArgs("est-1", "svc-1", "prof-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO servico_profissionais`).
		WithArgs("est-1", "svc-1", "prof-dona").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	id, err := svc.CreateService(
		context.Background(), "est-1", "Corte", 80, 60,
		[]string{"prof-1", "prof-dona"}, "",
	)
	if err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	if id != "svc-1" {
		t.Fatalf("id=%q; want svc-1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateServiceRejeitaProfissionalOutroTenant(t *testing.T) {
	t.Parallel()

	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer raw.Close()
	svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM profissionais`).
		WithArgs("est-1", pq.Array([]string{"prof-outro"})).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()

	_, err = svc.CreateService(
		context.Background(), "est-1", "Corte", 80, 60,
		[]string{"prof-outro"}, "",
	)
	if !errors.Is(err, ErrProfissionalVinculoInvalido) {
		t.Fatalf("err=%v; want ErrProfissionalVinculoInvalido", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateServiceReplaceAtomicoEIsolaTenant(t *testing.T) {
	t.Parallel()

	t.Run("replace vínculos", func(t *testing.T) {
		t.Parallel()
		raw, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New: %v", err)
		}
		defer raw.Close()
		svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

		ids := []string{"prof-1"}
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id FROM servicos`).
			WithArgs("svc-1", "est-1").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("svc-1"))
		mock.ExpectQuery(`SELECT id FROM profissionais`).
			WithArgs("est-1", pq.Array(ids)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("prof-1"))
		mock.ExpectExec(`UPDATE servicos`).
			WithArgs("svc-1", "est-1", "Corte", 90.0, 45, true).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM servico_profissionais`).
			WithArgs("est-1", "svc-1").
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`INSERT INTO servico_profissionais`).
			WithArgs("est-1", "svc-1", "prof-1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		// BuscarServicoPorID pós-commit
		mock.ExpectQuery(`SELECT s.id, s.nome, s.preco_base`).
			WithArgs("svc-1", "est-1").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "nome", "preco_base", "duracao_base_minutos", "ativo", "categoria_id", "categoria_nome",
			}).AddRow("svc-1", "Corte", 90.0, 45, true, nil, nil))
		mock.ExpectQuery(`SELECT sa.id, sa.servico_id`).
			WithArgs("est-1", "svc-1").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "servico_id", "nome", "preco_adicional", "duracao_adicional_minutos",
			}))
		mock.ExpectQuery(`SELECT profissional_id`).
			WithArgs("est-1", "svc-1").
			WillReturnRows(sqlmock.NewRows([]string{"profissional_id"}).AddRow("prof-1"))

		serv, err := svc.UpdateService(context.Background(), "est-1", "svc-1", UpdateServiceInput{
			Nome:            "Corte",
			PrecoBase:       90,
			DuracaoBase:     45,
			Ativo:           true,
			ProfissionalIDs: &ids,
		})
		if err != nil {
			t.Fatalf("UpdateService: %v", err)
		}
		if len(serv.ProfissionalIDs) != 1 || serv.ProfissionalIDs[0] != "prof-1" {
			t.Fatalf("profissional_ids=%v", serv.ProfissionalIDs)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("serviço outro tenant → not found", func(t *testing.T) {
		t.Parallel()
		raw, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New: %v", err)
		}
		defer raw.Close()
		svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id FROM servicos`).
			WithArgs("svc-outro", "est-1").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		_, err = svc.UpdateService(context.Background(), "est-1", "svc-outro", UpdateServiceInput{
			Nome:        "X",
			PrecoBase:   10,
			DuracaoBase: 30,
			Ativo:       true,
		})
		if !errors.Is(err, ErrServicoNaoEncontrado) {
			t.Fatalf("err=%v; want ErrServicoNaoEncontrado", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("omite profissional_ids mantém vínculos", func(t *testing.T) {
		t.Parallel()
		raw, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New: %v", err)
		}
		defer raw.Close()
		svc := NewProcedimentoService(sqlx.NewDb(raw, "sqlmock"))

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id FROM servicos`).
			WithArgs("svc-1", "est-1").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("svc-1"))
		mock.ExpectExec(`UPDATE servicos`).
			WithArgs("svc-1", "est-1", "Corte", 90.0, 45, false).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		mock.ExpectQuery(`SELECT s.id, s.nome, s.preco_base`).
			WithArgs("svc-1", "est-1").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "nome", "preco_base", "duracao_base_minutos", "ativo", "categoria_id", "categoria_nome",
			}).AddRow("svc-1", "Corte", 90.0, 45, false, nil, nil))
		mock.ExpectQuery(`SELECT sa.id, sa.servico_id`).
			WithArgs("est-1", "svc-1").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "servico_id", "nome", "preco_adicional", "duracao_adicional_minutos",
			}))
		mock.ExpectQuery(`SELECT profissional_id`).
			WithArgs("est-1", "svc-1").
			WillReturnRows(sqlmock.NewRows([]string{"profissional_id"}).AddRow("prof-keep"))

		serv, err := svc.UpdateService(context.Background(), "est-1", "svc-1", UpdateServiceInput{
			Nome:        "Corte",
			PrecoBase:   90,
			DuracaoBase: 45,
			Ativo:       false,
			// ProfissionalIDs nil → não toca servico_profissionais
		})
		if err != nil {
			t.Fatalf("UpdateService: %v", err)
		}
		if len(serv.ProfissionalIDs) != 1 || serv.ProfissionalIDs[0] != "prof-keep" {
			t.Fatalf("profissional_ids=%v; want [prof-keep]", serv.ProfissionalIDs)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func jsonContainsKey(payload []byte, key string) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
