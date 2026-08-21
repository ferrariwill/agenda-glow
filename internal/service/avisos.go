package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	AvisoSeveridadeInfo     = "INFO"
	AvisoSeveridadeWarning  = "WARNING"
	AvisoSeveridadeCritical = "CRITICAL"

	AvisoAudienceSuperAdmins      = "SUPER_ADMINS"
	AvisoAudienceAllTenants       = "ALL_TENANTS"
	AvisoAudienceEstabelecimentos = "ESTABELECIMENTOS"

	avisoTituloMaxLen = 200
	avisoCorpoMaxLen  = 2000
	avisoListDefault  = 50
	avisoListMax      = 100
	avisoInboxDefault = 50
)

var (
	ErrAvisoTituloInvalido           = errors.New("invalid_titulo")
	ErrAvisoCorpoInvalido            = errors.New("invalid_corpo")
	ErrAvisoSeveridadeInvalida       = errors.New("invalid_severidade")
	ErrAvisoAudienceInvalida         = errors.New("invalid_audience")
	ErrAvisoEstabelecimentosFaltando = errors.New("missing_estabelecimento_ids")
	ErrAvisoExpiresAtInvalido        = errors.New("invalid_expires_at")
	ErrAvisoNaoEncontrado            = errors.New("aviso_not_found")
)

// AvisoService gerencia broadcasts globais e o inbox autenticado.
type AvisoService struct {
	db *sqlx.DB
}

func NewAvisoService(db *sqlx.DB) *AvisoService {
	return &AvisoService{db: db}
}

// Aviso é o shape de gestão (admin) de um aviso global.
type Aviso struct {
	ID                 string     `json:"id"`
	Titulo             string     `json:"titulo"`
	Corpo              *string    `json:"corpo"`
	Severidade         string     `json:"severidade"`
	AudienceTipo       string     `json:"audience_tipo"`
	EstabelecimentoIDs []string   `json:"estabelecimento_ids"`
	Ativo              bool       `json:"ativo"`
	CreatedBy          string     `json:"created_by"`
	CriadoEm           time.Time  `json:"criado_em"`
	ExpiresAt          *time.Time `json:"expires_at"`
}

// CreateAvisoInput entrada de criação.
type CreateAvisoInput struct {
	Titulo             string
	Corpo              *string
	Severidade         string
	AudienceTipo       string
	EstabelecimentoIDs []string
	ExpiresAt          *time.Time
	CreatedBy          string
}

// UpdateAvisoInput atualização parcial (audience/junction imutáveis neste MVP).
type UpdateAvisoInput struct {
	Titulo      *string
	Corpo       *string
	Severidade  *string
	Ativo       *bool
	ExpiresAt   *time.Time
	ClearExpiry bool // true quando expires_at veio explicitamente como null
}

// InboxNotification shape alinhado ao front (NotificationBell).
type InboxNotification struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      *string    `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at"`
}

// InboxResult lista paginada + unread total visível.
type InboxResult struct {
	Items       []InboxNotification `json:"items"`
	UnreadCount int                 `json:"unread_count"`
}

type avisoRow struct {
	ID           string         `db:"id"`
	Titulo       string         `db:"titulo"`
	Corpo        sql.NullString `db:"corpo"`
	Severidade   string         `db:"severidade"`
	AudienceTipo string         `db:"audience_tipo"`
	Ativo        bool           `db:"ativo"`
	CreatedBy    string         `db:"created_by"`
	CriadoEm     time.Time      `db:"criado_em"`
	ExpiresAt    sql.NullTime   `db:"expires_at"`
}

type inboxRow struct {
	ID        string         `db:"id"`
	Title     string         `db:"title"`
	Body      sql.NullString `db:"body"`
	CreatedAt time.Time      `db:"created_at"`
	ReadAt    sql.NullTime   `db:"read_at"`
}

// Create cria um aviso global e, se necessário, a junction de estabelecimentos.
func (s *AvisoService) Create(ctx context.Context, in CreateAvisoInput) (*Aviso, error) {
	titulo, corpo, severidade, audience, estIDs, expiresAt, err := normalizeCreate(in)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.CreatedBy) == "" {
		return nil, fmt.Errorf("created_by obrigatório")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if audience == AvisoAudienceEstabelecimentos {
		if err := assertEstabelecimentosExistem(ctx, tx, estIDs); err != nil {
			return nil, err
		}
	}

	const insert = `
INSERT INTO avisos_globais (titulo, corpo, severidade, audience_tipo, created_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, titulo, corpo, severidade, audience_tipo, ativo, created_by, criado_em, expires_at
`
	var row avisoRow
	if err := tx.GetContext(ctx, &row, insert, titulo, corpo, severidade, audience, in.CreatedBy, expiresAt); err != nil {
		return nil, fmt.Errorf("criar aviso: %w", err)
	}

	if audience == AvisoAudienceEstabelecimentos {
		for _, estID := range estIDs {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO aviso_estabelecimentos (aviso_id, estabelecimento_id)
VALUES ($1, $2)
`, row.ID, estID); err != nil {
				return nil, fmt.Errorf("vincular estabelecimento: %w", err)
			}
		}
	} else {
		estIDs = []string{}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit aviso: %w", err)
	}

	out := rowToAviso(row)
	out.EstabelecimentoIDs = estIDs
	return out, nil
}

// ListAdmin lista avisos para gestão (inclui inativos/expirados).
func (s *AvisoService) ListAdmin(ctx context.Context, ativo *bool, limit, offset int) ([]Aviso, int, error) {
	limit, offset = clampListPaging(limit, offset)

	args := []any{}
	where := "TRUE"
	if ativo != nil {
		args = append(args, *ativo)
		where = fmt.Sprintf("ativo = $%d", len(args))
	}

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM avisos_globais WHERE %s`, where)
	var total int
	if err := s.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, fmt.Errorf("contar avisos: %w", err)
	}

	args = append(args, limit, offset)
	listQ := fmt.Sprintf(`
SELECT id, titulo, corpo, severidade, audience_tipo, ativo, created_by, criado_em, expires_at
FROM avisos_globais
WHERE %s
ORDER BY criado_em DESC
LIMIT $%d OFFSET $%d
`, where, len(args)-1, len(args))

	var rows []avisoRow
	if err := s.db.SelectContext(ctx, &rows, listQ, args...); err != nil {
		return nil, 0, fmt.Errorf("listar avisos: %w", err)
	}

	items := make([]Aviso, 0, len(rows))
	for _, row := range rows {
		aviso := rowToAviso(row)
		ids, err := s.loadEstabelecimentoIDs(ctx, row.ID)
		if err != nil {
			return nil, 0, err
		}
		aviso.EstabelecimentoIDs = ids
		items = append(items, *aviso)
	}
	return items, total, nil
}

// Update atualiza campos mutáveis; audience permanece imutável.
func (s *AvisoService) Update(ctx context.Context, id string, in UpdateAvisoInput) (*Aviso, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrAvisoNaoEncontrado
	}

	current, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}

	titulo := current.Titulo
	if in.Titulo != nil {
		titulo, err = validateTitulo(*in.Titulo)
		if err != nil {
			return nil, err
		}
	}

	corpo := current.Corpo
	if in.Corpo != nil {
		corpo, err = validateCorpo(in.Corpo)
		if err != nil {
			return nil, err
		}
	}

	severidade := current.Severidade
	if in.Severidade != nil {
		severidade, err = validateSeveridade(*in.Severidade)
		if err != nil {
			return nil, err
		}
	}

	ativo := current.Ativo
	if in.Ativo != nil {
		ativo = *in.Ativo
	}

	expiresAt := current.ExpiresAt
	if in.ClearExpiry {
		expiresAt = nil
	} else if in.ExpiresAt != nil {
		if !in.ExpiresAt.After(time.Now().UTC()) {
			return nil, ErrAvisoExpiresAtInvalido
		}
		expiresAt = in.ExpiresAt
	}

	const update = `
UPDATE avisos_globais
SET titulo = $2, corpo = $3, severidade = $4, ativo = $5, expires_at = $6
WHERE id = $1
RETURNING id, titulo, corpo, severidade, audience_tipo, ativo, created_by, criado_em, expires_at
`
	var row avisoRow
	err = s.db.GetContext(ctx, &row, update, id, titulo, corpo, severidade, ativo, expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAvisoNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("atualizar aviso: %w", err)
	}

	out := rowToAviso(row)
	ids, err := s.loadEstabelecimentoIDs(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	out.EstabelecimentoIDs = ids
	return out, nil
}

// ListInbox retorna avisos visíveis ao usuário autenticado.
func (s *AvisoService) ListInbox(
	ctx context.Context,
	userID, role string,
	estabelecimentoID *string,
	limit int,
) (*InboxResult, error) {
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)
	if userID == "" {
		return nil, fmt.Errorf("user_id obrigatório")
	}
	if limit <= 0 {
		limit = avisoInboxDefault
	}
	if limit > avisoListMax {
		limit = avisoListMax
	}

	estID := ""
	if estabelecimentoID != nil {
		estID = strings.TrimSpace(*estabelecimentoID)
	}

	itemsQ := `
SELECT a.id,
       a.titulo AS title,
       a.corpo AS body,
       a.criado_em AS created_at,
       al.read_at
FROM avisos_globais a
LEFT JOIN aviso_leituras al ON al.aviso_id = a.id AND al.user_id = $1
WHERE ` + inboxVisibilitySQL("a", "$2", "$3") + `
ORDER BY a.criado_em DESC
LIMIT $4
`
	var rows []inboxRow
	if err := s.db.SelectContext(ctx, &rows, itemsQ, userID, role, nullIfEmpty(estID), limit); err != nil {
		return nil, fmt.Errorf("listar inbox: %w", err)
	}

	unreadQ := `
SELECT COUNT(*)
FROM avisos_globais a
LEFT JOIN aviso_leituras al ON al.aviso_id = a.id AND al.user_id = $1
WHERE ` + inboxVisibilitySQL("a", "$2", "$3") + `
  AND al.read_at IS NULL
`
	var unread int
	if err := s.db.GetContext(ctx, &unread, unreadQ, userID, role, nullIfEmpty(estID)); err != nil {
		return nil, fmt.Errorf("contar não lidas: %w", err)
	}

	items := make([]InboxNotification, 0, len(rows))
	for _, row := range rows {
		items = append(items, inboxRowToItem(row))
	}
	return &InboxResult{Items: items, UnreadCount: unread}, nil
}

// MarkRead marca um aviso visível como lido (idempotente).
func (s *AvisoService) MarkRead(
	ctx context.Context,
	userID, role, avisoID string,
	estabelecimentoID *string,
) (*InboxNotification, error) {
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)
	avisoID = strings.TrimSpace(avisoID)
	if userID == "" || avisoID == "" {
		return nil, ErrAvisoNaoEncontrado
	}

	estID := ""
	if estabelecimentoID != nil {
		estID = strings.TrimSpace(*estabelecimentoID)
	}

	visibleQ := `
SELECT a.id,
       a.titulo AS title,
       a.corpo AS body,
       a.criado_em AS created_at,
       al.read_at
FROM avisos_globais a
LEFT JOIN aviso_leituras al ON al.aviso_id = a.id AND al.user_id = $1
WHERE a.id = $4
  AND ` + inboxVisibilitySQL("a", "$2", "$3") + `
`
	var row inboxRow
	err := s.db.GetContext(ctx, &row, visibleQ, userID, role, nullIfEmpty(estID), avisoID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAvisoNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("buscar aviso inbox: %w", err)
	}

	const upsert = `
INSERT INTO aviso_leituras (aviso_id, user_id, read_at)
VALUES ($1, $2, NOW())
ON CONFLICT (aviso_id, user_id) DO UPDATE
SET read_at = aviso_leituras.read_at
RETURNING read_at
`
	var readAt time.Time
	if err := s.db.GetContext(ctx, &readAt, upsert, avisoID, userID); err != nil {
		return nil, fmt.Errorf("marcar lida: %w", err)
	}
	row.ReadAt = sql.NullTime{Time: readAt, Valid: true}
	item := inboxRowToItem(row)
	return &item, nil
}

func inboxVisibilitySQL(alias, roleParam, estParam string) string {
	// Cast estabelecimento to uuid on every use (including IS NOT NULL).
	// Postgres cannot infer the type of an untyped NULL/$n in "$n IS NOT NULL"
	// when the only other typed use is inside EXISTS — that yields
	// "could not determine data type of parameter $n" at runtime.
	estUUID := estParam + "::uuid"
	return fmt.Sprintf(`
%s.ativo = TRUE
AND (%s.expires_at IS NULL OR %s.expires_at > NOW())
AND (
  (%s = 'SUPER_ADMIN' AND %s.audience_tipo = 'SUPER_ADMINS')
  OR (%s = 'DONA' AND %s.audience_tipo = 'ALL_TENANTS')
  OR (
    %s = 'DONA'
    AND %s.audience_tipo = 'ESTABELECIMENTOS'
    AND %s IS NOT NULL
    AND EXISTS (
      SELECT 1 FROM aviso_estabelecimentos ae
      WHERE ae.aviso_id = %s.id AND ae.estabelecimento_id = %s
    )
  )
)`,
		alias, alias, alias,
		roleParam, alias,
		roleParam, alias,
		roleParam, alias, estUUID,
		alias, estUUID,
	)
}

func (s *AvisoService) getByID(ctx context.Context, id string) (*Aviso, error) {
	const q = `
SELECT id, titulo, corpo, severidade, audience_tipo, ativo, created_by, criado_em, expires_at
FROM avisos_globais
WHERE id = $1
`
	var row avisoRow
	err := s.db.GetContext(ctx, &row, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAvisoNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("buscar aviso: %w", err)
	}
	out := rowToAviso(row)
	ids, err := s.loadEstabelecimentoIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	out.EstabelecimentoIDs = ids
	return out, nil
}

func (s *AvisoService) loadEstabelecimentoIDs(ctx context.Context, avisoID string) ([]string, error) {
	const q = `
SELECT estabelecimento_id
FROM aviso_estabelecimentos
WHERE aviso_id = $1
ORDER BY estabelecimento_id
`
	var ids []string
	if err := s.db.SelectContext(ctx, &ids, q, avisoID); err != nil {
		return nil, fmt.Errorf("listar estabelecimentos do aviso: %w", err)
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

func assertEstabelecimentosExistem(ctx context.Context, tx *sqlx.Tx, ids []string) error {
	const q = `SELECT COUNT(*) FROM estabelecimentos WHERE id = ANY($1)`
	var count int
	if err := tx.GetContext(ctx, &count, q, pq.Array(ids)); err != nil {
		return fmt.Errorf("validar estabelecimentos: %w", err)
	}
	if count != len(ids) {
		return ErrAvisoEstabelecimentosFaltando
	}
	return nil
}

func normalizeCreate(in CreateAvisoInput) (
	titulo string,
	corpo *string,
	severidade string,
	audience string,
	estIDs []string,
	expiresAt *time.Time,
	err error,
) {
	titulo, err = validateTitulo(in.Titulo)
	if err != nil {
		return
	}
	corpo, err = validateCorpo(in.Corpo)
	if err != nil {
		return
	}
	severidade, err = validateSeveridade(in.Severidade)
	if err != nil {
		return
	}
	audience, err = validateAudience(in.AudienceTipo)
	if err != nil {
		return
	}

	estIDs = uniqueNonEmpty(in.EstabelecimentoIDs)
	if audience == AvisoAudienceEstabelecimentos {
		if len(estIDs) == 0 {
			err = ErrAvisoEstabelecimentosFaltando
			return
		}
	} else {
		estIDs = []string{}
	}

	if in.ExpiresAt != nil {
		if !in.ExpiresAt.After(time.Now().UTC()) {
			err = ErrAvisoExpiresAtInvalido
			return
		}
		expiresAt = in.ExpiresAt
	}
	return
}

func validateTitulo(raw string) (string, error) {
	titulo := strings.TrimSpace(raw)
	if titulo == "" || utf8.RuneCountInString(titulo) > avisoTituloMaxLen {
		return "", ErrAvisoTituloInvalido
	}
	return titulo, nil
}

func validateCorpo(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	corpo := strings.TrimSpace(*raw)
	if corpo == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(corpo) > avisoCorpoMaxLen {
		return nil, ErrAvisoCorpoInvalido
	}
	return &corpo, nil
}

func validateSeveridade(raw string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	if s == "" {
		s = AvisoSeveridadeInfo
	}
	switch s {
	case AvisoSeveridadeInfo, AvisoSeveridadeWarning, AvisoSeveridadeCritical:
		return s, nil
	default:
		return "", ErrAvisoSeveridadeInvalida
	}
}

func validateAudience(raw string) (string, error) {
	a := strings.ToUpper(strings.TrimSpace(raw))
	switch a {
	case AvisoAudienceSuperAdmins, AvisoAudienceAllTenants, AvisoAudienceEstabelecimentos:
		return a, nil
	default:
		return "", ErrAvisoAudienceInvalida
	}
}

func uniqueNonEmpty(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func clampListPaging(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = avisoListDefault
	}
	if limit > avisoListMax {
		limit = avisoListMax
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func rowToAviso(row avisoRow) *Aviso {
	out := &Aviso{
		ID:                 row.ID,
		Titulo:             row.Titulo,
		Severidade:         row.Severidade,
		AudienceTipo:       row.AudienceTipo,
		EstabelecimentoIDs: []string{},
		Ativo:              row.Ativo,
		CreatedBy:          row.CreatedBy,
		CriadoEm:           row.CriadoEm.UTC(),
	}
	if row.Corpo.Valid {
		c := row.Corpo.String
		out.Corpo = &c
	}
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time.UTC()
		out.ExpiresAt = &t
	}
	return out
}

func inboxRowToItem(row inboxRow) InboxNotification {
	item := InboxNotification{
		ID:        row.ID,
		Title:     row.Title,
		CreatedAt: row.CreatedAt.UTC(),
	}
	if row.Body.Valid {
		b := row.Body.String
		item.Body = &b
	}
	if row.ReadAt.Valid {
		t := row.ReadAt.Time.UTC()
		item.ReadAt = &t
	}
	return item
}
