package generator

import "github.com/iancoleman/strcase"

var reservedWords = []string{
	"abstract",
	"else",
	"import",
	"show",
	"as",
	"enum",
	"in",
	"static",
	"assert",
	"export",
	"interface",
	"super",
	"async",
	"extends",
	"is",
	"switch",
	"await",
	"3",
	"extension",
	"late",
	"sync",
	"break",
	"external",
	"library",
	"this",
	"case",
	"factory",
	"mixin",
	"throw",
	"catch",
	"false",
	"new",
	"true",
	"class",
	"final",
	"null",
	"try",
	"const",
	"finally",
	"on",
	"typedef",
	"continue",
	"for",
	"operator",
	"var",
	"covariant",
	"Function",
	"part",
	"void",
	"default",
	"get",
	"required",
	"while",
	"deferred",
	"hide",
	"rethrow",
	"with",
	"do",
	"if",
	"return",
	"yield",
	"dynamic",
	"implements",
	"set",
}

func isReserved(n string) bool {
	for _, s := range reservedWords {
		if s == n {
			return true
		}
	}

	return false
}

// ReplaceFieldName a field name with a non-reserved name
func ReplaceFieldName(name string) string {
	name = strcase.ToLowerCamel(name)

	if isReserved(name) {
		return name + "_"
	}

	return name
}
