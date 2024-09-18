package scalars

import (
	"crypto/sha256"
	"database/sql/driver"
	"fmt"
	"net/mail"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
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
}

// NewID creates a new random ID.
func NewID() ID {
	return ID{uuid.New()}
}

func (id ID) String() string {
	return id.UUID.String()
}

func (id ID) IsEmpty() bool {
	return id.UUID == uuid.Nil
}

func (id ID) Validate() error {
	if id.IsEmpty() {
		return fmt.Errorf("ID is empty")
	}
	if err := id.Validate(); err != nil {
		return err
	}
	return nil
}

func (id *ID) ParseString(str string) error {
	uid, err := uuid.Parse(str)
	if err != nil {
		return err
	}
	*id = ID{uid}
	return nil
}

// JSON

func (id ID) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, id.String()), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	err = id.ParseString(s)
	if err != nil {
		return err
	}
	return nil
}

// SQL
// These methods have been implemented by the underlying uuid.UUID type.

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
	// Code is auto-generated by Cursor AI

	// Hash the hash into a 16 length byte slice
	hashBytes := sha256.Sum256([]byte(hash))
	shortHash := hashBytes[:16]

	// Convert the short hash to a UUID
	uid, err := uuid.FromBytes(shortHash)
	if err != nil {
		return ID{}
	}
	return ID{uid}
}

// NewIDFromString creates a new ID from a string. The string must be a valid UUID.
func NewIDFromString(str string) (ID, error) {
	uid, err := uuid.Parse(str)
	if err != nil {
		return ID{}, err
	}
	return ID{uid}, nil
}

/* * * * * * *
 * Timestamp
* * * * * * */

type Timestamp struct {
	graphql.Time
}

func NewTime(t time.Time) Timestamp {
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
	_t, err := NewFromString(str)
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

func NewFromString(str string) (Timestamp, error) {
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return Timestamp{}, err
	}
	return NewTime(t), nil
}

func (t Timestamp) ToGolangTime() time.Time {
	return t.Time.Time
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
	return NewDateFromStringFormat(str, dateDefaultFormat)
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
