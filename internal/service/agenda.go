package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type OrigemAgendamento string

const (
	OrigemInterno OrigemAgendamento = "INTERNO"
	OrigemExterno OrigemAgendamento = "EXTERNO"
)

var (
	ErrProfissionalNaoEncontrado = errors.New("profissional não encontrado ou inativo")
	ErrServicoNaoEncontrado      = errors.New("serviço não encontrado ou inativo")
	ErrAdicionalInvalido         = errors.New("um ou mais adicionais não pertencem ao serviço informado")
	ErrColisaoHorario            = errors.New("A profissional selecionada já possui um atendimento agendado neste intervalo de horário")
	ErrOrigemAgendamentoInvalida = errors.New("origem do agendamento inválida: use INTERNO ou EXTERNO")
)

type AgendaService struct {
	db      *sqlx.DB
	baseURL string
	mailer  Mailer
}

func NewAgendaService(db *sqlx.DB, opts ...AgendaOptions) *AgendaService {
	svc := &AgendaService{db: db}
	if len(opts) > 0 {
		svc.baseURL = strings.TrimRight(opts[0].BaseURL, "/")
		svc.mailer = opts[0].Mailer
	}
	return svc
}

type servicoBase struct {
	ID                 string  `db:"id"`
	Nome               string  `db:"nome"`
	PrecoBase          float64 `db:"preco_base"`
	DuracaoBaseMinutos int     `db:"duracao_base_minutos"`
}

type totaisAdicionais struct {
	DuracaoMinutos int     `db:"duracao_minutos"`
	Preco          float64 `db:"preco"`
	Quantidade     int     `db:"quantidade"`
}

// CriarAgendamento resolve ou cadastra a cliente, calcula duração/preço,
// valida colisão de horários e persiste o agendamento atomicamente.
func (s *AgendaService) CriarAgendamento(
	ctx context.Context,
	estabelecimentoID, clienteNome, clienteTelefone, profissionalID, servicoID string,
	adicionaisIDs []string,
	inicio time.Time,
	origem OrigemAgendamento,
) (ResultadoAgendamento, error) {
	var resultado ResultadoAgendamento

	if err := validarOrigemAgendamento(origem); err != nil {
		return resultado, err
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return resultado, fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	clienteID, err := s.resolverOuCadastrarCliente(ctx, tx, estabelecimentoID, clienteNome, clienteTelefone)
	if err != nil {
		return resultado, err
	}

	if err := s.validarProfissional(ctx, tx, estabelecimentoID, profissionalID); err != nil {
		return resultado, err
	}

	servico, err := s.buscarServicoBase(ctx, tx, estabelecimentoID, servicoID)
	if err != nil {
		return resultado, err
	}

	atendimento := struct {
		duracaoMinutos int
		precoTotal     float64
	}{
		duracaoMinutos: servico.DuracaoBaseMinutos,
		precoTotal:     servico.PrecoBase,
	}

	if len(adicionaisIDs) > 0 {
		totais, err := s.somarAdicionais(ctx, tx, estabelecimentoID, servicoID, adicionaisIDs)
		if err != nil {
			return resultado, err
		}
		if totais.Quantidade != len(adicionaisIDs) {
			return resultado, ErrAdicionalInvalido
		}
		atendimento.duracaoMinutos += totais.DuracaoMinutos
		atendimento.precoTotal = roundMoney(atendimento.precoTotal + totais.Preco)
	}

	fim := inicio.Add(time.Duration(atendimento.duracaoMinutos) * time.Minute)

	tipoColisao, detalhe, err := s.avaliarColisaoHorario(ctx, tx, estabelecimentoID, profissionalID, inicio, fim)
	if err != nil {
		return resultado, err
	}

	status := "AGENDADO"
	minutosInvadidos := 0
	switch tipoColisao {
	case colisaoTotalInicio:
		return resultado, ErrColisaoHorario
	case colisaoSobreposicaoTermino:
		status = "EM_APROVACAO"
		if detalhe != nil {
			minutosInvadidos = detalhe.MinutosInvadidos
		}
	}

	viaClube, err := s.clientePossuiAssinaturaComCreditos(ctx, tx, estabelecimentoID, clienteTelefone)
	if err != nil {
		return resultado, err
	}

	agendamentoID, err := s.inserirAgendamento(
		ctx, tx,
		estabelecimentoID,
		clienteID,
		profissionalID, servicoID,
		inicio, fim,
		status,
		viaClube,
		origem,
	)
	if err != nil {
		return resultado, err
	}

	if len(adicionaisIDs) > 0 {
		if err := s.inserirAdicionaisAgendamento(ctx, tx, agendamentoID, adicionaisIDs); err != nil {
			return resultado, err
		}
	}

	if err := tx.Commit(); err != nil {
		return resultado, fmt.Errorf("confirmar transação: %w", err)
	}

	resultado = ResultadoAgendamento{
		ID:               agendamentoID,
		Status:           status,
		MinutosInvadidos: minutosInvadidos,
	}

	if status == "EM_APROVACAO" {
		go s.dispararEmailAprovacaoEncaixe(agendamentoID, minutosInvadidos)
	}

	go s.dispararLembreteWhatsAppAgendamento(agendamentoID)

	return resultado, nil
}

func validarOrigemAgendamento(origem OrigemAgendamento) error {
	switch origem {
	case OrigemInterno, OrigemExterno:
		return nil
	default:
		return ErrOrigemAgendamentoInvalida
	}
}

func (s *AgendaService) resolverOuCadastrarCliente(
	ctx context.Context,
	tx *sqlx.Tx,
	estabelecimentoID, nome, telefone string,
) (string, error) {
	const upsert = `
INSERT INTO clientes (estabelecimento_id, nome, telefone)
VALUES ($1, $2, $3)
ON CONFLICT (estabelecimento_id, telefone) DO UPDATE SET nome = clientes.nome
RETURNING id
`
	var clienteID string
	if err := tx.GetContext(ctx, &clienteID, upsert, estabelecimentoID, nome, telefone); err != nil {
		return "", fmt.Errorf("resolver cliente: %w", err)
	}

	return clienteID, nil
}

func (s *AgendaService) buscarServicoBase(
	ctx context.Context,
	tx *sqlx.Tx,
	estabelecimentoID, servicoID string,
) (*servicoBase, error) {
	const query = `
SELECT id, nome, preco_base, duracao_base_minutos
FROM servicos
WHERE id = $1 AND estabelecimento_id = $2 AND ativo = TRUE
`
	var servico servicoBase
	if err := tx.GetContext(ctx, &servico, query, servicoID, estabelecimentoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrServicoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar serviço base: %w", err)
	}

	return &servico, nil
}

func (s *AgendaService) somarAdicionais(
	ctx context.Context,
	tx *sqlx.Tx,
	estabelecimentoID, servicoID string,
	adicionaisIDs []string,
) (*totaisAdicionais, error) {
	const query = `
SELECT
    COALESCE(SUM(sa.duracao_adicional_minutos), 0)::INTEGER AS duracao_minutos,
    COALESCE(SUM(sa.preco_adicional), 0) AS preco,
    COUNT(sa.id)::INTEGER AS quantidade
FROM servico_adicionais sa
INNER JOIN servicos s ON s.id = sa.servico_id
WHERE sa.servico_id = $1
  AND s.estabelecimento_id = $2
  AND sa.id = ANY($3)
`
	var totais totaisAdicionais
	if err := tx.GetContext(ctx, &totais, query, servicoID, estabelecimentoID, pq.Array(adicionaisIDs)); err != nil {
		return nil, fmt.Errorf("somar adicionais do serviço: %w", err)
	}

	return &totais, nil
}

func (s *AgendaService) inserirAgendamento(
	ctx context.Context,
	tx *sqlx.Tx,
	estabelecimentoID, clienteID, profissionalID, servicoID string,
	inicio, fim time.Time,
	status string,
	viaClube bool,
	origem OrigemAgendamento,
) (string, error) {
	const insert = `
INSERT INTO agendamentos (
    estabelecimento_id,
    cliente_id,
    profissional_id,
    servico_id,
    data_hora_inicio,
    data_hora_fim,
    status,
    via_clube_assinatura,
    origem_agendamento
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id
`
	var agendamentoID string
	if err := tx.GetContext(
		ctx,
		&agendamentoID,
		insert,
		estabelecimentoID,
		clienteID,
		profissionalID,
		servicoID,
		inicio,
		fim,
		status,
		viaClube,
		string(origem),
	); err != nil {
		return "", fmt.Errorf("criar agendamento: %w", err)
	}

	return agendamentoID, nil
}

func (s *AgendaService) inserirAdicionaisAgendamento(
	ctx context.Context,
	tx *sqlx.Tx,
	agendamentoID string,
	adicionaisIDs []string,
) error {
	const insert = `
INSERT INTO agendamento_adicionais (agendamento_id, adicional_id)
VALUES ($1, $2)
`
	for _, adicionalID := range adicionaisIDs {
		if _, err := tx.ExecContext(ctx, insert, agendamentoID, adicionalID); err != nil {
			return fmt.Errorf("vincular adicional %s ao agendamento: %w", adicionalID, err)
		}
	}

	return nil
}

func (s *AgendaService) validarProfissional(ctx context.Context, tx *sqlx.Tx, estabelecimentoID, profissionalID string) error {
	const query = `
SELECT id FROM profissionais
WHERE id = $1 AND estabelecimento_id = $2 AND ativo = TRUE
`
	var id string
	if err := tx.GetContext(ctx, &id, query, profissionalID, estabelecimentoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProfissionalNaoEncontrado
		}
		return fmt.Errorf("validar profissional: %w", err)
	}

	return nil
}

func (s *AgendaService) clientePossuiAssinaturaComCreditos(
	ctx context.Context,
	tx *sqlx.Tx,
	estabelecimentoID, telefone string,
) (bool, error) {
	const query = `
SELECT ca.id
FROM clientes_assinantes ca
INNER JOIN planos_assinatura pa ON pa.id = ca.plano_id
WHERE ca.estabelecimento_id = $1
  AND ca.cliente_telefone = $2
  AND ca.status = 'ATIVO'
  AND ca.visitas_restantes > 0
  AND pa.ativo = TRUE
LIMIT 1
FOR SHARE
`
	var id string
	err := tx.GetContext(ctx, &id, query, estabelecimentoID, telefone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("verificar assinatura do cliente: %w", err)
	}

	return true, nil
}
