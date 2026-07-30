package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestRegisterEstablishmentCreatesNotificationConfigAtomically(t *testing.T) {
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE slug").
		WithArgs("studio-glow").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery("INSERT INTO estabelecimentos").
		WithArgs("Studio Glow", "studio-glow").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tenant-new"))
	mock.ExpectExec("INSERT INTO configuracoes_notificacoes_agenda").
		WithArgs("tenant-new").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	service := NewEstabelecimentoService(sqlx.NewDb(raw, "sqlmock"))
	id, slug, err := service.RegisterEstablishment(context.Background(), "Studio Glow", "")
	if err != nil {
		t.Fatalf("RegisterEstablishment: %v", err)
	}
	if id != "tenant-new" || slug != "studio-glow" {
		t.Fatalf("cadastro inesperado: id=%q slug=%q", id, slug)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterEstablishmentRollsBackWhenNotificationConfigFails(t *testing.T) {
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE slug").
		WithArgs("studio-glow").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery("INSERT INTO estabelecimentos").
		WithArgs("Studio Glow", "studio-glow").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tenant-new"))
	mock.ExpectExec("INSERT INTO configuracoes_notificacoes_agenda").
		WithArgs("tenant-new").
		WillReturnError(context.DeadlineExceeded)
	mock.ExpectRollback()

	service := NewEstabelecimentoService(sqlx.NewDb(raw, "sqlmock"))
	if _, _, err := service.RegisterEstablishment(context.Background(), "Studio Glow", ""); err == nil {
		t.Fatal("esperava falha ao criar configuração padrão")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
