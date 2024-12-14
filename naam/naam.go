package naam

import (
	"encoding/json"
	"fmt"
	"strings"

	_ "fmt"

	"github.com/invopop/jsonschema"

	"github.com/teejays/gokutil/panics"
	"github.com/teejays/gokutil/strcase"
)

const _sep = "_"

// PartsSeperator is the string used to separated two major parts within the name.
const PartsSeperator = strcase.PartsSep

var Empty = Name{words: ""}
var ID = Name{words: "id"}

/* * * * * * * *
 * Named Interface
 * * * * * * * */

// Named is any object/struct that has a name
type Named interface {
	GetName() Name
}

/* * * * * * * *
 * Name
 * * * * * * * */

// Name is the ry struct that stores our name.
type Name struct {
	words string
}

// New creates a new name. The input string can be free form
func New(words string) Name {
	n := Name{}
	n.ParseString(words)
	return n
}

// Nil provides an empty Name, useful to compare and figure out if a Name is empty
func Nil() Name {
	return Name{}
}

// Join combines two or more names with a string in between: 'map', 'purchase', 'items', 'company' => 'purchase_map_items_map_company'.
func Join(sep string, names ...Name) Name {
	var strs []string
	for _, n := range names {
		strs = append(strs, n.words)
	}
	if sep != "" {
		sep = _sep + sep + _sep
	} else {
		sep = _sep
	}
	n := Name{
		words: strings.Join(strs, sep),
	}
	return n.clean()
}

// Less is a helper function that helps sort an array of Names.
// Example usage:
// ```
//
//	sort.Slice( sliceOfNames, func(i, j int) bool {
//		return naam.Less(app.Services[i].Name, app.Services[j].Name)
//	})
//
// ```
func Less(a, b Name) bool {
	return a.clean().words < b.clean().words
}

// clean ensures that underneath string is always in one standard format i.e. in lower snake case, maintaining major parts sep of `_ _ _`
func cleanWords(words string) string {
	// Trim edge spaces
	words = strings.TrimSpace(words)
	// Trim edge underscores
	words = strings.Trim(words, _sep)
	// Allow spaces, convert them to _sep
	words = strings.ReplaceAll(words, " ", _sep)
	parts := strings.Split(words, strcase.PartsSep)
	// Need to protect triple `_` seperators
	for i := range parts {
		parts[i] = strcase.ToSnake(parts[i])
	}
	words = strings.Join(parts, strcase.PartsSep)

	// Make all words lowercase for consistency
	words = strings.ToLower(words)
	return words
}

// String makes Name a Stringer, so Names are printed in a slightly nicer manner in print/log statements.
func (s Name) String() string {
	return strcase.ToPascal(s.words)
}

// MarshalJSON implements the `json.Marshaler` interface, so Name can be converted to a human friendly string in JSON.
func (s Name) MarshalJSON() ([]byte, error) {
	if s.IsEmpty() {
		return json.Marshal("")
	}
	return json.Marshal(s.String())
}

// MarshalJSON implements the `json.Unmarshaler` interface, so strings in JSON can easily converted to Name.
func (s *Name) UnmarshalJSON(b []byte) error {
	s.ParseString(string(b))
	return nil
}

func (Name) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Description: "A name provided as a string in snake_case.",
	}
}

func (s *Name) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	s.ParseString(str)
	return nil
}

// ParseString is a helper function that parses a string into a Name.
func (s *Name) ParseString(str string) {
	// unqoute
	str = strings.Trim(str, "\"")
	str = strings.Trim(str, "'")
	words := cleanWords(str)
	s.words = words
}

// MarshalText implements in the `encoding.TextMarshaler` interface.
// NOTE: Not sure why want to implement it.
func (s Name) MarshalText() ([]byte, error) {
	if s.IsEmpty() {
		return []byte{}, nil
	}
	return []byte(s.ToSnake()), nil
}

// Raw prints out the info in Name as is, useful for debugging.
func (s Name) Raw() string {
	return s.words
}

// clean ensures that underneath string is always in one standard format i.e. in lower snake case
func (s Name) clean() Name {
	return Name{words: cleanWords(s.words)}
}

// Contains returns true if s contains the provided sub string (case insensitive)
func (s Name) ContainsString(sub string) bool {
	return strings.Contains(s.words, cleanWords(sub))
}

// Append is a Name manipulator function. It appends a string to the end of the Name
func (s Name) Append(a string) Name {
	s.words = s.words + _sep + a
	return s.clean()
}

// AppendName is a wrapper around Join. It joins the Name `a` to the end of Name `s`.
func (s Name) AppendName(a Name) Name {
	return Join("", s, a)
}

// AppendNameNamespaced appends a name but with a namespace separator, so the the final results is `s__a`.
func (s Name) AppendNameNamespaced(a Name) Name {
	return Join(_sep, s, a)
}

// Prepend is a Name manipulator function. It prepends a string to the Name.
func (s Name) Prepend(a string) Name {
	s.words = a + _sep + s.words
	return s.clean()
}

// PrependName is a wrapper around Join. It joins the Name `a` at the start of Name `s`.
func (s Name) PrependName(a Name) Name {
	return Join("", a, s)
}

// PrependNameNamespaced appends a name but with a namespace separator, so the the final results is `a__s`.
func (s Name) PrependNameNamespaced(a Name) Name {
	return Join(_sep, a, s)
}

// AppendID is a helper function that adds `_id` to the Name.
func (n Name) AppendID() Name {
	return Join("", n, ID).clean()
}

// TrimSuffixString removes the provided string suffix from Name
func (n Name) TrimSuffixString(s string) Name {
	n.words = strings.TrimSuffix(n.words, s)
	return n.clean()
}

// TrimSuffixID undoes `AppendID`, as it removes any trailing `id` in the Name.
func (n Name) TrimSuffixID() Name {
	n.words = strings.TrimSuffix(n.words, _sep+"id")
	return n.clean()
}

// Truncates reduces the length of characters stored in the Name by limiting it to `n` characters. It is not losless,
// so the output of Truncate may not lead back to Name `s`.
func (s Name) Truncate(n int) Name {
	if len(s.words) > n {
		s.words = s.words[:n]
	}
	return s.clean()
}

// Pluralize attempts to make Name `s` into a plural version.
func (s Name) Pluralize() Name {
	parts := s.wordParts()
	lastPart := parts[len(parts)-1]
	parts[len(parts)-1] = strcase.MustPluralize(lastPart)
	return New(strings.Join(parts, " "))
}

// wordParts returns the individual words in the Name `s`.
// TODO: Handle major parts that are separated by `PartsSeperator`.
func (s Name) wordParts() []string {
	return strings.Split(s.words, _sep)
}

// IsEmpty checks if the Name is empty. This is sometimes interchangeably used with `s == name.Nil()`
func (s Name) IsEmpty() bool {
	return len(s.words) < 1
}

// Equal checks if Name `s` is the is the same as `a`.
func (s Name) Equal(a Name) bool {
	return s.words == a.words
}

// EqualString checks is Name `s` is equivalent to the Name generated by string `str`.
func (s Name) EqualString(str string) bool {
	a := New(str)
	return s.Equal(a)
}

// HasSuffixString checks is the Name `s` has a suffix string equivalent to `str`.
func (s Name) HasSuffixString(str string) bool {
	// Convert the string to name and then use it's words
	str = New(str).words
	if len(s.words) < len(str) {
		return false
	}
	return s.words[len(s.words)-len(str):] == str
}

// Get the `a` or `an` before the Name. It is used in sentences.
func (s Name) GetArticle() string {
	if strings.Contains("aeiou", string(s.words[0])) {
		return "an"
	}
	return "a"
}

/* * * * * * * *
 * Formatters (Main)
 * * * * * * * */

// ToPascal is an alias for ToPascal
func (s Name) ToPascal() string {
	panics.If(s.IsEmpty(), "naam.Name.ToPascal() called on empty Name")
	return strcase.ToPascal(s.words)
}

// ToCamel converts the Name into a lowerCamelCase version, however it doesn't respect the parts.
func (s Name) ToCamel() string {
	panics.If(s.IsEmpty(), "naam.Name.ToCamel() called on empty Name")
	return strcase.ToCamel(s.words)
}

// ToSnake prints a `snake_case` version of the Name. It should take into account any Name parts, so it could end up
// looking like `main__snake_case`. ToSnake is not lossless. Once in a snake case, the string cannot be directly converted back to a Name.
func (s Name) ToSnake() string {
	panics.If(s.IsEmpty(), "naam.Name.ToSnake() called on empty Name")
	return strcase.ToSnake(s.words)
}

// ToKebab prints a `snake_case` version of the Name. It, yet, does not take into account any Name parts.
func (s Name) ToKebab() string {
	panics.If(s.IsEmpty(), "naam.Name.ToKebab() called on empty Name")
	return strcase.ToKebab(s.words)
}

// ToTitle converts the Name into a TitleCase, but it respects any major parts of the name (separated by three `_`) and
// keeps them seperated by a single `-`. E.g. `hello___best_bay` -> `Hello - Best Bay`
func (s Name) ToTitle() string {
	panics.If(s.IsEmpty(), "naam.Name.ToTitle() called on empty Name")
	return strcase.ToTitle(s.words)
}

// // ToSnakedCamel converts the Name into a CamelCase, but it respects any major parts of the name (separated by three `_`) and
// // keeps them seperated by a single `_`. E.g. `hello___best_bay` -> `Hello_BestBay`
// func (s Name) ToSnakedCamel() string {
// 	return strcase.ToSnakedCamel(s.words)
// }

// ToSnakeUpper converts HELLO_DAISY
func (s Name) ToSnakeUpper() string {
	panics.If(s.IsEmpty(), "naam.Name.ToSnakeUpper() called on empty Name")
	str := s.ToSnake()
	return strings.ToUpper(str)
}

/* * * * * * * *
 * Formatters (Others)
 * * * * * * * */

// ToURL prints a url friendly representation of the name.
// e.g. `snake_case` = snake_case
// `core___snake_case` = core/snake_case
func (s Name) ToURL() string {
	// convert the name to snake_case, and then replace the parts seperator with a `/`
	w := s.ToSnake()
	w = strings.ReplaceAll(w, "__", "/") // namespaced parts are separated by two "_" in snake case version
	return w
}

// ToCompact returns an unrecoverable string representation of the name: 'purchase_id' => 'purchaseid'
func (s Name) ToCompact() string {
	panics.If(s.IsEmpty(), "naam.Name.ToCompact() called on empty Name")
	return strings.ToLower(strcase.ToPascal(s.words))
}

// ToCompactUpper returns an unrecoverable string representation of the name: 'purchase_id' => 'PURCHASEID'
func (s Name) ToCompactUpper() string {
	panics.If(s.IsEmpty(), "naam.Name.ToCompact() called on empty Name")
	return strings.ToUpper(s.ToCompact())
}

// ToCompressedName returns an unrecoverable string representation of the name: 'purchase_id' => 'prchsid'
func (s Name) ToCompressedName() Name {
	panics.If(s.IsEmpty(), "naam.Name.ToCompressed() called on empty Name")
	// Remove all the vowels (unless they are the first letter of the word)
	words := s.wordParts()

	newWords := ""

	for _, word := range words {
		for j, letter := range word {
			// If it's the first letter of the word, or it's an underscore, keep it
			if j == 0 || letter == '_' {
				newWords += string(letter)
				continue
			}
			// If it's a consonant, keep it
			// If it's a vowel, skip it
			if !strings.Contains("aeiou", string(letter)) {
				newWords += string(letter)
				continue
			}

			continue
		}
		// at the end of the word, add an underscore
		newWords += "_"
	}
	return New(newWords)
}

/* * * * * * * *
 * Formatters (Language Specific)
 * * * * * * * */
// Formatting Functions: These functions print Name into a string type that is suitable to the subjective environment.

// FormatSQL prints a general string representation of Name `s`, which may be appropriate for SQL environment.
func (s Name) FormatSQL() string {
	// Should we truncate things here?
	// If the name is more than 64 characters, truncate because postgres truncates
	// Otherwise yamltodb fails to recognize that an fkey/object with a truncated name is already
	// in the database, since the name in the DB is different (truncated), and hence we end up
	// trying to add the same fkey/object again.
	// TODO: Instead of just truncating, maybe take a hash and append in the name so we can have unique names
	if len(s.words) > 63 {
		s = s.ToCompressedName()
	}
	if len(s.words) > 63 {
		panic(fmt.Sprintf("naam.FormatSQL(): name '%s' is longer than 63 characters (despite compression): ", s.words))
	}
	return s.ToSnake()
}

// FormatSQLColumn prints a string representation of Name `s`, which may be appropriate for a SQL column name.
func (s Name) FormatSQLColumn() string {
	// Should we truncate things here?
	// If the name is more than 64 characters, truncate because postgres truncates
	// Otherwise yamltodb fails to recognize that an fkey/object with a truncated name is already
	// in the database, since the name in the DB is different (truncated), and hence we end up
	// trying to add the same fkey/object again.
	// TODO: Instead of just truncating, maybe take a hash and append in the name so we can have unique names
	if len(s.words) > 63 {
		panic(fmt.Sprintf("naam.FormatSQLColumn(): name '%s' is longer than 63 characters (which is the SQL limit): ", s.words))
	}
	return strcase.ToSnake(s.Truncate(63).words)
}

// FormatSQLTable prints a string representation of Name `s`, which may be appropriate for within a SQL table name.
func (s Name) FormatSQLTable() string {
	// Should we truncate things here?
	// If the name is more than 64 characters, truncate because postgres truncates
	// Otherwise yamltodb fails to recognize that an fkey/object with a truncated name is already
	// in the database, since the name in the DB is different (truncated), and hence we end up
	// trying to add the same fkey/object again.
	// TODO: Instead of just truncating, maybe take a hash and append in the name so we can have unique names
	if len(s.words) > 63 {
		panic(fmt.Sprintf("naam.FormatSQLTable(): name '%s' is longer than 63 characters (which is the SQL limit): ", s.words))
	}
	return strcase.ToSnake(s.Truncate(63).words)
}

// FormatGolangType prints a string representation of Name `s` which can be used an exported Type name in Golang.
func (s Name) FormatGolangType() string {
	return s.ToPascal()
}

// FormatGolangTypeUnexported
func (s Name) FormatGolangTypeUnexported() string {
	return s.ToCamel()
}

// FormatGolangFieldName prints a string representation of Name `s` which can be used an exported Field name in Golang.
func (s Name) FormatGolangFieldName() string {
	return s.ToPascal()
}

// FormatFileName prints a strings representation of Name `s` that can used as part of a Unix file or directory name.
func (s Name) FormatFileName() string {
	return s.ToSnake()
}

func (s Name) FormatTypescriptType() string {
	return s.ToPascal()
}

func (s Name) FormatTypescriptField() string {
	return s.ToCamel()
}

func (s Name) FormatTypescriptVar() string {
	return s.ToCamel()
}

func (s Name) FormatGraphql() string {
	return s.ToSnake()
}

func (s Name) FormatJSONField() string {
	return s.ToCamel()
}
