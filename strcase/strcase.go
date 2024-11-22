package strcase

import (
	"errors"
	"regexp"
	"strings"

	"github.com/iancoleman/strcase"
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

// ToTitle expects a string with multiple words separated by `_` and `___` and converts it to a title case
func ToTitle(s string) string {
	// Convert to snake case so we can distinguish between words
	s = strcase.ToSnake(s)
	parts := strings.Split(s, "__")
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

/* * * * * * *
* Pluralization and Singularization
* * * * * * */

func MustPluralize(s string) string {
	pl, err := Pluralize(s)
	panics.IfError(err, "Failed to pluralize the string")
	return pl
}

func Pluralize(s string) (string, error) {

	if s == "" {
		return "", nil
	}

	return toPlural(s)

}

// Map of irregular singular to plural nouns
var irregularNouns = map[string]string{
	"child":  "children",
	"person": "people",
	"man":    "men",
	"woman":  "women",
	"mouse":  "mice",
	"goose":  "geese",
	"ox":     "oxen",
	"foot":   "feet",
	"tooth":  "teeth",
	// Add other irregulars as needed
}

// Words that stay the same in plural
var unchangingNouns = []string{
	"sheep",
	"deer",
	"fish",
	"series",
	"species",
	"aircraft",
	"watercraft",
	"hovercraft",
}

// Checks if a word is an unchanging noun
func isUnchangingNoun(word string) bool {
	lowerWord := strings.ToLower(word)
	for _, noun := range unchangingNouns {
		if lowerWord == noun {
			return true
		}
	}
	return false
}

// Main pluralization function
func toPlural(word string) (string, error) {
	if word == "" {
		return "", errors.New("input word cannot be empty")
	}

	// Split the word into parts by underscores
	parts := strings.Split(word, "_")
	if len(parts) == 0 {
		return word, nil
	}

	// Pluralize only the last part
	lastPart := parts[len(parts)-1]
	pluralizedLastPart, err := pluralizeWord(lastPart)
	if err != nil {
		return "", err
	}

	// Replace the last part with its pluralized version
	parts[len(parts)-1] = pluralizedLastPart

	// Rejoin the parts with underscores
	return strings.Join(parts, "_"), nil
}

// Pluralizes a single word according to English pluralization rules
func pluralizeWord(word string) (string, error) {
	if word == "" {
		return "", errors.New("word to pluralize cannot be empty")
	}

	lowerWord := strings.ToLower(word)

	// Check if word is unchanging in plural
	if isUnchangingNoun(lowerWord) {
		return word, nil
	}

	// Check if word is an irregular noun
	if plural, ok := irregularNouns[lowerWord]; ok {
		// Preserve capitalization
		if isCapitalized(word) {
			return capitalize(plural), nil
		}
		return plural, nil
	}

	// Apply general pluralization rules
	switch {
	// Words ending with 's', 'x', 'z', 'ch', 'sh', 'ss' => add 'es'
	case regexp.MustCompile(`(?i)(s|x|z|ch|sh|ss)$`).MatchString(lowerWord):
		return word + "es", nil
	// Words ending with 'y' preceded by a consonant => replace 'y' with 'ies'
	case regexp.MustCompile(`(?i)[^aeiou]y$`).MatchString(lowerWord):
		return word[:len(word)-1] + "ies", nil
	// Words ending with 'f' or 'fe' => replace with 'ves'
	case regexp.MustCompile(`(?i)(f|fe)$`).MatchString(lowerWord):
		if strings.HasSuffix(lowerWord, "fe") {
			return word[:len(word)-2] + "ves", nil
		}
		return word[:len(word)-1] + "ves", nil
	// Words ending with 'o' preceded by a consonant => add 'es'
	case regexp.MustCompile(`(?i)[^aeiou]o$`).MatchString(lowerWord):
		return word + "es", nil
	// Default rule: add 's'
	default:
		return word + "s", nil
	}
}

// Checks if a word starts with an uppercase letter
func isCapitalized(word string) bool {
	if len(word) == 0 {
		return false
	}
	return strings.ToUpper(string(word[0])) == string(word[0])
}

// Capitalizes the first letter of a word
func capitalize(word string) string {
	if len(word) == 0 {
		return word
	}
	return strings.ToUpper(string(word[0])) + word[1:]
}
