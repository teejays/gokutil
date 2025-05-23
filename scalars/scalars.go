package scalars

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"

	"github.com/teejays/gokutil/errutil"
	jsonhelper "github.com/teejays/gokutil/gopi/json"
	"github.com/teejays/gokutil/panics"
)

type IScalar interface {
	String() string
	IsEmpty() bool
	Validate() error
	ParseString(str string) error
	// JSON
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
	// SQL
	Scan(value interface{}) error
	Value() (driver.Value, error)
	// GraphQL
	ImplementsGraphQLType(name string) bool
	UnmarshalGraphQL(input interface{}) error
}

// _ is a list of all the scalars. This is used to ensure that the listed scalars have all IScalar methods implemented.
var _ = []IScalar{
	&ID{},
	&Timestamp{},
	&Date{},
	&Email{},
	&Secret{},
}

/* * * * * * *
 * ID (UUID V4)
* * * * * * */

type ID struct {
	uuid.UUID
	// Sometimes we're simply comparing whether an ID includes a string e.g. when searching using a part of an ID.
	// However, goku's type safety in filters requires that ID's can only be compared with other ID's.
	// To handle this, we store the string value of the partial ID in matchString.
	// If matchString is populated, then the UUID field should be empty and ignored, and vice versa.
	matchString string
}

// NewID creates a new random ID.
func NewID() ID {
	return ID{UUID: uuid.New()}
}

func (id ID) String() string {
	if id.UUID != uuid.Nil {
		return id.UUID.String()
	}
	return id.matchString
}

func (id ID) IsEmpty() bool {
	return id.UUID == uuid.Nil && id.matchString == ""
}

func (id ID) Validate() error {
	// If both UUID and matchString are set, return an error
	if id.UUID != uuid.Nil && id.matchString != "" {
		return fmt.Errorf("ID is invalid: UUID and matchString cannot both be set")
	}
	// If both are empty, return an error
	if id.IsEmpty() {
		return fmt.Errorf("ID is empty")
	}
	// If UUID is invalid, return an error
	if err := uuid.Validate(id.String()); err != nil {
		return err
	}
	return nil
}

func (id *ID) ParseString(str string) error {
	uid, err := uuid.Parse(str)
	if err != nil {
		return err
	}
	*id = ID{UUID: uid}
	return nil
}

// JSON

func (id ID) MarshalJSON() ([]byte, error) {
	if id.IsEmpty() {
		return []byte("null"), nil
	}
	return strconv.AppendQuote(nil, id.String()), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	// If the data is null, set the ID to nil
	if string(data) == "null" {
		*id = ID{}
		return nil
	}
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	if s == "" {
		// ID is nil
		return nil
	}
	err = id.ParseString(s)
	if err != nil {
		return err
	}
	return nil
}

// SQL
// These methods have been implemented by the underlying uuid.UUID type.

func (id *ID) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	err := id.UUID.Scan(value)
	if err != nil {
		return err
	}
	return nil
}

func (d ID) Value() (driver.Value, error) {
	if d.IsEmpty() {
		return nil, nil
	}
	return d.UUID.Value()
}

// GraphQL
// GraphQL

func (id ID) ImplementsGraphQLType(name string) bool {
	return name == "ID"
}

func (id *ID) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		err = id.ParseString(input)
		if err != nil {
			return err
		}
		return nil
	default:
		err = fmt.Errorf("wrong type for ID: %T", input)
	}
	return err
}

// Custom Methods

// NewStaticID creates a new UUID from a hash.
func NewStaticID(hash string) ID {
	// Hash the hash into a 16 length byte slice
	hashBytes := sha256.Sum256([]byte(hash))
	// shortHash := hashBytes[:16]
	shortHash := make([]byte, 0, 16)

	// Take alternate (even) bytes until we have 16 bytes.
	// If there are not enough bytes, loop back to the start of the hash and repeat with the odd bytes.
	mod := 0
	i := 0
	for len(shortHash) < 16 {
		if i >= len(hashBytes) {
			// Reset counter
			i = 0
			// Switch between even and odd bytes
			mod = 1 - mod // 0 -> 1, 1 -> 0
		}
		if i%2 == mod {
			i++
			continue
		}
		shortHash = append(shortHash, hashBytes[i])
		i++
	}

	// Convert the short hash to a UUID
	uid, err := uuid.FromBytes(shortHash)
	if err != nil {
		return ID{}
	}
	return ID{UUID: uid}
}

// NewIDFromString creates a new ID from a string. The string must be a valid UUID.
func NewIDFromString(str string) (ID, error) {
	uid, err := uuid.Parse(str)
	if err != nil {
		return ID{}, err
	}
	return ID{UUID: uid}, nil
}

func NewIDMatchString(str string) ID {
	return ID{matchString: str}
}

/* * * * * * *
 * Timestamp
* * * * * * */

type Timestamp struct {
	graphql.Time
}

// NewTime creates a new Timestamp from a time.Time
// Deprecated: Use NewTimestamp instead
func NewTime(t time.Time) Timestamp {
	return NewTimestamp(t)
}

func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{Time: graphql.Time{Time: t}}
}

func (t Timestamp) IsEmpty() bool {
	return t.Time.IsZero()
}

func (t Timestamp) Validate() error {
	if t.IsEmpty() {
		return fmt.Errorf("Time is empty")
	}
	return nil
}

func (t *Timestamp) ParseString(str string) error {
	_t, err := NewTimestampFromString(str)
	if err != nil {
		return err
	}
	*t = _t
	return nil
}

func (t *Timestamp) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		*t = NewTime(v)
		return nil
	case string:
		return t.ParseString(v)
	default:
		return fmt.Errorf("could not decode SQL db value into scalar.Time field: %v", v)
	}
}

func (t Timestamp) Value() (driver.Value, error) {
	return t.ToGolangTime(), nil
}

func (d Timestamp) ImplementsGraphQLType(name string) bool {
	return name == "Timestamp"
}

// Custom Methods

// NewTimestampNow creates a new Timestamp with the current time. If the environment variable GOKUTIL_TIMESTAMP_NOW is set, it will use that value instead.
// This is useful for testing when you want to have a consistent time.
func NewTimestampNow() Timestamp {
	now := time.Now()
	overrideNowStr := os.Getenv("GOKUTIL_TIMESTAMP_NOW")
	if overrideNowStr != "" {
		var err error
		now, err = time.Parse(time.RFC3339, overrideNowStr)
		panics.IfError(err, "Parsing GOKUTIL_TIMESTAMP_NOW [%s]: %s. Make sure the value is RFC3339", overrideNowStr, err)
	}
	return NewTime(now)
}

func NewTimestampFromString(str string) (Timestamp, error) {
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return Timestamp{}, err
	}
	return NewTime(t), nil
}

func (t Timestamp) ToGolangTime() time.Time {
	return t.Time.Time
}

// Standardize ensures that the time is stored in the correct format i.e. in UTC.
func (t *Timestamp) Standardize() {
	t.Time.Time = t.Time.Time.UTC()
}

func (t Timestamp) Equal(other Timestamp) bool {
	return t.Time.Equal(other.Time.Time)
}

func (t Timestamp) IsBeforeNow() bool {
	now := NewTimestampNow()
	return t.Time.Time.Before(now.Time.Time)
}

/* * * * * * *
 * Date
* * * * * * */

// NewDate takes a string in the format "2006-01-02" and returns a Date. It errors if the string is not in the correct format.
type Date struct {
	year  int
	month time.Month
	day   int
}

var dateDefaultFormat = "2006-01-02"
var dateFullFormat = "2006-01-02T15:04:05.000Z" // Sometimes clients send the date in this format, since it's the default format for JS Date

func NewDate(year int, month time.Month, day int) (Date, error) {
	d := Date{year: year, month: month, day: day}
	if err := d.Validate(); err != nil {
		return Date{}, err
	}
	return d, nil
}

func (d Date) String() string {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC).Format(dateDefaultFormat)
}

func (d Date) IsEmpty() bool {
	return d.year == 0 && d.month == 0 && d.day == 0
}

func (d Date) Validate() error {
	if d.year == 0 && d.month == 0 && d.day == 0 {
		return fmt.Errorf("date is empty")
	}
	if d.month < 1 || d.month > 12 {
		return fmt.Errorf("month [%d] is out of range", d.month)
	}
	if d.day < 1 || d.day > 31 {
		return fmt.Errorf("day [%d] is out of range", d.day)
	}
	// Ensure the date is valid
	if _, err := time.Parse("2006-01-02", d.String()); err != nil {
		return fmt.Errorf("date [%s] is invalid: %v", d.String(), err)
	}
	return nil
}

func (d *Date) ParseString(str string) error {
	_d, err := NewDateFromString(str)
	if err != nil {
		return err
	}
	// Validate that the date is valid
	if err := _d.Validate(); err != nil {
		return err
	}
	*d = _d
	return nil
}

// JSON

func (d Date) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, d.String()), nil
}

func (d *Date) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	if s == "" {
		// date is nil
		return nil
	}
	// Parse the date
	date, err := NewDateFromString(s)
	if err != nil {
		return err
	}
	*d = date
	return nil
}

// SQL

func (d *Date) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var _d Date
	var err error

	switch v := value.(type) {
	case time.Time:
		_d = Date{year: v.Year(), month: v.Month(), day: v.Day()}
	case string:
		err = d.ParseString(v)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("could not decode SQL db value into scalar.Date field: %v", v)
	}
	// Validate that the date is valid
	if err := _d.Validate(); err != nil {
		return err
	}
	*d = _d
	return nil
}

func (d Date) Value() (driver.Value, error) {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC), nil
}

// GraphQL

func (d Date) ImplementsGraphQLType(name string) bool {
	return name == "Date"
}

func (d *Date) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		date, err := NewDateFromString(input)
		if err != nil {
			return err
		}
		*d = date
	default:
		err = fmt.Errorf("wrong type for Date: %T", input)
	}
	return err
}

// Custom Methods

// MustNewDate takes a string in the format "2006-01-02" and returns a Date. It panics if the string is not in the correct format.
func MustNewDate(year int, month time.Month, day int) Date {
	d, err := NewDate(year, month, day)
	panics.IfError(err, "creating Date [year=%d, month=%d, day=%d]: %s", year, month, day, err)
	return d
}

func NewDateFromString(str string) (Date, error) {
	v, err1 := NewDateFromStringFormat(str, dateDefaultFormat)
	if err1 == nil {
		return v, nil
	}
	v, err2 := NewDateFromStringFormat(str, dateFullFormat)
	if err2 == nil {
		return v, nil
	}
	return Date{}, fmt.Errorf("Could not parse date string [%s]: expected format [%s] or [%s]", str, dateDefaultFormat, dateFullFormat)

}

func NewDateFromStringFormat(str, format string) (Date, error) {
	t, err := time.Parse(format, str)
	if err != nil {
		return Date{}, fmt.Errorf("could not parse date string using format [%s]: %v", format, err)
	}
	d := Date{year: t.Year(), month: t.Month(), day: t.Day()}
	if err := d.Validate(); err != nil {
		return Date{}, fmt.Errorf("date [%s] is invalid: %v", d.String(), err)
	}
	return d, nil
}

func NewDateFromTime(t time.Time) Date {
	d := Date{year: t.Year(), month: t.Month(), day: t.Day()}
	if err := d.Validate(); err != nil {
		return Date{}
	}
	return d
}

/* * * * * * *
 * Email
* * * * * * */

type Email struct {
	GenericStringScalar
}

func NewEmail(value string) Email {
	return Email{GenericStringScalar: NewGenericStringScalar(value)}
}

func (e Email) Validate() error {
	if e.IsEmpty() {
		return fmt.Errorf("email is empty")
	}
	if !IsEmail(e.value) {
		return fmt.Errorf("email [%s] is invalid", e.value)
	}
	return nil
}

func (e Email) ImplementsGraphQLType(name string) bool {
	return name == "Email"
}

// Custom Methods

func IsEmail(email string) bool {
	emailAddress, err := mail.ParseAddress(email)
	return err == nil && emailAddress.Address == email
}

/* * * * * * *
 * Secret
* * * * * * */

type Secret struct {
	GenericStringScalar
}

func NewSecret(value string) Secret {
	return Secret{GenericStringScalar: NewGenericStringScalar(value)}
}

func (s Secret) ImplementsGraphQLType(name string) bool {
	return name == "Secret"
}

/* * * * * * *
 * GenericData
* * * * * * */

type GenericData struct {
	value string // JSON. We could use []byte, but string is comparable
}

func (g GenericData) String() string {
	return string(g.value)
}

func (g GenericData) IsEmpty() bool {
	return g.value == ""
}

func (g *GenericData) ParseString(str string) error {
	// Assume the string is a JSON string
	// Ensure it is valid JSON
	// if str is qouted, assume that it is a string
	if strings.HasPrefix(str, "\"") && strings.HasSuffix(str, "\"") {
		g.value = str
		return nil
	}
	// Assume object
	m := map[string]interface{}{}
	err := json.Unmarshal([]byte(str), &m)
	if err != nil {
		return err
	}
	g.value = str
	return nil
}

func (g GenericData) MarshalJSON() ([]byte, error) {
	if g.IsEmpty() {
		return []byte("null"), nil
	}
	return []byte(g.value), nil
}

func (g *GenericData) UnmarshalJSON(data []byte) error {
	return g.ParseString(string(data))
}

func (g *GenericData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	// Stored in SQL as type jsonb
	switch v := value.(type) {
	case []byte:
		err := g.ParseString(string(v))
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("could not scan SQL db value into scalar.GenericData field: %v", v)
	}

}

func (s GenericData) ImplementsGraphQLType(name string) bool {
	return name == "GenericData"
}

func (g *GenericData) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		err = g.ParseString(input)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("wrong type for GenericData: %T", input)
	}
}

func (g GenericData) Value() (driver.Value, error) {
	if g.IsEmpty() {
		return nil, nil
	}
	return g.value, nil
}

// NewGenericDataFromStruct name is misleading because it handles non-struct values as well.
func NewGenericDataFromStruct(value interface{}) (GenericData, error) {
	// If value is nil
	if value == nil {
		return GenericData{}, nil
	}
	// If value is a string
	if str, ok := value.(string); ok {
		return GenericData{value: strconv.Quote(str)}, nil
	}
	// If value is error
	if err, ok := value.(error); ok {
		if err == nil {
			return GenericData{}, nil
		}
		return GenericData{value: strconv.Quote(err.Error())}, nil
	}
	// Assume the value is an object
	bytes, err := json.Marshal(value)
	if err != nil {
		return GenericData{}, err
	}
	var m = map[string]interface{}{}
	err = json.Unmarshal(bytes, &m)
	if err != nil {
		return GenericData{}, err
	}

	return GenericData{value: string(bytes)}, nil
}

// LoadTo unmarshals the JSON value into the provided struct. The struct v should be a pointer.
func (g GenericData) LoadTo(v interface{}) error {
	if g.IsEmpty() {
		return fmt.Errorf("Cannot LoadTo an empty GenericData value")
	}

	return jsonhelper.UnmarshalStrict([]byte(g.value), v)
}

/* * * * * * *
 * Money
* * * * * * */

type Money struct {
	whole   uint
	decimal uint
	init    bool
}

func NewMoney(whole uint, decimal uint) (Money, error) {
	v := Money{whole: whole, decimal: decimal, init: true}
	if err := v.Validate(); err != nil {
		return Money{}, err
	}
	return v, nil
}

func (v Money) String() string {
	return fmt.Sprintf("%d.%d", v.whole, v.decimal)
}

func (v Money) IsEmpty() bool {
	return !v.init
}

func (v Money) Validate() error {
	if v.IsEmpty() {
		return fmt.Errorf("is empty")
	}

	// cannot be negative
	if v.whole < 0 {
		return fmt.Errorf("whole value [%d] cannot be negative", v.whole)
	}
	if v.decimal < 0 {
		return fmt.Errorf("decimal value [%d] cannot be negative", v.decimal)
	}
	// decimal must be between 0 and 99
	if v.decimal > 99 {
		return fmt.Errorf("decimal value [%d] is out of range [0-99]", v.decimal)
	}
	return nil
}

func (v *Money) ParseString(str string) error {
	_v, err := NewMoneyFromString(str)
	if err != nil {
		return err
	}
	// Validate that the date is valid
	if err := _v.Validate(); err != nil {
		return err
	}
	*v = _v
	return nil
}

// JSON

func (v Money) MarshalJSON() ([]byte, error) {
	// If the value is empty, return nil
	if v.IsEmpty() {
		return nil, nil
	}
	// Otherwise, return as a two decimal float
	s := fmt.Sprintf("%d.%02d", v.whole, v.decimal)
	return []byte(s), nil
}

func (v *Money) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		// the value did not have qoutes. Try using the raw value.
		s = string(data)
	}
	if s == "" {
		// money is nil
		return nil
	}
	// Parse
	_v, err := NewMoneyFromString(s)
	if err != nil {
		return err
	}
	*v = _v
	return nil
}

// SQL

func (v *Money) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var _v Money
	var err error

	switch v := value.(type) {
	case string:
		err = _v.ParseString(v)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("could not decode SQL db value into scalar.Money field: %v", v)
	}
	// Validate that the date is valid
	if err := _v.Validate(); err != nil {
		return err
	}
	*v = _v
	return nil
}

func (v Money) Value() (driver.Value, error) {
	return fmt.Sprintf("%d.%d", v.whole, v.decimal), nil
}

// GraphQL

func (v Money) ImplementsGraphQLType(name string) bool {
	return name == "Money"
}

func (v *Money) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		money, err := NewMoneyFromString(input)
		if err != nil {
			return err
		}
		*v = money
	default:
		err = fmt.Errorf("wrong type for Money: %T", input)
	}
	return err
}

// Custom Methods

func NewMoneyFromString(str string) (Money, error) {
	// Expect string to be in the format "XY.MN" where XY is the whole value and MN is the decimal value
	// split the string into whole and decimal parts
	if str == "" {
		return Money{}, fmt.Errorf("empty string")
	}
	parts := strings.Split(str, ".")
	// expect 1 or 2 parts
	if len(parts) > 2 || len(parts) == 0 {
		return Money{}, fmt.Errorf("invalid string for money [%s]", str)
	}

	// Function to parse string to uint
	var parseUint = func(s string) (uint, error) {
		i, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("could not parse to uint: %w", err)
		}
		if i < 0 {
			return 0, fmt.Errorf("value [%d] is negative", i)
		}
		return uint(i), nil
	}

	// Parse the whole value
	whole, err := parseUint(parts[0])
	if err != nil {
		return Money{}, fmt.Errorf("Parsing whole value [%s]: %w", parts[0], err)
	}

	var decimal uint
	if len(parts) > 1 {
		decimal, err = parseUint(parts[1])
		if err != nil {
			return Money{}, fmt.Errorf("Parsing decimal value [%s]: %w", parts[1], err)
		}
	}

	return NewMoney(whole, decimal)
}

func (v *Money) adjust() {
	if v.decimal > 99 {
		v.whole += v.decimal / 100
		v.decimal = v.decimal % 100
	}
}

func (v Money) GetWhole() uint {
	return v.whole
}

func (v Money) GetDecimal() uint {
	return v.decimal
}

func (v Money) ToFloat32() float32 {
	return float32(v.whole) + float32(v.decimal)/100
}

func (v Money) ToFloat64() float64 {
	return float64(v.whole) + float64(v.decimal)/100
}

func (v Money) Add(other Money) (Money, error) {
	new := Money{whole: v.whole + other.whole, decimal: v.decimal + other.decimal}
	new.adjust()
	return new, nil
}

// Not adding sub method because it can be tricky to handle negative values

/* * * * * * *
* Link
* * * * * * */

// Link is a scalar that represents a link/URL. It is a string scalar.
type Link struct {
	GenericStringScalar
}

func NewLink(value string) (Link, error) {
	link := Link{GenericStringScalar: NewGenericStringScalar(value)}
	if err := link.Validate(); err != nil {
		return Link{}, err
	}
	return link, nil
}

func (v Link) Validate() error {
	if v.IsEmpty() {
		return fmt.Errorf("link is empty")
	}
	if _, err := url.Parse(v.value); err != nil {
		return errutil.Wrap(err, "Parsing link as a URL")
	}
	return nil
}

func (v Link) ImplementsGraphQLType(name string) bool {
	return name == "Link"
}

/* * * * * * *
* Generic String Scalar
* * * * * * */

type GenericStringScalar struct {
	value string
}

func NewGenericStringScalar(value string) GenericStringScalar {
	return GenericStringScalar{value: value}
}

func (g GenericStringScalar) String() string {
	return g.value
}

func (g GenericStringScalar) IsEmpty() bool {
	return g.value == ""
}

func (g *GenericStringScalar) Validate() error {
	if g.IsEmpty() {
		return fmt.Errorf("value is empty")
	}
	return nil
}

func (g *GenericStringScalar) ParseString(str string) error {
	g.value = str
	return nil
}

// JSON

func (g GenericStringScalar) MarshalJSON() ([]byte, error) {
	s := strconv.Quote(g.String())
	return []byte(s), nil
}

func (g *GenericStringScalar) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	err = g.ParseString(s)
	if err != nil {
		return err
	}
	return nil
}

// SQL

func (g *GenericStringScalar) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		err := g.ParseString(v)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("could not scan SQL db value into scalar.GenericStringScalar: %v", v)
	}
}

func (g GenericStringScalar) Value() (driver.Value, error) {
	return g.String(), nil
}

// GraphQL

// ImplementsGraphQLType should be implemented by the actual scalar types
/*
func (g *GenericStringScalar) ImplementsGraphQLType(name string) bool {
	return name == "String"
}
*/

func (g *GenericStringScalar) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		err = g.ParseString(input)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("wrong type for GenericStringScalar: %T", input)
	}
}
