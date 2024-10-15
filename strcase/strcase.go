package strcase

import (
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/panics"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// PartsSep seprates major parts in the string vs. a single `_` which separates words. It is three underscores.
const PartsSep = "___"

var acronyms = []string{
	"API",
	"CSS",
	"DAL",
	"DB",
	"HTML",
	"HTTP",
	"HTTPS",
	"ID",
	"JS",
	"JSON",
	"JWT",
	"PEM", // Private Key format
	"RSA",
	"SHA",
	"SSH",
	"SQL",
	"SQL",
	"TS",
	"UI",
	"URI",
	"URL",
	"USA",
	"UUID",
	"XML",
	"YAML",
	"YML",
}

var acronymsLookup map[string]bool

func init() {
	// Note: Not sure if this even works. Our own Acronyms work better
	// for _, v := range acronyms {
	// 	strcase.ConfigureAcronym(v, strings.ToLower(v))
	// }
	acronymsLookup = map[string]bool{}
	for _, v := range acronyms {
		acronymsLookup[v] = true
	}
}

func ToPascal(s string) string {
	// strcase.ToCamel is actually PascalCase and ToLowerCamel is camelCase
	// Conver to snake case so we can distinguish between words
	s = strcase.ToSnake(s)
	fams := strings.Split(s, PartsSep)
	for i := range fams {
		words := strings.Split(fams[i], "_")
		for i := range words {
			words[i] = ToPascalWord(words[i])
		}
		fams[i] = strings.Join(words, "")
	}
	return strings.Join(fams, "_")
}

func ToCamel(s string) string {
	s = strcase.ToSnake(s)
	// fams are the namespaces
	fams := strings.Split(s, PartsSep)
	for i := range fams {
		words := strings.Split(fams[i], "_")
		for j := range words {
			word := words[j]
			// first word is always lower case
			// rest of the words are pascal case (uppercase if they are acronyms)
			if j == 0 {
				word = strcase.ToLowerCamel(word)
			} else {
				word = ToPascalWord(word)
				word = UpperizeAcronymWord(word)
			}
			words[j] = word
		}
		fams[i] = strings.Join(words, "")
	}

	// Join the namespaces with a single `_`
	return strings.Join(fams, "_")
}

// ToSnake expects a single word and  - converts a string to snake_case
func ToSnake(s string) string {
	for _, acr := range acronyms {
		// If there is an uppercase acronym XYZ in here, make it title case since strcase thinks x, y, and z in XYX are seperate words
		replace := strings.ToUpper(acr)
		if strings.Contains(s, replace) {
			with := cases.Title(language.English).String(strings.ToLower(acr))
			s = strings.ReplaceAll(s, replace, with)
		}
	}
	// if there are three `_`, we should maintain them as two `_` in snake case
	parts := strings.Split(s, PartsSep)
	for i := range parts {
		parts[i] = strcase.ToSnake(parts[i])
	}
	return strings.Join(parts, "__")
}

func ToKebab(s string) string {
	// simply convert to snake case and then replace `_` with `-`
	s = ToSnake(s)
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

func ToTitle(s string) string {
	parts := strings.Split(s, PartsSep)
	for i := range parts {
		words := strings.Split(parts[i], "_")
		for j := range words {
			words[j] = UpperizeAcronymWord(words[j])
			words[j] = capitalizeFirstLetterInWord(words[j])
		}
		parts[i] = strings.Join(words, " ")
	}
	return strings.Join(parts, " - ")
}

/* * * * * * *
* Helper Functions
* * * * * * */

func IsAcronym(s string) bool {
	return acronymsLookup[strings.ToUpper(s)]
}

// UpperizeAcronyms - upperize all acronyms in the string. It assumes that the words are split by the separator
func UpperizeAcronyms(s string, sep string) string {
	parts := strings.Split(s, sep)
	for i := range parts {
		parts[i] = UpperizeAcronymWord(parts[i])
	}
	return strings.Join(parts, sep)
}

// UpperizeAcronymWord makes the word uppercase if the word is an acronym.
// It handles the case where the word is a plural of an acronym.
func UpperizeAcronymWord(word string) string {

	// Check if the entire word is an acronym
	// e.g. ID, API, HTTP, HTTPS, etc.
	if IsAcronym(word) {
		return strings.ToUpper(word)
	}
	// Check if the word is a plural of an acronym
	// e.g. ids, apis, https, etc.
	if len(word) > 1 && word[len(word)-1] == 's' && IsAcronym(word[:len(word)-1]) {
		// Upperize all the characters except the last one
		return strings.ToUpper(word[:len(word)-1]) + "s"
	}

	return word

}

func ToPascalWord(w string) string {
	// do no
	w = strcase.ToCamel(w)
	w = UpperizeAcronymWord(w)
	return w
}

func capitalizeFirstLetterInWord(word string) string {
	if word == "" {
		return word
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

var pluralOverrides = map[string]string{
	"sheep":   "sheep",
	"fish":    "fish",
	"address": "addresses",
	"process": "processes",
}

// Pluralize should only be called on a snake cased string
func Pluralize(s string) string {
	if s == "" {
		return s
	}
	snakeS := ToSnake(s)
	if snakeS != s {
		log.WarnWithoutCtx("library strcase: Pluralize should only be called on a snake cased string. Got [%s]", s)
	}
	// We only need to pluralize the last word
	words := strings.Split(s, "_")
	lastWord := words[len(words)-1]
	if ans, exists := pluralOverrides[lastWord]; exists {
		words[len(words)-1] = ans
		return strings.Join(words, "_")
	}

	// If ends with `s`, don't know what to do
	if s[len(s)-1] == 's' {
		log.WarnWithoutCtx("library strcase: couldn't determine plural form of [%s]", s)
	}
	// If ends with `y`, change `y` to `ies` e.g. company -> companies
	if s[len(s)-1] == 'y' {
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}

var singularOverrides = map[string]string{
	"sheep":     "sheep",
	"fish":      "fish",
	"addresses": "address",
	"processes": "process",
}

func Singularize(s string) string {
	if s == "" {
		return s
	}
	if ans, exists := singularOverrides[strings.ToLower(s)]; exists {
		return ans
	}
	if s[len(s)-1] == 's' {
		return s[:len(s)-1]
	}
	panics.P("library strcase: couldn't determine singular form of '%s'", s)
	return s
}
