package naam

import (
	"encoding/json"
	"fmt"
	"strings"

	_ "fmt"

	"github.com/teejays/goku-util/panics"
	"github.com/teejays/goku-util/strcase"
)

// Named is any object/struct that has a name
type Named interface {
	GetName() Name
}

var Empty = Name{words: ""}
var ID = Name{words: "id"}

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
		sep = "_" + sep + "_"
	} else {
		sep = "_"
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
	words = strings.Trim(words, "_")
	// Allow spaces, convert them to "_"
	words = strings.ReplaceAll(words, " ", "_")
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

// MarshalJSON implements the `json.Marshaler` interface, so Name can be converted to a human friendly string in JSON.
func (s Name) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// MarshalJSON implements the `json.Unmarshaler` interface, so strings in JSON can easily converted to Name.
func (s *Name) UnmarshalJSON(b []byte) error {
	s.ParseString(string(b))
	return nil
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
	words := cleanWords(str)
	s.words = words
}

// MarshalText implements in the `encoding.TextMarshaler` interface.
// NOTE: Not sure why want to implement it.
func (s Name) MarshalText() ([]byte, error) {
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
	s.words = s.words + "_" + a
	return s.clean()
}

// AppendName is a wrapper around Join. It joins the Name `a` to the end of Name `s`.
func (s Name) AppendName(a Name) Name {
	return Join("", s, a)
}

// Prepend is a Name manipulator function. It prepends a string to the Name.
func (s Name) Prepend(a string) Name {
	s.words = a + "_" + s.words
	return s.clean()
}

// PrependName is a wrapper around Join. It joins the Name `a` at the start of Name `s`.
func (s Name) PrependName(a Name) Name {
	return Join("", a, s)
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
	n.words = strings.TrimSuffix(n.words, "_id")
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
	parts[len(parts)-1] = strcase.Pluralize(lastPart)
	return New(strings.Join(parts, " "))
}

// Pluralize attempts to make Name `s` into a singular version of itself.
func (s Name) Singularize() Name {
	parts := s.wordParts()
	lastPart := parts[len(parts)-1]
	parts[len(parts)-1] = strcase.Singularize(lastPart)
	return New(strings.Join(parts, " "))
}

// wordParts returns the individual words in the Name `s`.
// TODO: Handle major parts that are separated by `PartsSeperator`.
func (s Name) wordParts() []string {
	return strings.Split(s.words, "_")
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
	if len(s.words) < len(str) {
		return false
	}
	return s.words[len(s.words)-len(str):] == str
}

// String makes Name a Stringer, so Names are printed in a slightly nicer manner in print/log statements.
func (s Name) String() string {
	return strcase.ToSnakedCamel(s.words)
}

// ToCamel converts the Name `s` into a CamelCase version, however it doesn't respect the parts.
func (s Name) ToCamel() string {
	panics.If(s.IsEmpty(), "naam.Name.ToCamel() called on empty Name")
	return strcase.ToCamel(s.String())
}

// ToPascal is an alias for ToCamel
func (s Name) ToPascal() string {
	panics.If(s.IsEmpty(), "naam.Name.ToPascal() called on empty Name")
	return s.ToSnakedCamel()
}

// ToLowerCamel converts the Name into a lowerCamelCase version, however it doesn't respect the parts.
func (s Name) ToLowerCamel() string {
	panics.If(s.IsEmpty(), "naam.Name.ToLowerCamel() called on empty Name")
	return strcase.ToLowerCamel(s.String())
}

// ToSnake prints a `snake_case` version of the Name. It should take into account any Name parts, so it could end up
// looking like `main__snake_case`. ToSnake is not lossless. Once in a snake case, the string cannot be directly converted back to a Name.
func (s Name) ToSnake() string {
	panics.If(s.IsEmpty(), "naam.Name.ToSnake() called on empty Name")
	return strcase.ToSnake(s.words)
}

// PartsSeperator is the string used to separated two major parts within the name.
const PartsSeperator = strcase.PartsSep

// ToSnakedCamel converts the Name into a CamelCase, but it respects any major parts of the name (separated by three `_`) and
// keeps them seperated by a single `_`. E.g. `hello___best_bay` -> `Hello_BestBay`
func (s Name) ToSnakedCamel() string {
	return strcase.ToSnakedCamel(s.words)
}

// ToSnakeUpper converts HELLO_DAISY
func (s Name) ToSnakeUpper() string {
	return strings.ToUpper(s.ToSnake())
}

// ToKebab prints a `snake_case` version of the Name. It, yet, does not take into account any Name parts.
func (s Name) ToKebab() string {
	panics.If(s.IsEmpty(), "naam.Name.ToKebab() called on empty Name")
	return strcase.ToKebab(s.words)
}

// ToCompact returns an unrecoverable string representation of the name: 'purchase_id' => 'purchaseid'
func (s Name) ToCompact() string {
	panics.If(s.IsEmpty(), "naam.Name.ToCompact() called on empty Name")
	return strings.ToLower(strcase.ToCamel(s.words))
}

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
		panic(fmt.Sprintf("naam.FormatSQL(): name '%s' is longer than 63 characters (which is the SQL limit): ", s.words))
	}
	return strcase.ToSnake(s.Truncate(63).words)
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

// FormatGolang prints a general string representation of Name `s`, which may be appropriate for Golang codebase.
func (s Name) FormatGolang() string {
	// TODO: Use ToSnakedCamel
	return strcase.ToSnakedCamel(s.words)
}

// FormatGolangType prints a string representation of Name `s` which can be used an exported Type name in Golang.
func (s Name) FormatGolangType() string {
	// TODO: Use ToSnakedCamel instead, since it's a type name which cannot have `.`.
	return strcase.ToSnakedCamel(s.words)
}

// FormatGolangFieldName prints a string representation of Name `s` which can be used an exported Field name in Golang.
func (s Name) FormatGolangFieldName() string {
	return strcase.ToSplitCamel(s.words, ".")
}

// FormatFileName prints a strings representation of Name `s` that can used as part of a Unix file or directory name.
func (s Name) FormatFileName() string {
	return strcase.ToSnake(s.words)
}

func (s Name) FormatTypescript() string {
	return strcase.ToSnake(s.words)
}

func (s Name) FormatTypescriptType() string {
	return strcase.ToCamel(s.words)
}

func (s Name) FormatTypescriptField() string {
	return strcase.ToSnake(s.words)
}

func (s Name) FormatTypescriptVar() string {
	return strcase.ToLowerCamel(s.words)
}

func (s Name) FormatGraphql() string {
	return strcase.ToSnake(s.words)
}
