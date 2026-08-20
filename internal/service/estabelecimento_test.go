package service

import (
	"context"
	"database/sql"
	"errors"
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

func TestAtualizarConfigRequiresAtivo(t *testing.T) {
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE id = \\$1 AND ativo = TRUE FOR UPDATE").
		WithArgs("tenant-1").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	svc := NewEstabelecimentoService(sqlx.NewDb(raw, "sqlmock"))
	_, err = svc.AtualizarConfig(context.Background(), ConfigEstabelecimentoInput{
		EstabelecimentoID: "tenant-1",
		NomeComercial:     "Studio Glow",
		Slug:              "studio-glow",
	})
	if !errors.Is(err, ErrEstabelecimentoNaoEncontrado) {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAtualizarIdentidadeAdminUpdatesInactive(t *testing.T) {
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE id = \\$1 FOR UPDATE").
		WithArgs("tenant-inactive").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tenant-inactive"))
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE slug = \\$1 AND id <> \\$2").
		WithArgs("studio-glow", "tenant-inactive").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("UPDATE estabelecimentos").
		WithArgs("tenant-inactive", "Studio Glow", "studio-glow").
		WillReturnRows(sqlmock.NewRows([]string{"id", "nome_comercial", "slug", "logo_url"}).
			AddRow("tenant-inactive", "Studio Glow", "studio-glow", nil))
	mock.ExpectCommit()

	svc := NewEstabelecimentoService(sqlx.NewDb(raw, "sqlmock"))
	got, err := svc.AtualizarIdentidadeAdmin(context.Background(), ConfigEstabelecimentoInput{
		EstabelecimentoID: "tenant-inactive",
		NomeComercial:     "Studio Glow",
		Slug:              "studio-glow",
	})
	if err != nil {
		t.Fatalf("AtualizarIdentidadeAdmin: %v", err)
	}
	if got.ID != "tenant-inactive" || got.Slug != "studio-glow" {
		t.Fatalf("resultado inesperado: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAtualizarIdentidadeAdminClearsEmptyLogo(t *testing.T) {
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	empty := ""
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE id = \\$1 FOR UPDATE").
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tenant-1"))
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE slug = \\$1 AND id <> \\$2").
		WithArgs("studio-glow", "tenant-1").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("UPDATE estabelecimentos").
		WithArgs("tenant-1", "Studio Glow", "studio-glow").
		WillReturnRows(sqlmock.NewRows([]string{"id", "nome_comercial", "slug", "logo_url"}).
			AddRow("tenant-1", "Studio Glow", "studio-glow", nil))
	mock.ExpectCommit()

	svc := NewEstabelecimentoService(sqlx.NewDb(raw, "sqlmock"))
	got, err := svc.AtualizarIdentidadeAdmin(context.Background(), ConfigEstabelecimentoInput{
		EstabelecimentoID: "tenant-1",
		NomeComercial:     "Studio Glow",
		Slug:              "studio-glow",
		LogoURL:           &empty,
	})
	if err != nil {
		t.Fatalf("AtualizarIdentidadeAdmin: %v", err)
	}
	if got.LogoURL != nil {
		t.Fatalf("esperava logo limpo, got %#v", got.LogoURL)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAtualizarIdentidadeAdminSlugEmUso(t *testing.T) {
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE id = \\$1 FOR UPDATE").
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tenant-1"))
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE slug = \\$1 AND id <> \\$2").
		WithArgs("outro-slug", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tenant-2"))
	mock.ExpectRollback()

	svc := NewEstabelecimentoService(sqlx.NewDb(raw, "sqlmock"))
	_, err = svc.AtualizarIdentidadeAdmin(context.Background(), ConfigEstabelecimentoInput{
		EstabelecimentoID: "tenant-1",
		NomeComercial:     "Studio Glow",
		Slug:              "outro-slug",
	})
	if !errors.Is(err, ErrSlugEmUso) {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAtualizarIdentidadeAdminInvalidSlug(t *testing.T) {
	svc := NewEstabelecimentoService(nil)
	_, err := svc.AtualizarIdentidadeAdmin(context.Background(), ConfigEstabelecimentoInput{
		EstabelecimentoID: "tenant-1",
		NomeComercial:     "Studio Glow",
		Slug:              "INVALID SLUG",
	})
	if !errors.Is(err, ErrSlugInvalido) {
		t.Fatalf("err = %v", err)
	}
}
