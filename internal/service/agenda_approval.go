package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	ErrAgendamentoNaoEmAprovacao = errors.New("agendamento não está aguardando aprovação")
	ErrHorarioIndisponivel       = errors.New("horário indisponível para o procedimento")
	ErrHorarioInvalido           = errors.New("horário informado é inválido")
	ErrProfissionalSemEmail      = errors.New("profissional não possui e-mail cadastrado")
)

type Mailer interface {
	SendHTML(ctx context.Context, to, subject, htmlBody string) error
}

type AgendaOptions struct {
	BaseURL string
	Mailer  Mailer
}

type agendamentoPendente struct {
	ID                 string    `db:"id"`
	EstabelecimentoID  string    `db:"estabelecimento_id"`
	ProfissionalID     string    `db:"profissional_id"`
	ServicoID          string    `db:"servico_id"`
	DataHoraInicio     time.Time `db:"data_hora_inicio"`
	DataHoraFim        time.Time `db:"data_hora_fim"`
	Status             string    `db:"status"`
	ClienteNome        string    `db:"cliente_nome"`
	ClienteTelefone    string    `db:"cliente_telefone"`
	ServicoNome        string    `db:"servico_nome"`
	ProfissionalNome   string    `db:"profissional_nome"`
	ProfissionalEmail  *string   `db:"profissional_email"`
}

type ResultadoAgendamento struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	MinutosInvadidos int    `json:"minutos_invadidos,omitempty"`
}

// ApproveAppointment confirma um encaixe previamente marcado como EM_APROVACAO.
func (s *AgendaService) ApproveAppointment(ctx context.Context, appointmentID string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	ag, err := s.lockAgendamento(ctx, tx, appointmentID)
	if err != nil {
		return err
	}
	if ag.Status != "EM_APROVACAO" {
		return ErrAgendamentoNaoEmAprovacao
	}

	const update = `
UPDATE agendamentos
SET status = 'AGENDADO'
WHERE id = $1
`
	if _, err := tx.ExecContext(ctx, update, appointmentID); err != nil {
		return fmt.Errorf("aprovar agendamento: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar aprovação: %w", err)
	}

	return nil
}

// RescheduleAppointment move o agendamento para um horário totalmente livre e confirma.
func (s *AgendaService) RescheduleAppointment(ctx context.Context, appointmentID, newTimeHHMM string) error {
	newTimeHHMM = strings.TrimSpace(newTimeHHMM)
	if newTimeHHMM == "" {
		return ErrHorarioInvalido
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	ag, err := s.lockAgendamento(ctx, tx, appointmentID)
	if err != nil {
		return err
	}
	if ag.Status != "EM_APROVACAO" {
		return ErrAgendamentoNaoEmAprovacao
	}

	duracao := int(ag.DataHoraFim.Sub(ag.DataHoraInicio).Minutes())
	if duracao <= 0 {
		return fmt.Errorf("duração inválida do agendamento")
	}

	novoInicio, err := combinarDataHorario(ag.DataHoraInicio, newTimeHHMM)
	if err != nil {
		return err
	}
	novoFim := novoInicio.Add(time.Duration(duracao) * time.Minute)

	if err := s.intervaloTotalmenteLivre(ctx, tx, ag.EstabelecimentoID, ag.ProfissionalID, novoInicio, novoFim, appointmentID); err != nil {
		return err
	}

	const update = `
UPDATE agendamentos
SET data_hora_inicio = $2,
    data_hora_fim = $3,
    status = 'AGENDADO'
WHERE id = $1
`
	if _, err := tx.ExecContext(ctx, update, appointmentID, novoInicio, novoFim); err != nil {
		return fmt.Errorf("reagendar agendamento: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar reagendamento: %w", err)
	}

	go s.notificarClienteReagendamento(ag, novoInicio)

	return nil
}

func combinarDataHorario(diaReferencia time.Time, hhmm string) (time.Time, error) {
	parsed, err := time.Parse("15:04", hhmm)
	if err != nil {
		return time.Time{}, ErrHorarioInvalido
	}
	y, m, d := diaReferencia.Date()
	loc := diaReferencia.Location()
	return time.Date(y, m, d, parsed.Hour(), parsed.Minute(), 0, 0, loc), nil
}

func (s *AgendaService) lockAgendamento(ctx context.Context, tx *sqlx.Tx, appointmentID string) (*agendamentoPendente, error) {
	const query = `
SELECT
    a.id,
    a.estabelecimento_id,
    a.profissional_id,
    a.servico_id,
    a.data_hora_inicio,
    a.data_hora_fim,
    a.status,
    c.nome AS cliente_nome,
    c.telefone AS cliente_telefone,
    s.nome AS servico_nome,
    p.nome AS profissional_nome,
    p.email AS profissional_email
FROM agendamentos a
INNER JOIN clientes c ON c.id = a.cliente_id
INNER JOIN servicos s ON s.id = a.servico_id
INNER JOIN profissionais p ON p.id = a.profissional_id
WHERE a.id = $1
FOR UPDATE OF a
`
	var ag agendamentoPendente
	if err := tx.GetContext(ctx, &ag, query, appointmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAgendamentoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar agendamento: %w", err)
	}
	return &ag, nil
}

func (s *AgendaService) buscarAgendamento(ctx context.Context, appointmentID string) (*agendamentoPendente, error) {
	const query = `
SELECT
    a.id,
    a.estabelecimento_id,
    a.profissional_id,
    a.servico_id,
    a.data_hora_inicio,
    a.data_hora_fim,
    a.status,
    c.nome AS cliente_nome,
    c.telefone AS cliente_telefone,
    s.nome AS servico_nome,
    p.nome AS profissional_nome,
    p.email AS profissional_email
FROM agendamentos a
INNER JOIN clientes c ON c.id = a.cliente_id
INNER JOIN servicos s ON s.id = a.servico_id
INNER JOIN profissionais p ON p.id = a.profissional_id
WHERE a.id = $1
`
	var ag agendamentoPendente
	if err := s.db.GetContext(ctx, &ag, query, appointmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAgendamentoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar agendamento: %w", err)
	}
	return &ag, nil
}

func (s *AgendaService) listarAdicionaisAgendamento(ctx context.Context, appointmentID string) ([]string, error) {
	const query = `SELECT adicional_id FROM agendamento_adicionais WHERE agendamento_id = $1`
	var ids []string
	if err := s.db.SelectContext(ctx, &ids, query, appointmentID); err != nil {
		return nil, fmt.Errorf("listar adicionais: %w", err)
	}
	return ids, nil
}

func (s *AgendaService) dispararEmailAprovacaoEncaixe(appointmentID string, minutosInvadidos int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ag, err := s.buscarAgendamento(ctx, appointmentID)
	if err != nil {
		log.Printf("email encaixe: buscar agendamento %s: %v", appointmentID, err)
		return
	}
	if ag.ProfissionalEmail == nil || strings.TrimSpace(*ag.ProfissionalEmail) == "" {
		log.Printf("email encaixe: profissional %s sem e-mail cadastrado", ag.ProfissionalID)
		return
	}

	adicionais, err := s.listarAdicionaisAgendamento(ctx, appointmentID)
	if err != nil {
		log.Printf("email encaixe: listar adicionais %s: %v", appointmentID, err)
		return
	}

	slots, err := s.GetAvailableSlots(
		ctx,
		ag.EstabelecimentoID,
		ag.ProfissionalID,
		ag.DataHoraInicio,
		ag.ServicoID,
		adicionais,
	)
	if err != nil {
		log.Printf("email encaixe: calcular horários livres %s: %v", appointmentID, err)
		return
	}

	alternativas := primeirosHorariosAlternativos(slots, ag.DataHoraInicio.Format("15:04"), 3)
	baseURL := strings.TrimRight(s.baseURL, "/")

	htmlBody, err := renderEmailAprovacaoEncaixe(emailAprovacaoEncaixeData{
		ProfissionalNome:  ag.ProfissionalNome,
		ClienteNome:       ag.ClienteNome,
		ServicoNome:       ag.ServicoNome,
		HorarioSolicitado: ag.DataHoraInicio.Format("15:04"),
		MinutosInvadidos:  minutosInvadidos,
		ApproveURL:        fmt.Sprintf("%s/api/v1/public/appointments/%s/approve", baseURL, appointmentID),
		Alternativas:      buildAlternativasReschedule(baseURL, appointmentID, alternativas),
	})
	if err != nil {
		log.Printf("email encaixe: render HTML %s: %v", appointmentID, err)
		return
	}

	mailer := s.mailer
	if mailer == nil {
		mailer = LogMailer{}
	}

	subject := fmt.Sprintf("[AgendaGlow] Encaixe pendente — %s (%s)", ag.ClienteNome, ag.ServicoNome)
	if err := mailer.SendHTML(ctx, strings.TrimSpace(*ag.ProfissionalEmail), subject, htmlBody); err != nil {
		log.Printf("email encaixe: enviar para %s: %v", *ag.ProfissionalEmail, err)
	}
}

func primeirosHorariosAlternativos(slots []string, horarioSolicitado string, limite int) []string {
	if limite <= 0 {
		return nil
	}
	out := make([]string, 0, limite)
	for _, slot := range slots {
		if slot == horarioSolicitado {
			continue
		}
		out = append(out, slot)
		if len(out) >= limite {
			break
		}
	}
	return out
}

type alternativaHorario struct {
	Horario string
	URL     string
}

type emailAprovacaoEncaixeData struct {
	ProfissionalNome  string
	ClienteNome       string
	ServicoNome       string
	HorarioSolicitado string
	MinutosInvadidos  int
	ApproveURL        string
	Alternativas      []alternativaHorario
}

func buildAlternativasReschedule(baseURL, appointmentID string, horarios []string) []alternativaHorario {
	out := make([]alternativaHorario, 0, len(horarios))
	for _, h := range horarios {
		out = append(out, alternativaHorario{
			Horario: h,
			URL: fmt.Sprintf(
				"%s/api/v1/public/appointments/%s/reschedule?new_time=%s",
				baseURL, appointmentID, h,
			),
		})
	}
	return out
}

func renderEmailAprovacaoEncaixe(data emailAprovacaoEncaixeData) (string, error) {
	const tpl = `<!DOCTYPE html>
<html lang="pt-BR">
<head><meta charset="UTF-8"><title>Encaixe pendente</title></head>
<body style="font-family:Arial,sans-serif;color:#222;line-height:1.5;">
  <h2>Olá, {{.ProfissionalNome}}!</h2>
  <p><strong>Cliente:</strong> {{.ClienteNome}}<br>
     <strong>Serviço:</strong> {{.ServicoNome}}<br>
     <strong>Horário solicitado:</strong> {{.HorarioSolicitado}}</p>
  <p style="background:#fff3cd;padding:12px;border-radius:6px;">
    Este atendimento está livre no início, mas vai invadir o início do seu próximo cliente em
    <strong>{{.MinutosInvadidos}} minutos</strong>.
  </p>
  <p>
    <form method="POST" action="{{.ApproveURL}}" style="display:inline;">
      <button type="submit" style="background:#28a745;color:#fff;border:none;padding:10px 16px;border-radius:6px;cursor:pointer;">
        Aceitar Agendamento
      </button>
    </form>
  </p>
  {{if .Alternativas}}
  <h3>Horários alternativos (sem sobreposição)</h3>
  <p>Se preferir não fazer o encaixe, ofereça um destes horários à cliente:</p>
  <ul style="list-style:none;padding:0;">
  {{range .Alternativas}}
    <li style="margin-bottom:8px;">
      <form method="POST" action="{{.URL}}" style="display:inline;">
        <button type="submit" style="background:#007bff;color:#fff;border:none;padding:8px 14px;border-radius:6px;cursor:pointer;">
          Oferecer {{.Horario}}
        </button>
      </form>
    </li>
  {{end}}
  </ul>
  {{else}}
  <p><em>Não há outros horários totalmente livres neste dia para este procedimento.</em></p>
  {{end}}
</body>
</html>`

	t, err := template.New("encaixe").Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *AgendaService) notificarClienteReagendamento(ag *agendamentoPendente, novoInicio time.Time) {
	log.Printf(
		"[notificação cliente] %s (%s): horário alterado para %s — futuro WhatsApp",
		ag.ClienteNome,
		ag.ClienteTelefone,
		novoInicio.Format("02/01/2006 15:04"),
	)
}

// LogMailer registra e-mails no stdout (desenvolvimento / fallback).
type LogMailer struct{}

func (LogMailer) SendHTML(_ context.Context, to, subject, htmlBody string) error {
	log.Printf("[email] para=%s assunto=%q corpo=%d bytes\n%s", to, subject, len(htmlBody), htmlBody)
	return nil
}

// SMTPMailer envia e-mails via SMTP configurado por variáveis de ambiente.
type SMTPMailer struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func NewSMTPMailerFromEnv() Mailer {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return LogMailer{}
	}
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = strings.TrimSpace(os.Getenv("SMTP_USER"))
	}
	return &SMTPMailer{
		Host:     host,
		Port:     port,
		User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
		Password: strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		From:     from,
	}
}

func (m *SMTPMailer) SendHTML(_ context.Context, to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%s", m.Host, m.Port)
	msg := buildMIMEMessage(m.From, to, subject, htmlBody)

	var auth smtp.Auth
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Password, m.Host)
	}
	return smtp.SendMail(addr, auth, m.From, []string{to}, []byte(msg))
}

func buildMIMEMessage(from, to, subject, htmlBody string) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(htmlBody)
	return buf.String()
}
