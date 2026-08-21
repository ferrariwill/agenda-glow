package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	adminhandler "github.com/agendaglow/agendaglow/backend/internal/handler"
	"github.com/agendaglow/agendaglow/internal/config"
	publichandler "github.com/agendaglow/agendaglow/internal/handler"
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
	especialidadeSvc := service.NewEspecialidadeService(db)
	procedimentoSvc := service.NewProcedimentoService(db)
	categoriaServicoSvc := service.NewCategoriaServicoService(db)
	agendaSvc := service.NewAgendaService(db, service.AgendaOptions{
		BaseURL: envOrDefault("APP_BASE_URL", "http://localhost:8081"),
		Mailer:  service.NewSMTPMailerFromEnv(),
	})
	financeiroSvc := service.NewFinanceiroService(db)
	authSvc := service.NewAuthService(db)
	bootstrapSvc := service.NewBootstrapService(db)
	saasGuard := security.NewSaaSGuard(db, 60*time.Second)
	filaSvc := service.NewFilaEsperaService(db)
	insumoSvc := service.NewInsumoService(db)
	servicoInsumoSvc := service.NewServicoInsumoService(db)
	estoquePrevisaoSvc := service.NewEstoquePrevisaoService(db)
	whatsAppGate := security.NewWhatsAppGate(db, 30*time.Second)
	earlySlotSvc := service.NewEarlySlotService(db, envOrDefault("APP_BASE_URL", "http://localhost:8081"))
	agendaSvc.SetEarlySlotService(earlySlotSvc)

	bookingHandler := publichandler.NewBookingPageHandler(estabelecimentoSvc)
	publicSlotsHandler := publichandler.NewPublicSlotsHandler(agendaSvc, estabelecimentoSvc)
	publicAppointmentsHandler := publichandler.NewPublicAppointmentsHandler(agendaSvc)
	whatsAppWebhookHandler := publichandler.NewWhatsAppWebhookHandler(agendaSvc, estabelecimentoSvc)
	whatsAppIntegrationHandler := adminhandler.NewWhatsAppIntegrationHandler(estabelecimentoSvc)
	earlySlotHandler := adminhandler.NewEarlySlotHandler(earlySlotSvc)
	configHandler := adminhandler.NewEstabelecimentoConfigHandler(estabelecimentoSvc)
	agendaNotificationsHandler := adminhandler.NewAgendaNotificationsHandler(agendaSvc)
	avisoSvc := service.NewAvisoService(db)
	adminEstHandler := adminhandler.NewAdminEstablishmentsHandler(estabelecimentoSvc, authSvc, whatsAppGate)
	adminPlansHandler := adminhandler.NewAdminPlansHandler(planoSaasSvc, saasGuard)
	adminAvisosHandler := adminhandler.NewAdminAvisosHandler(avisoSvc)
	meNotificacoesHandler := adminhandler.NewMeNotificacoesHandler(avisoSvc)
	tenantCatalogHandler := adminhandler.NewTenantCatalogHandler(profissionalSvc, procedimentoSvc, categoriaServicoSvc)
	estoqueHandler := adminhandler.NewEstoqueHandler(servicoInsumoSvc, estoquePrevisaoSvc)
	tenantFinanceHandler := adminhandler.NewTenantFinanceHandler(financeiroSvc)
	authHandler := adminhandler.NewAuthHandler(authSvc)
	bootstrapAPI := adminhandler.NewBootstrapAPIHandler(
		bootstrapSvc,
		estabelecimentoSvc,
		agendaSvc,
		especialidadeSvc,
		profissionalSvc,
		procedimentoSvc,
		financeiroSvc,
		filaSvc,
		insumoSvc,
		earlySlotSvc,
	)
	go earlySlotSvc.RunExpirationWorker(context.Background(), 30*time.Second)

	dashboardHandler, err := adminhandler.NewDashboardDonaHandler(financeiroSvc, estabelecimentoSvc)
	if err != nil {
		log.Fatalf("carregar templates do painel: %v", err)
	}

	dashboardProfHandler, err := adminhandler.NewDashboardProfissionalHandler(agendaSvc, financeiroSvc)
	if err != nil {
		log.Fatalf("carregar templates da profissional: %v", err)
	}

	adminConfigHandler, err := adminhandler.NewAdminConfigHandler(procedimentoSvc, profissionalSvc, especialidadeSvc, financeiroSvc, estabelecimentoSvc)
	if err != nil {
		log.Fatalf("carregar templates admin: %v", err)
	}

	superAdminUIHandler, err := adminhandler.NewSuperAdminUIHandler(estabelecimentoSvc, planoSaasSvc, authSvc, saasGuard)
	if err != nil {
		log.Fatalf("carregar templates super admin: %v", err)
	}

	saasValidation := security.SaaSValidationMiddleware(saasGuard)
	whatsAppFeature := security.RequireWhatsAppEnabled(whatsAppGate)

	// Dona do salão: role DONA + assinatura SaaS ativa
	donaRoute := func(h http.HandlerFunc) http.Handler {
		return chainHandlers(
			security.RequireDona,
			http.HandlerFunc(saasValidation(h)),
		)
	}

	// Dona + assinatura ativa + recurso WhatsApp liberado pelo SUPER_ADMIN.
	donaWhatsAppRoute := func(h http.HandlerFunc) http.Handler {
		return chainHandlers(
			security.RequireDona,
			http.HandlerFunc(saasValidation(whatsAppFeature(h))),
		)
	}

	// Dona ou secretaria: agenda, clientes e cobrança (assinatura SaaS ativa)
	tenantStaffRoute := func(h http.HandlerFunc) http.Handler {
		return chainHandlers(
			security.RequireTenantStaff,
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

	// Entrada do sistema (público)
	mux.HandleFunc("GET /login", authHandler.LoginPage)
	mux.HandleFunc("GET /login/superadmin", authHandler.LoginPage)
	mux.HandleFunc("GET /login/dona", authHandler.LoginPage)
	mux.HandleFunc("GET /login/profissional", authHandler.LoginPage)

	// Autenticação unificada (público)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /login/superadmin", authHandler.LoginForm)
	mux.HandleFunc("POST /login/dona", authHandler.LoginForm)
	mux.HandleFunc("POST /login/profissional", authHandler.LoginForm)

	// Alteração de senha do próprio usuário (qualquer role autenticada; sem guarda SaaS)
	mux.Handle("POST /api/v1/auth/change-password", security.AuthenticateMiddleware(http.HandlerFunc(authHandler.ChangePassword)))

	// Super Admin — API JSON + UI HTML
	mux.Handle("GET /api/v1/admin/establishments", superAdminRoute(adminEstHandler.List))
	mux.Handle("POST /api/v1/admin/establishments", superAdminRoute(adminEstHandler.Create))
	mux.Handle("PUT /api/v1/admin/establishments/{id}", superAdminRoute(adminEstHandler.Update))
	mux.Handle("PUT /api/v1/admin/establishments/{id}/status", superAdminRoute(adminEstHandler.ToggleStatus))
	mux.Handle("PUT /api/v1/admin/establishments/{id}/toggle-whatsapp", superAdminRoute(adminEstHandler.ToggleWhatsApp))
	mux.Handle("POST /api/v1/admin/establishments/{id}/create-dona", superAdminRoute(adminEstHandler.CreateDona))
	mux.Handle("POST /api/v1/admin/establishments/{id}/logo", superAdminRoute(adminEstHandler.UploadLogo))
	mux.Handle("POST /api/v1/admin/establishments/{id}/assign-plan", superAdminRoute(adminPlansHandler.AssignPlan))
	mux.Handle("GET /api/v1/admin/plans", superAdminRoute(adminPlansHandler.List))
	mux.Handle("POST /api/v1/admin/plans", superAdminRoute(adminPlansHandler.Create))
	mux.Handle("PUT /api/v1/admin/plans/{id}", superAdminRoute(adminPlansHandler.Update))
	mux.Handle("POST /api/v1/admin/avisos", superAdminRoute(adminAvisosHandler.Create))
	mux.Handle("GET /api/v1/admin/avisos", superAdminRoute(adminAvisosHandler.List))
	mux.Handle("PATCH /api/v1/admin/avisos/{id}", superAdminRoute(adminAvisosHandler.Patch))
	mux.Handle("GET /api/v1/admin/bootstrap", superAdminRoute(bootstrapAPI.AdminBootstrap))

	// Inbox de avisos globais (qualquer role autenticada; sem guarda SaaS)
	mux.Handle("GET /api/v1/me/notificacoes", security.AuthenticateMiddleware(http.HandlerFunc(meNotificacoesHandler.List)))
	mux.Handle("POST /api/v1/me/notificacoes/{id}/read", security.AuthenticateMiddleware(http.HandlerFunc(meNotificacoesHandler.MarkRead)))

	// Bootstrap e REST JSON para o front-end React
	mux.Handle("GET /api/v1/bootstrap", tenantStaffRoute(bootstrapAPI.TenantBootstrap))
	mux.Handle("GET /api/v1/dashboard/gerencial", donaRoute(bootstrapAPI.DashboardGerencial))
	mux.Handle("GET /api/v1/specialties", donaRoute(bootstrapAPI.ListSpecialties))
	mux.Handle("POST /api/v1/specialties", donaRoute(bootstrapAPI.CreateSpecialty))
	mux.Handle("PUT /api/v1/specialties/{id}", donaRoute(bootstrapAPI.UpdateSpecialty))
	mux.Handle("PUT /api/v1/professionals/{id}", donaRoute(bootstrapAPI.UpdateProfessional))
	mux.Handle("POST /api/v1/professionals/{id}/foto", donaRoute(tenantCatalogHandler.UploadProfessionalFoto))
	mux.Handle("POST /api/v1/appointments", tenantStaffRoute(bootstrapAPI.CreateAppointment))
	mux.Handle("POST /api/v1/appointments/{id}/cancel", tenantStaffRoute(bootstrapAPI.CancelAppointment))
	mux.Handle("PATCH /api/v1/appointments/{id}/early-slot-preference", tenantStaffRoute(earlySlotHandler.SetPreference))
	mux.Handle("GET /api/v1/early-slot-rounds/{id}", tenantStaffRoute(earlySlotHandler.GetRound))
	mux.Handle("GET /api/v1/early-slot-rounds", tenantStaffRoute(earlySlotHandler.ListRounds))
	mux.Handle("POST /api/v1/appointments/{id}/charge", tenantStaffRoute(bootstrapAPI.ChargeAppointment))
	mux.Handle("POST /api/v1/clients", tenantStaffRoute(bootstrapAPI.CreateClient))
	mux.Handle("POST /api/v1/cash-flow", donaRoute(bootstrapAPI.CreateLancamento))
	mux.Handle("POST /api/v1/waitlist/{id}/notify", tenantStaffRoute(bootstrapAPI.MarkFilaNotificada))
	mux.Handle("GET /api/v1/supplies", donaRoute(bootstrapAPI.ListInsumos))
	mux.Handle("POST /api/v1/supplies", donaRoute(bootstrapAPI.CreateInsumo))
	// forecast antes de /supplies/{id} para evitar colisão com path genérico
	mux.Handle("GET /api/v1/supplies/forecast", donaRoute(estoqueHandler.ForecastSupplies))
	mux.Handle("PUT /api/v1/supplies/{id}", donaRoute(bootstrapAPI.UpdateInsumo))
	mux.Handle("POST /api/v1/supplies/{id}/adjust", donaRoute(bootstrapAPI.AdjustInsumoEstoque))
	mux.Handle("DELETE /api/v1/supplies/{id}", donaRoute(bootstrapAPI.DeleteInsumo))

	mux.Handle("GET /api/v1/professional/dashboard", professionalRoute(bootstrapAPI.ProfessionalDashboard))
	mux.Handle("POST /api/v1/professional/appointments", professionalRoute(bootstrapAPI.CreateProfessionalAppointment))
	mux.Handle("POST /api/v1/professional/appointments/{id}/complete", professionalRoute(bootstrapAPI.CompleteAppointment))
	mux.Handle("GET /api/v1/professional/bootstrap", professionalRoute(bootstrapAPI.TenantBootstrap))

	// Go 1.22+ ServeMux rejects overlapping patterns such as
	// GET /public/{slug}/catalog and GET /public/early-slot-offers/{token}
	// (path early-slot-offers/catalog matches both). Dispatch 2-segment GETs
	// through one pattern so public URLs stay stable.
	mux.HandleFunc("GET /api/v1/public/{seg1}/{seg2}", func(w http.ResponseWriter, r *http.Request) {
		seg1 := r.PathValue("seg1")
		seg2 := r.PathValue("seg2")
		switch {
		case seg1 == "early-slot-offers":
			r.SetPathValue("token", seg2)
			earlySlotHandler.GetOffer(w, r)
		case seg2 == "catalog":
			r.SetPathValue("slug", seg1)
			bootstrapAPI.PublicCatalog(w, r)
		case seg2 == "slots":
			r.SetPathValue("slug", seg1)
			publicSlotsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
	mux.Handle("POST /api/v1/public/{slug}/appointments", http.HandlerFunc(bootstrapAPI.CreateAppointment))
	mux.HandleFunc("POST /api/v1/public/early-slot-offers/{token}/accept", earlySlotHandler.Accept)
	mux.HandleFunc("POST /api/v1/public/early-slot-offers/{token}/decline", earlySlotHandler.Decline)
	mux.HandleFunc("PATCH /api/v1/public/appointments/manage/{token}/early-slot-preference", earlySlotHandler.SetPreferencePublic)

	mux.Handle("GET /superadmin/dashboard", superAdminRoute(superAdminUIHandler.Dashboard))
	mux.Handle("POST /superadmin/establishments", superAdminRoute(superAdminUIHandler.CreateEstablishment))
	mux.Handle("POST /superadmin/establishments/{id}/suspend", superAdminRoute(superAdminUIHandler.SuspendEstablishment))
	mux.Handle("POST /superadmin/establishments/{id}/activate", superAdminRoute(superAdminUIHandler.ActivateEstablishment))
	mux.Handle("POST /superadmin/establishments/{id}/renew", superAdminRoute(superAdminUIHandler.RenewEstablishment))
	mux.Handle("POST /superadmin/establishments/{id}/assign-plan", superAdminRoute(superAdminUIHandler.AssignPlanEstablishment))
	mux.Handle("POST /superadmin/establishments/{id}/create-dona", superAdminRoute(superAdminUIHandler.CreateDonaOwner))
	mux.Handle("GET /superadmin/planos", superAdminRoute(superAdminUIHandler.Planos))
	mux.Handle("POST /superadmin/planos", superAdminRoute(superAdminUIHandler.CreatePlan))
	mux.Handle("POST /superadmin/planos/{id}", superAdminRoute(superAdminUIHandler.UpdatePlan))

	// Dona do salão — finanças, configuração e painel gerencial
	mux.Handle("GET /api/v1/service-categories", donaRoute(tenantCatalogHandler.ListServiceCategories))
	mux.Handle("POST /api/v1/service-categories", donaRoute(tenantCatalogHandler.CreateServiceCategory))
	mux.Handle("PUT /api/v1/service-categories/{id}", donaRoute(tenantCatalogHandler.UpdateServiceCategory))
	mux.Handle("DELETE /api/v1/service-categories/{id}", donaRoute(tenantCatalogHandler.DeleteServiceCategory))
	mux.Handle("GET /api/v1/services", donaRoute(tenantCatalogHandler.ListServices))
	mux.Handle("POST /api/v1/services", donaRoute(tenantCatalogHandler.CreateService))
	mux.Handle("PUT /api/v1/services/{id}", donaRoute(tenantCatalogHandler.UpdateService))
	mux.Handle("POST /api/v1/services/{id}/additionals", donaRoute(tenantCatalogHandler.CreateServiceAdditional))
	mux.Handle("GET /api/v1/services/{id}/supplies", donaRoute(estoqueHandler.ListServiceSupplies))
	mux.Handle("POST /api/v1/services/{id}/supplies", donaRoute(estoqueHandler.CreateServiceSupply))
	mux.Handle("PUT /api/v1/services/{id}/supplies", donaRoute(estoqueHandler.ReplaceServiceSupplies))
	mux.Handle("PUT /api/v1/services/{id}/supplies/{linkId}", donaRoute(estoqueHandler.UpdateServiceSupply))
	mux.Handle("DELETE /api/v1/services/{id}/supplies/{linkId}", donaRoute(estoqueHandler.DeleteServiceSupply))
	mux.Handle("GET /api/v1/professionals", donaRoute(tenantCatalogHandler.ListProfessionals))
	mux.Handle("POST /api/v1/professionals", donaRoute(tenantCatalogHandler.CreateProfessional))
	mux.Handle("GET /api/v1/finance/report", donaRoute(tenantFinanceHandler.GetReport))
	mux.Handle("GET /api/v1/finance/professionals/{id}/pending", donaRoute(tenantFinanceHandler.GetProfessionalPending))
	mux.Handle("POST /api/v1/finance/professionals/{id}/pay", donaRoute(tenantFinanceHandler.PayProfessionalCommissions))
	mux.Handle("POST /v1/estabelecimentos/config", donaRoute(configHandler.ServeHTTP))
	mux.Handle("GET /api/v1/estabelecimentos/{id}/notificacoes-agenda", donaRoute(agendaNotificationsHandler.Get))
	mux.Handle("PUT /api/v1/estabelecimentos/{id}/notificacoes-agenda", donaRoute(agendaNotificationsHandler.Put))
	mux.Handle("GET /admin/servicos", donaRoute(adminConfigHandler.Servicos))
	mux.Handle("POST /admin/servicos", donaRoute(adminConfigHandler.CreateServico))
	mux.Handle("POST /admin/servicos/{id}/adicionais", donaRoute(adminConfigHandler.CreateAdicional))
	mux.Handle("GET /admin/equipe", donaRoute(adminConfigHandler.Equipe))
	mux.Handle("POST /admin/equipe", donaRoute(adminConfigHandler.CreateProfissional))
	mux.Handle("POST /admin/equipe/{id}", donaRoute(adminConfigHandler.UpdateProfissional))
	mux.Handle("GET /admin/especialidades", donaRoute(adminConfigHandler.Especialidades))
	mux.Handle("POST /admin/especialidades", donaRoute(adminConfigHandler.CreateEspecialidade))
	mux.Handle("POST /admin/especialidades/{id}", donaRoute(adminConfigHandler.UpdateEspecialidade))
	mux.Handle("GET /admin/caixa", donaRoute(adminConfigHandler.Caixa))
	mux.Handle("POST /admin/caixa/lancamento", donaRoute(adminConfigHandler.CreateLancamento))
	mux.Handle("GET /dashboard/gerencial", donaRoute(dashboardHandler.ServeHTTP))
	mux.Handle("POST /dashboard/gerencial/professionals/{id}/pay", donaRoute(dashboardHandler.PayProfessional))
	mux.Handle("POST /dashboard/gerencial/lancamento", donaRoute(dashboardHandler.RegisterExpense))

	// Profissional parceira — agenda individual isolada por profissional_id do token
	mux.Handle("GET /dashboard/profissional", professionalRoute(dashboardProfHandler.ServeHTTP))
	mux.Handle("GET /dashboard/profissional/timeline", professionalRoute(dashboardProfHandler.Timeline))
	mux.Handle("POST /dashboard/profissional/appointments/{id}/complete", professionalRoute(dashboardProfHandler.CompleteAppointment))

	// API pública — ações de agendamento e webhook WhatsApp Gateway
	// (GET /{slug}/slots e GET early-slot-offers/{token} ficam no dispatcher 2-segmentos acima)
	mux.HandleFunc("POST /api/v1/public/appointments/{id}/approve", publicAppointmentsHandler.Approve)
	mux.HandleFunc("POST /api/v1/public/appointments/{id}/reschedule", publicAppointmentsHandler.Reschedule)
	mux.HandleFunc("GET /api/v1/public/appointments/manage/{token}", publicAppointmentsHandler.Manage)
	mux.HandleFunc("POST /api/v1/public/appointments/manage/{token}/cancel", publicAppointmentsHandler.CancelByManagementToken)
	mux.HandleFunc("POST /api/v1/webhook/whatsapp-callback", whatsAppWebhookHandler.Callback)
	mux.HandleFunc("POST /api/v1/webhook/whatsapp-connected", whatsAppWebhookHandler.Connected)
	mux.HandleFunc("POST /api/v1/webhook/whatsapp-gateway", whatsAppWebhookHandler.Gateway)

	mux.Handle("GET /api/v1/whatsapp/integration", donaRoute(whatsAppIntegrationHandler.GetIntegration))
	mux.Handle("POST /api/v1/whatsapp/integration/start", donaWhatsAppRoute(whatsAppIntegrationHandler.StartConnection))

	// Página pública de agendamento (sem autenticação — cliente final)
	mux.Handle("GET /{slug}", bookingHandler)

	addr := config.ListenAddr()

	log.Printf("AgendaGlow API ouvindo em %s (porta reservada %s — Gateway usa %s)",
		addr, config.DefaultAPIPort, config.GatewayAPIPort)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	worker := service.NewAgendaNotificationWorker(
		db,
		service.NewGatewayNotificationSenderFromEnv(),
		service.AgendaNotificationWorkerOptions{
			BaseURL:  envOrDefault("APP_BASE_URL", "http://localhost:8081"),
			Interval: notificationWorkerInterval(),
		},
	)
	var workerWG sync.WaitGroup
	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		worker.Start(ctx)
	}()

	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("servidor encerrado: %v", err)
		}
		stop()
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	workerWG.Wait()
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

func notificationWorkerInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("AGENDA_NOTIFICATION_WORKER_INTERVAL_SECONDS"))
	if raw == "" {
		return time.Minute
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 1 {
		return time.Minute
	}
	return time.Duration(seconds) * time.Second
}
