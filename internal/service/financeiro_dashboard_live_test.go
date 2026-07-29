package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestGetDashboardGerencialLiveDB(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://postgres:glow_secure_pwd_2026@localhost:5435/agenda_glow_prod?sslmode=disable"
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("db unavailable: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	fin := NewFinanceiroService(db)
	start, end, _ := PeriodoMesAtual()

	estID := "a1000002-0002-4002-8002-000000000002"
	_, err = fin.GetDashboardGerencial(ctx, estID, "Estúdio Glow", start, end)
	if err != nil {
		t.Fatalf("GetDashboardGerencial: %v", err)
	}
}

func TestGetEstablishmentFinancialReportLiveDB(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://postgres:glow_secure_pwd_2026@localhost:5435/agenda_glow_prod?sslmode=disable"
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("db unavailable: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	fin := NewFinanceiroService(db)
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.Local)

	estID := "a1000002-0002-4002-8002-000000000002"
	_, err = fin.GetEstablishmentFinancialReport(ctx, estID, start, end)
	if err != nil {
		t.Fatalf("GetEstablishmentFinancialReport: %v", err)
	}
}
