package service

import (
	"regexp"
	"strings"
	"unicode"
)

var multiHyphen = regexp.MustCompile(`-+`)

// NormalizarSlug converte texto livre em slug URL-safe.
// Ex.: "Salão da Cláudia!" → "salao-da-claudia"
func NormalizarSlug(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(input))

	prevHyphen := false
	for _, r := range input {
		r = removerAcento(unicode.ToLower(r))

		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		case r == ' ', r == '_', r == '-', r == '/':
			if !prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}

	slug := multiHyphen.ReplaceAllString(strings.Trim(b.String(), "-"), "-")
	return slug
}

func removerAcento(r rune) rune {
	switch r {
	case 'á', 'à', 'â', 'ã', 'ä', 'å':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'ô', 'õ', 'ö':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ç':
		return 'c'
	case 'ñ':
		return 'n'
	default:
		return r
	}
}
