package service

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeCreate_ValidatesFields(t *testing.T) {
	t.Parallel()

	future := time.Now().UTC().Add(2 * time.Hour)
	corpo := "Sistema indisponível domingo."
	est := []string{"11111111-1111-4111-8111-111111111111"}

	titulo, body, sev, aud, ids, exp, err := normalizeCreate(CreateAvisoInput{
		Titulo:             " Manutenção ",
		Corpo:              &corpo,
		Severidade:         "warning",
		AudienceTipo:       "ESTABELECIMENTOS",
		EstabelecimentoIDs: est,
		ExpiresAt:          &future,
	})
	if err != nil {
		t.Fatalf("normalizeCreate: %v", err)
	}
	if titulo != "Manutenção" || body == nil || *body != corpo {
		t.Fatalf("titulo/corpo inesperados: %q %#v", titulo, body)
	}
	if sev != AvisoSeveridadeWarning || aud != AvisoAudienceEstabelecimentos {
		t.Fatalf("sev/aud = %s/%s", sev, aud)
	}
	if len(ids) != 1 || exp == nil {
		t.Fatalf("ids/exp inesperados: %#v %#v", ids, exp)
	}
}

func TestNormalizeCreate_RejectsInvalid(t *testing.T) {
	t.Parallel()

	past := time.Now().UTC().Add(-time.Hour)
	longCorpo := strings.Repeat("a", avisoCorpoMaxLen+1)

	casos := []struct {
		nome string
		in   CreateAvisoInput
		want error
	}{
		{nome: "titulo vazio", in: CreateAvisoInput{Titulo: "  ", AudienceTipo: AvisoAudienceAllTenants}, want: ErrAvisoTituloInvalido},
		{nome: "corpo longo", in: CreateAvisoInput{Titulo: "ok", Corpo: &longCorpo, AudienceTipo: AvisoAudienceAllTenants}, want: ErrAvisoCorpoInvalido},
		{nome: "severidade", in: CreateAvisoInput{Titulo: "ok", Severidade: "FATAL", AudienceTipo: AvisoAudienceAllTenants}, want: ErrAvisoSeveridadeInvalida},
		{nome: "audience", in: CreateAvisoInput{Titulo: "ok", AudienceTipo: "TODOS"}, want: ErrAvisoAudienceInvalida},
		{nome: "est vazio", in: CreateAvisoInput{Titulo: "ok", AudienceTipo: AvisoAudienceEstabelecimentos}, want: ErrAvisoEstabelecimentosFaltando},
		{nome: "expires passado", in: CreateAvisoInput{Titulo: "ok", AudienceTipo: AvisoAudienceAllTenants, ExpiresAt: &past}, want: ErrAvisoExpiresAtInvalido},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nome, func(t *testing.T) {
			t.Parallel()
			_, _, _, _, _, _, err := normalizeCreate(caso.in)
			if err != caso.want {
				t.Fatalf("erro=%v; want %v", err, caso.want)
			}
		})
	}
}

func TestNormalizeCreate_IgnoraEstabelecimentosForaDeAudience(t *testing.T) {
	t.Parallel()

	_, _, _, aud, ids, _, err := normalizeCreate(CreateAvisoInput{
		Titulo:             "Broadcast",
		AudienceTipo:       AvisoAudienceAllTenants,
		EstabelecimentoIDs: []string{"11111111-1111-4111-8111-111111111111"},
	})
	if err != nil {
		t.Fatalf("normalizeCreate: %v", err)
	}
	if aud != AvisoAudienceAllTenants || len(ids) != 0 {
		t.Fatalf("aud=%s ids=%v", aud, ids)
	}
}

func TestInboxVisibilitySQL_ContainsAudienceRules(t *testing.T) {
	t.Parallel()

	sql := inboxVisibilitySQL("a", "$2", "$3")
	for _, snippet := range []string{
		"a.ativo = TRUE",
		"SUPER_ADMINS",
		"ALL_TENANTS",
		"ESTABELECIMENTOS",
		"aviso_estabelecimentos",
	} {
		if !strings.Contains(sql, snippet) {
			t.Fatalf("SQL sem %q:\n%s", snippet, sql)
		}
	}
}
