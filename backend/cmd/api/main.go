package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	adminhandler "github.com/agendaglow/agendaglow/backend/internal/handler"
	publichandler "github.com/agendaglow/agendaglow/internal/handler"
	"github.com/agendaglow/agendaglow/internal/config"
	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	if err := config.ValidateRuntime(); err != nil {
		log.Fatalf("configuração de portas inválida: %v", err)
	}

	databaseURL, err := config.DatabaseURL()
	if err != nil {
		log.Fatalf("DATABASE_URL: %v", err)
	}

	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		log.Fatalf("conectar ao banco: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	estabelecimentoSvc := service.NewEstabelecimentoService(db)
	planoSaasSvc := service.NewPlanoSaasService(db)
	profissionalSvc := service.NewProfissionalService(db)
	procedimentoSvc := service.NewProcedimentoService(db)
	agendaSvc := service.NewAgendaService(db, service.AgendaOptions{
		BaseURL: envOrDefault("APP_BASE_URL", "http://localhost:8081"),
		Mailer:  service.NewSMTPMailerFromEnv(),
	})
	financeiroSvc := service.NewFinanceiroService(db)
	authSvc := service.NewAuthService(db)
	saasGuard := security.NewSaaSGuard(db, 60*time.Second)

	bookingHandler := publichandler.NewBookingPageHandler(estabelecimentoSvc)
	publicSlotsHandler := publichandler.NewPublicSlotsHandler(agendaSvc, estabelecimentoSvc)
	publicAppointmentsHandler := publichandler.NewPublicAppointmentsHandler(agendaSvc)
	whatsAppWebhookHandler := publichandler.NewWhatsAppWebhookHandler(agendaSvc)
	configHandler := adminhandler.NewEstabelecimentoConfigHandler(estabelecimentoSvc)
	adminEstHandler := adminhandler.NewAdminEstablishmentsHandler(estabelecimentoSvc)
	adminPlansHandler := adminhandler.NewAdminPlansHandler(planoSaasSvc, saasGuard)
	tenantCatalogHandler := adminhandler.NewTenantCatalogHandler(profissionalSvc, procedimentoSvc)
	tenantFinanceHandler := adminhandler.NewTenantFinanceHandler(financeiroSvc)
	authHandler := adminhandler.NewAuthHandler(authSvc)

	dashboardHandler, err := adminhandler.NewDashboardDonaHandler(financeiroSvc, estabelecimentoSvc)
	if err != nil {
		log.Fatalf("carregar templates do painel: %v", err)
	}

	dashboardProfHandler, err := adminhandler.NewDashboardProfissionalHandler(agendaSvc, financeiroSvc)
	if err != nil {
		log.Fatalf("carregar templates da profissional: %v", err)
	}

	adminConfigHandler, err := adminhandler.NewAdminConfigHandler(procedimentoSvc, profissionalSvc, financeiroSvc, estabelecimentoSvc)
	if err != nil {
		log.Fatalf("carregar templates admin: %v", err)
	}

	superAdminUIHandler, err := adminhandler.NewSuperAdminUIHandler(estabelecimentoSvc, planoSaasSvc, saasGuard)
	if err != nil {
		log.Fatalf("carregar templates super admin: %v", err)
	}

	saasValidation := security.SaaSValidationMiddleware(saasGuard)

	// Dona do salão: role DONA + assinatura SaaS ativa
	donaRoute := func(h http.HandlerFunc) http.Handler {
		return chainHandlers(
			security.RequireDona,
			http.HandlerFunc(saasValidation(h)),
		)
	}

	// Profissional parceira: role PROFISSIONAL (escopo profissional_id no token)
	professionalRoute := func(h http.HandlerFunc) http.Handler {
		return security.RequireProfissional(http.HandlerFunc(h))
	}

	// Super Admin: role SUPER_ADMIN + e-mail autorizado
	superAdminRoute := func(h http.HandlerFunc) http.Handler {
		return security.RequireSuperAdmin(http.HandlerFunc(h))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// Autenticação unificada (público)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /login/superadmin", authHandler.LoginForm)
	mux.HandleFunc("POST /login/dona", authHandler.LoginForm)
	mux.HandleFunc("POST /login/profissional", authHandler.LoginForm)

	// Super Admin — API JSON + UI HTML
	mux.Handle("GET /api/v1/admin/establishments", superAdminRoute(adminEstHandler.List))
	mux.Handle("POST /api/v1/admin/establishments", superAdminRoute(adminEstHandler.Create))
	mux.Handle("PUT /api/v1/admin/establishments/{id}/status", superAdminRoute(adminEstHandler.ToggleStatus))
	mux.Handle("POST /api/v1/admin/establishments/{id}/assign-plan", superAdminRoute(adminPlansHandler.AssignPlan))
	mux.Handle("GET /api/v1/admin/plans", superAdminRoute(adminPlansHandler.List))
	mux.Handle("POST /api/v1/admin/plans", superAdminRoute(adminPlansHandler.Create))

	mux.Handle("GET /superadmin/dashboard", superAdminRoute(superAdminUIHandler.Dashboard))
	mux.Handle("POST /superadmin/establishments", superAdminRoute(superAdminUIHandler.CreateEstablishment))
	mux.Handle("POST /superadmin/establishments/{id}/suspend", superAdminRoute(superAdminUIHandler.SuspendEstablishment))
	mux.Handle("POST /superadmin/establishments/{id}/activate", superAdminRoute(superAdminUIHandler.ActivateEstablishment))
	mux.Handle("POST /superadmin/establishments/{id}/renew", superAdminRoute(superAdminUIHandler.RenewEstablishment))
	mux.Handle("GET /superadmin/planos", superAdminRoute(superAdminUIHandler.Planos))
	mux.Handle("POST /superadmin/planos", superAdminRoute(superAdminUIHandler.CreatePlan))

	// Dona do salão — finanças, configuração e painel gerencial
	mux.Handle("GET /api/v1/services", donaRoute(tenantCatalogHandler.ListServices))
	mux.Handle("POST /api/v1/services", donaRoute(tenantCatalogHandler.CreateService))
	mux.Handle("POST /api/v1/services/{id}/additionals", donaRoute(tenantCatalogHandler.CreateServiceAdditional))
	mux.Handle("GET /api/v1/professionals", donaRoute(tenantCatalogHandler.ListProfessionals))
	mux.Handle("POST /api/v1/professionals", donaRoute(tenantCatalogHandler.CreateProfessional))
	mux.Handle("GET /api/v1/finance/report", donaRoute(tenantFinanceHandler.GetReport))
	mux.Handle("GET /api/v1/finance/professionals/{id}/pending", donaRoute(tenantFinanceHandler.GetProfessionalPending))
	mux.Handle("POST /api/v1/finance/professionals/{id}/pay", donaRoute(tenantFinanceHandler.PayProfessionalCommissions))
	mux.Handle("POST /v1/estabelecimentos/config", donaRoute(configHandler.ServeHTTP))
	mux.Handle("GET /admin/servicos", donaRoute(adminConfigHandler.Servicos))
	mux.Handle("POST /admin/servicos", donaRoute(adminConfigHandler.CreateServico))
	mux.Handle("POST /admin/servicos/{id}/adicionais", donaRoute(adminConfigHandler.CreateAdicional))
	mux.Handle("GET /admin/equipe", donaRoute(adminConfigHandler.Equipe))
	mux.Handle("POST /admin/equipe", donaRoute(adminConfigHandler.CreateProfissional))
	mux.Handle("GET /admin/caixa", donaRoute(adminConfigHandler.Caixa))
	mux.Handle("POST /admin/caixa/lancamento", donaRoute(adminConfigHandler.CreateLancamento))
	mux.Handle("GET /dashboard/gerencial", donaRoute(dashboardHandler.ServeHTTP))
	mux.Handle("POST /dashboard/gerencial/professionals/{id}/pay", donaRoute(dashboardHandler.PayProfessional))
	mux.Handle("POST /dashboard/gerencial/lancamento", donaRoute(dashboardHandler.RegisterExpense))

	// Profissional parceira — agenda individual isolada por profissional_id do token
	mux.Handle("GET /dashboard/profissional", professionalRoute(dashboardProfHandler.ServeHTTP))
	mux.Handle("GET /dashboard/profissional/timeline", professionalRoute(dashboardProfHandler.Timeline))
	mux.Handle("POST /dashboard/profissional/appointments/{id}/complete", professionalRoute(dashboardProfHandler.CompleteAppointment))

	// API pública — horários livres, ações de agendamento e webhook WhatsApp Gateway
	mux.Handle("GET /api/v1/public/{slug}/slots", publicSlotsHandler)
	mux.HandleFunc("POST /api/v1/public/appointments/{id}/approve", publicAppointmentsHandler.Approve)
	mux.HandleFunc("POST /api/v1/public/appointments/{id}/reschedule", publicAppointmentsHandler.Reschedule)
	mux.HandleFunc("POST /api/v1/webhook/whatsapp-callback", whatsAppWebhookHandler.Callback)

	// Página pública de agendamento (sem autenticação — cliente final)
	mux.Handle("GET /{slug}", bookingHandler)

	addr := config.ListenAddr()

	log.Printf("AgendaGlow API ouvindo em %s (porta reservada %s — Gateway usa %s)",
		addr, config.DefaultAPIPort, config.GatewayAPIPort)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("servidor encerrado: %v", err)
	}
}

func chainHandlers(first func(http.Handler) http.Handler, next http.Handler) http.Handler {
	return first(next)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}
