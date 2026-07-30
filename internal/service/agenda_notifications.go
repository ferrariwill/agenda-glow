package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	NotificationTypeReservationConfirmation = "CONFIRMACAO_RESERVA"
	NotificationTypeConfirmationRequest     = "PEDIDO_CONFIRMACAO"
	NotificationTypeReminder                = "LEMBRETE"
	NotificationTypeCancellationByCustomer  = "CANCELAMENTO_CLIENTE"
	NotificationTypeCancellationConfirmed   = "CANCELAMENTO_CONFIRMADO"
)

const (
	NotificationStatusPending = "PENDENTE"
	// NotificationStatusRecorded marca auditoria de evento recebido do cliente:
	// nunca é reservada nem enviada pelo worker.
	NotificationStatusRecorded = "REGISTRADO"
)

const (
	CancellationOriginPublicManagement = "gestao_publica"
	CancellationOriginWhatsApp         = "whatsapp"
)

// Um agendamento só recebe mensagem agendada enquanto o horário vale: EM_APROVACAO
// ainda depende da profissional e não dispara reserva, pedido de confirmação ou
// lembrete. A confirmação de cancelamento é o oposto — só faz sentido depois que o
// agendamento já está CANCELADO.
var (
	statusesForScheduledNotifications   = []string{"AGENDADO", "CONFIRMADO"}
	statusesForCancellationNotification = []string{"CANCELADO"}
	postCancellationNotificationTypes   = []string{NotificationTypeCancellationConfirmed}
)

// EligibleAppointmentStatuses lista os status operacionais em que o tipo informado
// ainda pode gerar envio.
func EligibleAppointmentStatuses(notificationType string) []string {
	if notificationType == NotificationTypeCancellationConfirmed {
		return append([]string(nil), statusesForCancellationNotification...)
	}
	return append([]string(nil), statusesForScheduledNotifications...)
}

// BuildManagementURL monta a URL navegável do frontend. Os endpoints JSON de
// consulta/cancelamento permanecem sob /api/v1/public/appointments/manage.
func BuildManagementURL(baseURL, token string) string {
	path := "/p/agendamento/" + strings.TrimSpace(token)
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + path
}

var (
	ErrInvalidNotificationSettings = errors.New("invalid_notification_settings")
	ErrManagementTokenExpired      = errors.New("management_token_expired")
	ErrAppointmentNotCancellable   = errors.New("appointment_not_cancellable")
	ErrCancellationWindowClosed    = errors.New("cancellation_window_closed")
	ErrCancellationReasonRequired  = errors.New("reason_required")
)

var (
	allowedNotificationVariables = []string{
		"nome_salao", "servico", "profissional", "data_hora", "endereco", "link_gestao",
	}
	placeholderPattern     = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)
	managementTokenPattern = regexp.MustCompile(
		`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
	)
)

type NotificationAgendaConfig struct {
	LembretesAtivos               bool     `db:"lembretes_ativos" json:"lembretes_ativos"`
	AntecedenciaConfirmacaoHoras  int      `db:"antecedencia_confirmacao_horas" json:"antecedencia_confirmacao_horas"`
	AntecedenciaLembreteHoras     int      `db:"antecedencia_lembrete_horas" json:"antecedencia_lembrete_horas"`
	JanelaMinimaCancelamentoHoras int      `db:"janela_minima_cancelamento_horas" json:"janela_minima_cancelamento_horas"`
	MotivoCancelamentoObrigatorio bool     `db:"motivo_cancelamento_obrigatorio" json:"motivo_cancelamento_obrigatorio"`
	TemplateConfirmacao           string   `db:"template_confirmacao" json:"template_confirmacao"`
	TemplateLembrete              string   `db:"template_lembrete" json:"template_lembrete"`
	AllowedVariables              []string `db:"-" json:"allowed_variables"`
	WhatsAppStatus                string   `db:"whatsapp_status" json:"whatsapp_status"`
}

func AllowedNotificationVariables() []string {
	return append([]string(nil), allowedNotificationVariables...)
}

func ValidateNotificationAgendaConfig(in NotificationAgendaConfig) error {
	if in.AntecedenciaConfirmacaoHoras < 1 || in.AntecedenciaConfirmacaoHoras > 168 ||
		in.AntecedenciaLembreteHoras < 1 || in.AntecedenciaLembreteHoras > 48 ||
		in.JanelaMinimaCancelamentoHoras < 0 || in.JanelaMinimaCancelamentoHoras > 168 {
		return ErrInvalidNotificationSettings
	}
	if !validNotificationTemplate(in.TemplateConfirmacao) || !validNotificationTemplate(in.TemplateLembrete) {
		return ErrInvalidNotificationSettings
	}
	return nil
}

func validNotificationTemplate(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 2000 || strings.ContainsAny(value, "<>") {
		return false
	}
	allowed := make(map[string]struct{}, len(allowedNotificationVariables))
	for _, name := range allowedNotificationVariables {
		allowed[name] = struct{}{}
	}
	for _, match := range placeholderPattern.FindAllStringSubmatch(value, -1) {
		if _, ok := allowed[match[1]]; !ok {
			return false
		}
	}
	withoutKnown := placeholderPattern.ReplaceAllString(value, "")
	return !strings.Contains(withoutKnown, "{{") && !strings.Contains(withoutKnown, "}}")
}

func renderNotificationTemplate(template string, values map[string]string) (string, error) {
	if !validNotificationTemplate(template) {
		return "", ErrInvalidNotificationSettings
	}
	rendered := placeholderPattern.ReplaceAllStringFunc(template, func(placeholder string) string {
		match := placeholderPattern.FindStringSubmatch(placeholder)
		return values[match[1]]
	})
	return rendered, nil
}

func (s *AgendaService) GetNotificationAgendaConfig(ctx context.Context, establishmentID string) (*NotificationAgendaConfig, error) {
	const query = `
INSERT INTO configuracoes_notificacoes_agenda (estabelecimento_id)
VALUES ($1)
ON CONFLICT (estabelecimento_id) DO UPDATE
SET estabelecimento_id = EXCLUDED.estabelecimento_id
RETURNING lembretes_ativos, antecedencia_confirmacao_horas,
          antecedencia_lembrete_horas, janela_minima_cancelamento_horas,
          motivo_cancelamento_obrigatorio, template_confirmacao, template_lembrete,
          (SELECT whatsapp_status FROM estabelecimentos WHERE id = $1) AS whatsapp_status
`
	var out NotificationAgendaConfig
	if err := s.db.GetContext(ctx, &out, query, establishmentID); err != nil {
		return nil, fmt.Errorf("buscar configurações de notificação: %w", err)
	}
	out.AllowedVariables = AllowedNotificationVariables()
	return &out, nil
}

func (s *AgendaService) UpdateNotificationAgendaConfig(ctx context.Context, establishmentID string, in NotificationAgendaConfig) (*NotificationAgendaConfig, error) {
	in.TemplateConfirmacao = strings.TrimSpace(in.TemplateConfirmacao)
	in.TemplateLembrete = strings.TrimSpace(in.TemplateLembrete)
	if err := ValidateNotificationAgendaConfig(in); err != nil {
		return nil, err
	}
	const query = `
INSERT INTO configuracoes_notificacoes_agenda (
    estabelecimento_id, lembretes_ativos, antecedencia_confirmacao_horas,
    antecedencia_lembrete_horas, janela_minima_cancelamento_horas,
    motivo_cancelamento_obrigatorio, template_confirmacao, template_lembrete
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (estabelecimento_id) DO UPDATE SET
    lembretes_ativos = EXCLUDED.lembretes_ativos,
    antecedencia_confirmacao_horas = EXCLUDED.antecedencia_confirmacao_horas,
    antecedencia_lembrete_horas = EXCLUDED.antecedencia_lembrete_horas,
    janela_minima_cancelamento_horas = EXCLUDED.janela_minima_cancelamento_horas,
    motivo_cancelamento_obrigatorio = EXCLUDED.motivo_cancelamento_obrigatorio,
    template_confirmacao = EXCLUDED.template_confirmacao,
    template_lembrete = EXCLUDED.template_lembrete,
    atualizado_em = NOW()
RETURNING lembretes_ativos, antecedencia_confirmacao_horas,
          antecedencia_lembrete_horas, janela_minima_cancelamento_horas,
          motivo_cancelamento_obrigatorio, template_confirmacao, template_lembrete,
          (SELECT whatsapp_status FROM estabelecimentos WHERE id = $1) AS whatsapp_status
`
	var out NotificationAgendaConfig
	if err := s.db.GetContext(ctx, &out, query, establishmentID, in.LembretesAtivos,
		in.AntecedenciaConfirmacaoHoras, in.AntecedenciaLembreteHoras,
		in.JanelaMinimaCancelamentoHoras, in.MotivoCancelamentoObrigatorio,
		in.TemplateConfirmacao, in.TemplateLembrete); err != nil {
		return nil, fmt.Errorf("atualizar configurações de notificação: %w", err)
	}
	out.AllowedVariables = AllowedNotificationVariables()
	return &out, nil
}

type AppointmentManagementView struct {
	Appointment struct {
		Service              string    `json:"service"`
		Professional         string    `json:"professional"`
		StartsAt             time.Time `json:"starts_at"`
		Status               string    `json:"status"`
		CustomerConfirmation string    `json:"customer_confirmation"`
	} `json:"appointment"`
	Establishment struct {
		Name         string `json:"name"`
		ContactPhone string `json:"contact_phone"`
	} `json:"establishment"`
	Cancellation CancellationAvailability `json:"cancellation"`
}

type CancellationAvailability struct {
	Allowed            bool    `json:"allowed"`
	ReasonRequired     bool    `json:"reason_required"`
	MinimumNoticeHours int     `json:"minimum_notice_hours"`
	DenialReason       *string `json:"denial_reason"`
}

type managementAppointment struct {
	ID                             string    `db:"id"`
	EstablishmentID                string    `db:"estabelecimento_id"`
	Service                        string    `db:"servico"`
	Professional                   string    `db:"profissional"`
	StartsAt                       time.Time `db:"data_hora_inicio"`
	Status                         string    `db:"status"`
	CustomerConfirmation           string    `db:"confirmacao_cliente"`
	TokenExpiresAt                 time.Time `db:"gestao_token_expires_at"`
	EstablishmentName              string    `db:"nome_salao"`
	ContactPhone                   string    `db:"telefone_contato"`
	MinimumCancellationNoticeHours int       `db:"janela_minima_cancelamento_horas"`
	ReasonRequired                 bool      `db:"motivo_cancelamento_obrigatorio"`
}

func cancellationAvailability(status string, startsAt, now time.Time, minimumHours int, reasonRequired bool) CancellationAvailability {
	out := CancellationAvailability{
		Allowed:            true,
		ReasonRequired:     reasonRequired,
		MinimumNoticeHours: minimumHours,
	}
	var denial string
	switch status {
	case "CANCELADO":
		denial = "appointment_cancelled"
	case "CONCLUIDO":
		denial = "appointment_completed"
	default:
		if now.Add(time.Duration(minimumHours) * time.Hour).After(startsAt) {
			denial = "cancellation_window_closed"
		}
	}
	if denial != "" {
		out.Allowed = false
		out.DenialReason = &denial
	}
	return out
}

func (s *AgendaService) GetAppointmentManagement(ctx context.Context, token string, now time.Time) (*AppointmentManagementView, error) {
	ag, err := s.lookupManagementAppointment(ctx, s.db, token, false)
	if err != nil {
		return nil, err
	}
	if !now.Before(ag.TokenExpiresAt) {
		return nil, ErrManagementTokenExpired
	}
	var out AppointmentManagementView
	out.Appointment.Service = ag.Service
	out.Appointment.Professional = ag.Professional
	out.Appointment.StartsAt = ag.StartsAt
	out.Appointment.Status = ag.Status
	out.Appointment.CustomerConfirmation = ag.CustomerConfirmation
	out.Establishment.Name = ag.EstablishmentName
	out.Establishment.ContactPhone = ag.ContactPhone
	out.Cancellation = cancellationAvailability(ag.Status, ag.StartsAt, now,
		ag.MinimumCancellationNoticeHours, ag.ReasonRequired)
	return &out, nil
}

type PublicCancellationResult struct {
	Status               string `json:"status"`
	AppointmentStatus    string `json:"appointment_status"`
	CustomerConfirmation string `json:"customer_confirmation"`
	SlotReleased         bool   `json:"slot_released"`
}

type CancellationWindowError struct {
	MinimumNoticeHours int
	ContactPhone       string
}

func (e *CancellationWindowError) Error() string { return ErrCancellationWindowClosed.Error() }
func (e *CancellationWindowError) Unwrap() error { return ErrCancellationWindowClosed }

func (s *AgendaService) CancelAppointmentByManagementToken(ctx context.Context, token, reason string, now time.Time) (*PublicCancellationResult, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar cancelamento: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	ag, err := s.lookupManagementAppointment(ctx, tx, token, true)
	if err != nil {
		return nil, err
	}
	if !now.Before(ag.TokenExpiresAt) {
		return nil, ErrManagementTokenExpired
	}
	result, err := cancelManagementAppointment(ctx, tx, ag, reason, CancellationOriginPublicManagement, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmar cancelamento: %w", err)
	}
	return result, nil
}

type sqlxGetter interface {
	GetContext(context.Context, any, string, ...any) error
}

func (s *AgendaService) lookupManagementAppointment(ctx context.Context, db sqlxGetter, token string, lock bool) (*managementAppointment, error) {
	token = strings.TrimSpace(token)
	if !managementTokenPattern.MatchString(token) {
		return nil, ErrAgendamentoNaoEncontrado
	}
	query := `
SELECT a.id, a.estabelecimento_id, s.nome AS servico, p.nome AS profissional,
       a.data_hora_inicio, a.status, a.confirmacao_cliente, a.gestao_token_expires_at,
       e.nome_comercial AS nome_salao, COALESCE(e.whatsapp_phone_number, '') AS telefone_contato,
       cfg.janela_minima_cancelamento_horas, cfg.motivo_cancelamento_obrigatorio
FROM agendamentos a
JOIN estabelecimentos e ON e.id = a.estabelecimento_id
JOIN servicos s ON s.id = a.servico_id AND s.estabelecimento_id = a.estabelecimento_id
JOIN profissionais p ON p.id = a.profissional_id AND p.estabelecimento_id = a.estabelecimento_id
JOIN configuracoes_notificacoes_agenda cfg ON cfg.estabelecimento_id = a.estabelecimento_id
WHERE a.gestao_token = $1
`
	if lock {
		query += " FOR UPDATE OF a"
	}
	var ag managementAppointment
	if err := db.GetContext(ctx, &ag, query, token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAgendamentoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar agendamento por token: %w", err)
	}
	return &ag, nil
}

func cancelManagementAppointment(ctx context.Context, tx *sqlx.Tx, ag *managementAppointment, reason, origin string, now time.Time) (*PublicCancellationResult, error) {
	if ag.Status == "CANCELADO" && ag.CustomerConfirmation == "CANCELADO_CLIENTE" {
		return cancelledResult(false), nil
	}
	if ag.Status == "CONCLUIDO" || ag.Status == "CANCELADO" {
		return nil, ErrAppointmentNotCancellable
	}
	reason = strings.TrimSpace(reason)
	if ag.ReasonRequired && reason == "" {
		return nil, ErrCancellationReasonRequired
	}
	if now.Add(time.Duration(ag.MinimumCancellationNoticeHours) * time.Hour).After(ag.StartsAt) {
		return nil, &CancellationWindowError{
			MinimumNoticeHours: ag.MinimumCancellationNoticeHours,
			ContactPhone:       ag.ContactPhone,
		}
	}
	const update = `
UPDATE agendamentos
SET status = 'CANCELADO',
    confirmacao_cliente = 'CANCELADO_CLIENTE',
    motivo_cancelamento_cliente = NULLIF($3, '')
WHERE id = $1 AND estabelecimento_id = $2
`
	if _, err := tx.ExecContext(ctx, update, ag.ID, ag.EstablishmentID, reason); err != nil {
		return nil, fmt.Errorf("cancelar agendamento: %w", err)
	}
	if err := recordCancellationAudit(ctx, tx, ag.EstablishmentID, ag.ID, origin, reason != "", now); err != nil {
		return nil, err
	}
	if err := enqueueNotification(ctx, tx, ag.EstablishmentID, ag.ID, NotificationTypeCancellationConfirmed); err != nil {
		return nil, err
	}
	return cancelledResult(true), nil
}

// recordCancellationAudit grava o cancelamento vindo do cliente como linha terminal
// de auditoria, na mesma transação do cancelamento e sem gerar envio. O resumo não
// carrega telefone, token nem o texto do motivo.
func recordCancellationAudit(ctx context.Context, tx *sqlx.Tx, establishmentID, appointmentID, origin string, reasonProvided bool, now time.Time) error {
	summary, err := json.Marshal(map[string]any{
		"evento":           "cancelamento_cliente",
		"origem":           origin,
		"motivo_informado": reasonProvided,
	})
	if err != nil {
		return fmt.Errorf("resumir auditoria de cancelamento: %w", err)
	}
	const insert = `
INSERT INTO agendamento_notificacoes (
    estabelecimento_id, agendamento_id, tipo, status_envio,
    proxima_tentativa_em, payload_resumido
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (estabelecimento_id, agendamento_id, tipo) DO NOTHING
`
	if _, err := tx.ExecContext(ctx, insert, establishmentID, appointmentID,
		NotificationTypeCancellationByCustomer, NotificationStatusRecorded, now, summary); err != nil {
		return fmt.Errorf("registrar auditoria de cancelamento: %w", err)
	}
	return nil
}

func cancelledResult(slotReleased bool) *PublicCancellationResult {
	return &PublicCancellationResult{
		Status:               "cancelled",
		AppointmentStatus:    "CANCELADO",
		CustomerConfirmation: "CANCELADO_CLIENTE",
		SlotReleased:         slotReleased,
	}
}

func enqueueNotification(ctx context.Context, tx *sqlx.Tx, establishmentID, appointmentID, notificationType string) error {
	const insert = `
INSERT INTO agendamento_notificacoes (
    estabelecimento_id, agendamento_id, tipo, status_envio, proxima_tentativa_em
) VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (estabelecimento_id, agendamento_id, tipo) DO NOTHING
`
	if _, err := tx.ExecContext(ctx, insert, establishmentID, appointmentID, notificationType, NotificationStatusPending); err != nil {
		return fmt.Errorf("enfileirar notificação %s: %w", notificationType, err)
	}
	return nil
}

type NotificationSender interface {
	Send(context.Context, WhatsAppNotificationInput) (string, error)
}

type AgendaNotificationWorker struct {
	db         *sqlx.DB
	sender     NotificationSender
	baseURL    string
	interval   time.Duration
	maxRetries int
	now        func() time.Time
	onError    func(error)
}

type AgendaNotificationWorkerOptions struct {
	BaseURL    string
	Interval   time.Duration
	MaxRetries int
	Now        func() time.Time
	OnError    func(error)
}

func NewAgendaNotificationWorker(db *sqlx.DB, sender NotificationSender, opts AgendaNotificationWorkerOptions) *AgendaNotificationWorker {
	if opts.Interval <= 0 {
		opts.Interval = time.Minute
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 5
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.OnError == nil {
		opts.OnError = func(err error) {
			log.Printf("agenda notification worker: %v", err)
		}
	}
	return &AgendaNotificationWorker{
		db: db, sender: sender, baseURL: strings.TrimRight(opts.BaseURL, "/"),
		interval: opts.Interval, maxRetries: opts.MaxRetries, now: opts.Now,
		onError: opts.OnError,
	}
}

func (w *AgendaNotificationWorker) Start(ctx context.Context) {
	w.runLogged(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runLogged(ctx)
		}
	}
}

func (w *AgendaNotificationWorker) runLogged(ctx context.Context) {
	if err := w.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
		// No payload, phone or management token is logged.
		w.onError(err)
	}
}

func (w *AgendaNotificationWorker) RunOnce(ctx context.Context) error {
	now := w.now()
	if _, err := w.db.ExecContext(ctx, `
UPDATE agendamento_notificacoes
SET status_envio = 'FALHOU', proxima_tentativa_em = $1::timestamp, atualizado_em = $1::timestamp,
    ultimo_erro = 'reserva de envio expirada'
WHERE status_envio = 'ENVIANDO'
  AND atualizado_em < $1::timestamp - INTERVAL '15 minutes'
`, now); err != nil {
		return fmt.Errorf("recuperar notificações travadas: %w", err)
	}
	if err := w.discoverDue(ctx, now); err != nil {
		return err
	}
	for {
		job, err := w.reserveNext(ctx, now)
		if err != nil {
			return err
		}
		if job == nil {
			return nil
		}
		var externalID string
		variables, sendErr := w.notificationVariables(job)
		if sendErr == nil {
			externalID, sendErr = w.sender.Send(ctx, WhatsAppNotificationInput{
				TenantID: job.EstablishmentID, PhoneNumber: job.Phone,
				TemplateName: templateNameForType(job.Type), AppointmentID: job.AppointmentID,
				Variables: variables,
			})
		}
		if err := w.finish(ctx, job, externalID, sendErr, now); err != nil {
			return err
		}
	}
}

// agendaNotificationDiscoverySQL enfileira as notificações vencendo. O par
// whatsapp_enabled + whatsapp_status é o mesmo gate composto da fila de
// antecipação: salão sem canal vivo não enfileira envio.
const agendaNotificationDiscoverySQL = `
INSERT INTO agendamento_notificacoes (
    estabelecimento_id, agendamento_id, tipo, status_envio, proxima_tentativa_em
)
SELECT a.estabelecimento_id, a.id, due.tipo, 'PENDENTE', $1::timestamp
FROM agendamentos a
JOIN clientes c ON c.id = a.cliente_id AND c.estabelecimento_id = a.estabelecimento_id
JOIN estabelecimentos e ON e.id = a.estabelecimento_id
JOIN configuracoes_notificacoes_agenda cfg ON cfg.estabelecimento_id = a.estabelecimento_id
CROSS JOIN LATERAL (
    SELECT 'PEDIDO_CONFIRMACAO'::varchar AS tipo
    WHERE a.confirmacao_cliente = 'PENDENTE'
      AND a.data_hora_inicio <= $1::timestamp + make_interval(hours => cfg.antecedencia_confirmacao_horas)
    UNION ALL
    SELECT 'LEMBRETE'::varchar
    WHERE a.data_hora_inicio <= $1::timestamp + make_interval(hours => cfg.antecedencia_lembrete_horas)
) due
WHERE e.ativo = TRUE AND e.whatsapp_status = 'CONECTADO'
  AND COALESCE(e.whatsapp_enabled, FALSE) = TRUE
  AND cfg.lembretes_ativos = TRUE
  AND a.status = ANY($2)
  AND a.data_hora_inicio > $1::timestamp
  AND BTRIM(c.telefone) <> ''
ON CONFLICT (estabelecimento_id, agendamento_id, tipo) DO NOTHING
`

func (w *AgendaNotificationWorker) discoverDue(ctx context.Context, now time.Time) error {
	if _, err := w.db.ExecContext(ctx, agendaNotificationDiscoverySQL, now, pq.Array(statusesForScheduledNotifications)); err != nil {
		return fmt.Errorf("descobrir notificações vencendo: %w", err)
	}
	return nil
}

type notificationJob struct {
	ID                string    `db:"id"`
	EstablishmentID   string    `db:"estabelecimento_id"`
	AppointmentID     string    `db:"agendamento_id"`
	Type              string    `db:"tipo"`
	Attempts          int       `db:"tentativas"`
	Phone             string    `db:"telefone"`
	EstablishmentName string    `db:"nome_salao"`
	Service           string    `db:"servico"`
	Professional      string    `db:"profissional"`
	StartsAt          time.Time `db:"data_hora_inicio"`
	Address           string    `db:"endereco"`
	ManagementToken   string    `db:"gestao_token"`
	TemplateConfirm   string    `db:"template_confirmacao"`
	TemplateReminder  string    `db:"template_lembrete"`
}

func (w *AgendaNotificationWorker) notificationVariables(job *notificationJob) ([]string, error) {
	dateTime := job.StartsAt.Format("02/01/2006 15:04")
	managementURL := BuildManagementURL(w.baseURL, job.ManagementToken)
	values := map[string]string{
		"nome_salao":   job.EstablishmentName,
		"servico":      job.Service,
		"profissional": job.Professional,
		"data_hora":    dateTime,
		"endereco":     job.Address,
		"link_gestao":  managementURL,
	}
	var configuredTemplate string
	switch job.Type {
	case NotificationTypeReservationConfirmation, NotificationTypeConfirmationRequest:
		configuredTemplate = job.TemplateConfirm
	case NotificationTypeReminder:
		configuredTemplate = job.TemplateReminder
	}
	if configuredTemplate != "" {
		rendered, err := renderNotificationTemplate(configuredTemplate, values)
		if err != nil {
			return nil, err
		}
		return []string{rendered}, nil
	}
	return []string{
		job.EstablishmentName, job.Service, job.Professional,
		dateTime, job.Address, managementURL,
	}, nil
}

func (w *AgendaNotificationWorker) reserveNext(ctx context.Context, now time.Time) (*notificationJob, error) {
	tx, err := w.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck
	const query = `
SELECT n.id, n.estabelecimento_id, n.agendamento_id, n.tipo, n.tentativas,
       c.telefone, e.nome_comercial AS nome_salao, s.nome AS servico,
       p.nome AS profissional, a.data_hora_inicio,
       CONCAT_WS(', ', NULLIF(e.logradouro, ''), NULLIF(e.cidade, ''), NULLIF(e.uf, '')) AS endereco,
       a.gestao_token::text AS gestao_token,
       cfg.template_confirmacao, cfg.template_lembrete
FROM agendamento_notificacoes n
JOIN agendamentos a ON a.id = n.agendamento_id AND a.estabelecimento_id = n.estabelecimento_id
JOIN clientes c ON c.id = a.cliente_id AND c.estabelecimento_id = a.estabelecimento_id
JOIN servicos s ON s.id = a.servico_id AND s.estabelecimento_id = a.estabelecimento_id
JOIN profissionais p ON p.id = a.profissional_id AND p.estabelecimento_id = a.estabelecimento_id
JOIN estabelecimentos e ON e.id = a.estabelecimento_id
JOIN configuracoes_notificacoes_agenda cfg ON cfg.estabelecimento_id = a.estabelecimento_id
WHERE n.status_envio IN ('PENDENTE', 'FALHOU')
  AND n.proxima_tentativa_em <= $1 AND n.tentativas < $2
  AND e.ativo = TRUE AND e.whatsapp_status = 'CONECTADO'
  AND COALESCE(e.whatsapp_enabled, FALSE) = TRUE
  AND cfg.lembretes_ativos = TRUE
  AND CASE WHEN n.tipo = ANY($3) THEN a.status = ANY($4) ELSE a.status = ANY($5) END
  AND BTRIM(c.telefone) <> ''
ORDER BY n.proxima_tentativa_em, n.criado_em
FOR UPDATE OF n SKIP LOCKED
LIMIT 1
`
	var job notificationJob
	if err := tx.GetContext(ctx, &job, query, now, w.maxRetries,
		pq.Array(postCancellationNotificationTypes),
		pq.Array(statusesForCancellationNotification),
		pq.Array(statusesForScheduledNotifications)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("reservar notificação: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE agendamento_notificacoes
SET status_envio = 'ENVIANDO', tentativas = tentativas + 1, atualizado_em = $2
WHERE id = $1
`, job.ID, now); err != nil {
		return nil, fmt.Errorf("marcar notificação enviando: %w", err)
	}
	job.Attempts++
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &job, nil
}

func (w *AgendaNotificationWorker) finish(ctx context.Context, job *notificationJob, externalID string, sendErr error, now time.Time) error {
	if sendErr == nil {
		summary, _ := json.Marshal(map[string]any{"template": templateNameForType(job.Type), "attempt": job.Attempts})
		_, err := w.db.ExecContext(ctx, `
UPDATE agendamento_notificacoes
SET status_envio = 'ENVIADO', enviado_em = $2, external_message_id = NULLIF($3, ''),
    payload_resumido = $4, ultimo_erro = NULL, atualizado_em = $2
WHERE id = $1 AND status_envio = 'ENVIANDO'
`, job.ID, now, externalID, summary)
		return err
	}
	delay := retryBackoff(job.Attempts)
	status := "FALHOU"
	if job.Attempts >= w.maxRetries {
		delay = 365 * 24 * time.Hour
	}
	_, err := w.db.ExecContext(ctx, `
UPDATE agendamento_notificacoes
SET status_envio = $2, proxima_tentativa_em = $3, ultimo_erro = $4, atualizado_em = $5
WHERE id = $1 AND status_envio = 'ENVIANDO'
`, job.ID, status, now.Add(delay), truncateError(sendErr.Error()), now)
	return err
}

func retryBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := time.Minute << (attempt - 1)
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}

func truncateError(value string) string {
	if len(value) > 1000 {
		return value[:1000]
	}
	return value
}

func templateNameForType(notificationType string) string {
	switch notificationType {
	case NotificationTypeReservationConfirmation:
		return "confirmacao_reserva"
	case NotificationTypeConfirmationRequest:
		return "pedido_confirmacao_agenda"
	case NotificationTypeCancellationConfirmed:
		return "cancelamento_confirmado"
	default:
		return whatsappTemplateLembrete
	}
}
