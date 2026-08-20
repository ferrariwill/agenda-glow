package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestValidarDataNascimento(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.FixedZone("America/Sao_Paulo", -3*60*60)
	}
	agora := time.Now().In(loc)
	hoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, loc)
	amanha := hoje.AddDate(0, 0, 1)
	ontem := hoje.AddDate(0, 0, -1)

	casos := []struct {
		nome    string
		raw     string
		wantErr error
	}{
		{nome: "passado", raw: "1992-03-15", wantErr: nil},
		{nome: "hoje", raw: hoje.Format("2006-01-02"), wantErr: nil},
		{nome: "ontem", raw: ontem.Format("2006-01-02"), wantErr: nil},
		{nome: "futuro", raw: amanha.Format("2006-01-02"), wantErr: ErrDataNascimentoFutura},
		{nome: "formato inválido", raw: "15/03/1992", wantErr: ErrDataNascimentoInvalida},
		{nome: "vazio", raw: "   ", wantErr: ErrDataNascimentoInvalida},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nome, func(t *testing.T) {
			t.Parallel()
			err := ValidarDataNascimento(caso.raw)
			if !errors.Is(err, caso.wantErr) {
				t.Fatalf("ValidarDataNascimento(%q) = %v; want %v", caso.raw, err, caso.wantErr)
			}
		})
	}
}

func TestCreateProfessionalRejeitaDataFutura(t *testing.T) {
	t.Parallel()

	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer raw.Close()

	svc := NewProfissionalService(sqlx.NewDb(raw, "sqlmock"))
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	futuro := time.Now().In(loc).AddDate(0, 0, 2).Format("2006-01-02")

	_, err = svc.CreateProfessional(
		context.Background(),
		"est-1", "Ana", "esp-1", 40,
		&futuro, nil,
	)
	if !errors.Is(err, ErrDataNascimentoFutura) {
		t.Fatalf("CreateProfessional futuro = %v; want ErrDataNascimentoFutura", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("não deveria tocar o banco: %v", err)
	}
}

func TestSetFotoURLAtualizaEIsolaTenant(t *testing.T) {
	t.Parallel()

	t.Run("sucesso", func(t *testing.T) {
		t.Parallel()
		raw, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New: %v", err)
		}
		defer raw.Close()
		svc := NewProfissionalService(sqlx.NewDb(raw, "sqlmock"))

		mock.ExpectExec(`UPDATE profissionais`).
			WithArgs("prof-1", "est-1", "https://cdn.example/avatars/a.png").
			WillReturnResult(sqlmock.NewResult(0, 1))

		if err := svc.SetFotoURL(context.Background(), "est-1", "prof-1", "https://cdn.example/avatars/a.png"); err != nil {
			t.Fatalf("SetFotoURL: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("cross-tenant 404", func(t *testing.T) {
		t.Parallel()
		raw, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New: %v", err)
		}
		defer raw.Close()
		svc := NewProfissionalService(sqlx.NewDb(raw, "sqlmock"))

		mock.ExpectExec(`UPDATE profissionais`).
			WithArgs("prof-outro", "est-1", "https://cdn.example/x.png").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err = svc.SetFotoURL(context.Background(), "est-1", "prof-outro", "https://cdn.example/x.png")
		if !errors.Is(err, ErrProfissionalNaoEncontrado) {
			t.Fatalf("SetFotoURL cross-tenant = %v; want ErrProfissionalNaoEncontrado", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestListProfessionalsIncluiFotoEData(t *testing.T) {
	t.Parallel()

	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer raw.Close()
	svc := NewProfissionalService(sqlx.NewDb(raw, "sqlmock"))

	foto := "https://cdn.example/a.png"
	data := "1992-03-15"
	rows := sqlmock.NewRows([]string{
		"id", "nome", "especialidade_id", "especialidade_nome",
		"comissao_porcentagem", "ativo", "pendente_aprovacao",
		"foto_url", "data_nascimento",
	}).AddRow("prof-1", "Ana", "esp-1", "Manicure", 40.0, true, false, foto, data)

	mock.ExpectQuery(`SELECT p.id, p.nome`).
		WithArgs("est-1").
		WillReturnRows(rows)

	lista, err := svc.ListProfessionals(context.Background(), "est-1")
	if err != nil {
		t.Fatalf("ListProfessionals: %v", err)
	}
	if len(lista) != 1 {
		t.Fatalf("len=%d; want 1", len(lista))
	}
	if lista[0].FotoURL == nil || *lista[0].FotoURL != foto {
		t.Fatalf("foto_url=%v; want %s", lista[0].FotoURL, foto)
	}
	if lista[0].DataNascimento == nil || *lista[0].DataNascimento != data {
		t.Fatalf("data_nascimento=%v; want %s", lista[0].DataNascimento, data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
