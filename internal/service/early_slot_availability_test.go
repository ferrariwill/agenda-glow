package service

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestTenantBootstrapPublishesCanonicalEarlySlotSignal(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		status  string
		active  bool
	}{
		{"connected", true, WhatsAppStatusConectado, true},
		{"disabled", false, WhatsAppStatusConectado, false},
		{"disconnected", true, WhatsAppStatusDesconectado, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rawDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer rawDB.Close()
			svc := NewBootstrapService(sqlx.NewDb(rawDB, "sqlmock"))

			mock.ExpectQuery(regexp.QuoteMeta(`FROM estabelecimentos e`)).
				WithArgs("tenant-1").
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "nome_comercial", "slug", "logo_url", "dona_atua_como_profissional",
					"plano_id", "data_vencimento", "assinatura_status",
				}).AddRow("tenant-1", "Glow", "glow", nil, false, nil, nil, nil))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(whatsapp_enabled, FALSE)`)).
				WithArgs("tenant-1").
				WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(tc.enabled))
			if tc.enabled {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(whatsapp_status, 'DESCONECTADO')`)).
					WithArgs("tenant-1").
					WillReturnRows(sqlmock.NewRows([]string{"whatsapp_status"}).AddRow(tc.status))
			}

			view, err := svc.loadTenantView(context.Background(), "tenant-1")
			if err != nil {
				t.Fatal(err)
			}
			if view.EarlySlotQueueActive != tc.active {
				t.Fatalf("active = %v, esperado %v", view.EarlySlotQueueActive, tc.active)
			}
			if tc.active && view.EarlySlotQueueInactiveReason != nil {
				t.Fatalf("motivo deve ser null quando ativo: %v", *view.EarlySlotQueueInactiveReason)
			}
			if !tc.active && (view.EarlySlotQueueInactiveReason == nil ||
				*view.EarlySlotQueueInactiveReason != earlySlotMotivoCanalIndisponivel) {
				t.Fatalf("motivo inativo inválido: %#v", view.EarlySlotQueueInactiveReason)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTenantBootstrapGateFailureIsFailClosedWithoutPayloadError(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer rawDB.Close()
	svc := NewBootstrapService(sqlx.NewDb(rawDB, "sqlmock"))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM estabelecimentos e`)).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "nome_comercial", "slug", "logo_url", "dona_atua_como_profissional",
			"plano_id", "data_vencimento", "assinatura_status",
		}).AddRow("tenant-1", "Glow", "glow", nil, false, nil, nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(whatsapp_enabled, FALSE)`)).
		WithArgs("tenant-1").
		WillReturnError(context.DeadlineExceeded)

	view, err := svc.loadTenantView(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("falha do gate não deve falhar o bootstrap: %v", err)
	}
	if view.EarlySlotQueueActive || view.EarlySlotQueueInactiveReason == nil {
		t.Fatalf("gate com erro deve publicar inativo: %#v", view)
	}
}

func TestPublicCatalogPublishesBooleanWithoutInfrastructureReason(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer rawDB.Close()
	svc := NewEstabelecimentoService(sqlx.NewDb(rawDB, "sqlmock"))

	mock.ExpectQuery(regexp.QuoteMeta(`WHERE slug = $1 AND ativo = TRUE`)).
		WithArgs("glow").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "nome_comercial", "slug", "logo_url",
		}).AddRow("tenant-1", "Glow", "glow", nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(whatsapp_enabled, FALSE)`)).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(whatsapp_status, 'DESCONECTADO')`)).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_status"}).AddRow(WhatsAppStatusConectado))

	est, err := svc.BuscarPorSlug(context.Background(), "glow")
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(est)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["early_slot_notifications_available"] != true {
		t.Fatalf("booleano público ausente/inválido: %s", body)
	}
	if _, leaked := payload["early_slot_queue_inactive_reason"]; leaked {
		t.Fatalf("catálogo público não pode expor motivo de infraestrutura: %s", body)
	}
}

func TestPreferencePersistsWhenChannelUnavailable(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer rawDB.Close()
	svc := NewEarlySlotService(sqlx.NewDb(rawDB, "sqlmock"), "")
	optedAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE agendamentos SET aceita_adiantar=$3`)).
		WithArgs("agendamento-1", "tenant-1", true).
		WillReturnRows(sqlmock.NewRows([]string{"aceita_adiantar_em"}).AddRow(optedAt))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(whatsapp_enabled, FALSE)`)).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(false))

	at, err := svc.SetPreference(context.Background(), "tenant-1", "agendamento-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if !at.Equal(optedAt) {
		t.Fatalf("timestamp persistido = %v, esperado %v", at, optedAt)
	}
	if svc.NotificationsAvailable(context.Background(), "tenant-1") {
		t.Fatal("opt-in persistido não pode transformar canal indisponível em disponível")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationsAvailableFailsClosed(t *testing.T) {
	svc := &EarlySlotService{
		channelReady: func(context.Context, sqlx.QueryerContext, string) (bool, error) {
			return false, errors.New("timeout")
		},
	}
	if svc.NotificationsAvailable(context.Background(), "tenant-1") {
		t.Fatal("erro do gate deve resultar em false")
	}
}
