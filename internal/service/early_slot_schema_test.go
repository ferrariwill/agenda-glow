package service

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// O sqlmock devolve as colunas que a query pedir, então ele nunca reprova uma
// referência a coluna inexistente. Este teste fecha essa lacuna lendo as
// migrações: se uma query da antecipação voltar a ler uma coluna que o schema
// real já removeu de `agendamentos`, o build de teste falha aqui.
func TestEarlySlotQueriesDoNotReadDroppedAgendamentoColumns(t *testing.T) {
	dropped := droppedColumns(t, "agendamentos")
	if len(dropped) == 0 {
		t.Fatal("nenhuma coluna removida encontrada nas migrações — o guard perderia o sentido")
	}
	// Sanidade: as colunas que motivaram a reprovação precisam estar no conjunto.
	for _, esperada := range []string{"cliente_nome", "cliente_telefone"} {
		if !contains(dropped, esperada) {
			t.Fatalf("migração 000004 remove %q de agendamentos, mas o parser não detectou (detectadas: %v)",
				esperada, dropped)
		}
	}

	for _, query := range sqlLiterals(t, "early_slot.go") {
		for _, coluna := range dropped {
			// `a` é o alias de `agendamentos` em todas as queries do arquivo.
			for _, referencia := range []string{"a." + coluna, "agendamentos." + coluna} {
				if strings.Contains(query, referencia) {
					t.Errorf("query da antecipação lê %q, coluna removida de agendamentos pela migração;"+
						" use o JOIN em clientes (c.nome / c.telefone)", referencia)
				}
			}
		}
	}
}

// sqlLiterals devolve as strings do arquivo — só o código, sem comentários, que
// naturalmente citam os nomes de coluna ao explicar a própria regra.
func sqlLiterals(t *testing.T, file string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parsear %s: %v", file, err)
	}
	var literals []string
	ast.Inspect(parsed, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if ok && lit.Kind == token.STRING {
			if value, err := strconv.Unquote(lit.Value); err == nil {
				literals = append(literals, value)
			}
		}
		return true
	})
	return literals
}

// O aceite move o agendamento para o slot da rodada, então precisa confirmar
// que o candidato ainda pertence ao profissional da rodada e que o novo horário
// cabe no expediente — caso contrário uma troca de profissional após a oferta
// jogaria o atendimento na agenda de outra pessoa.
func TestAcceptLookupBindsCandidateToRoundProfessionalAndSchedule(t *testing.T) {
	exigidos := map[string]string{
		"a.profissional_id = r.profissional_id": "vínculo com o profissional da rodada",
		"expedientes_profissionais":             "revalidação de expediente",
		"ep.inicio_almoco":                      "revalidação de almoço",
		"JOIN clientes c":                       "identidade do cliente vinda de clientes",
	}
	for trecho, motivo := range exigidos {
		if !strings.Contains(earlySlotAcceptLookupSQL, trecho) {
			t.Errorf("query de aceite sem %s (trecho ausente: %q)", motivo, trecho)
		}
	}
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

var (
	alterTableRe = regexp.MustCompile(`(?is)alter\s+table\s+(?:if\s+exists\s+)?([a-z_][a-z0-9_]*)`)
	dropColumnRe = regexp.MustCompile(`(?i)drop\s+column\s+(?:if\s+exists\s+)?"?([a-z_][a-z0-9_]*)"?`)
	addColumnRe  = regexp.MustCompile(`(?i)add\s+column\s+(?:if\s+not\s+exists\s+)?"?([a-z_][a-z0-9_]*)"?`)
)

// droppedColumns reproduz o estado final do schema para uma tabela: uma coluna
// removida e depois recriada por migração posterior não conta como removida.
func droppedColumns(t *testing.T, table string) []string {
	t.Helper()
	arquivos, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.up.sql"))
	if err != nil {
		t.Fatalf("listar migrações: %v", err)
	}
	if len(arquivos) == 0 {
		t.Fatal("nenhuma migração encontrada")
	}
	sort.Strings(arquivos)

	removidas := map[string]bool{}
	for _, arquivo := range arquivos {
		conteudo, err := os.ReadFile(arquivo)
		if err != nil {
			t.Fatalf("ler %s: %v", arquivo, err)
		}
		for _, statement := range strings.Split(string(conteudo), ";") {
			alvo := alterTableRe.FindStringSubmatch(statement)
			if alvo == nil || !strings.EqualFold(alvo[1], table) {
				continue
			}
			for _, m := range dropColumnRe.FindAllStringSubmatch(statement, -1) {
				removidas[strings.ToLower(m[1])] = true
			}
			for _, m := range addColumnRe.FindAllStringSubmatch(statement, -1) {
				delete(removidas, strings.ToLower(m[1]))
			}
		}
	}

	resultado := make([]string, 0, len(removidas))
	for coluna := range removidas {
		resultado = append(resultado, coluna)
	}
	sort.Strings(resultado)
	fmt.Fprintf(os.Stderr, "colunas removidas de %s: %v\n", table, resultado)
	return resultado
}
