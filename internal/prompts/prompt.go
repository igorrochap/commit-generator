package prompts

import _ "embed"

//go:embed en.md
var English string

//go:embed pt_br.md
var PtBr string

var registry = map[string]string{
	"en":    English,
	"pt-BR": PtBr,
}

func Get(lang string) (string, bool) {
	p, ok := registry[lang]
	return p, ok
}
