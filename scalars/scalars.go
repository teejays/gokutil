package scalars

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
)

type ID struct {
	uuid.UUID
}

func NewID() ID {
	return ID{uuid.New()}
}

func NewStaticID(hash string) ID {
	return ID{uuid.New()}
}

func ParseID(s string) (ID, error) {
	uid, err := uuid.Parse(s)
	if err != nil {
		return ID{}, err
	}
	return ID{uid}, nil
}

func (id ID) IsEmpty() bool {
	return id.UUID == uuid.Nil
}

// ImplementsGraphQLType maps this custom Go type to the graphql scalar type in the schema.
func (id ID) ImplementsGraphQLType(name string) bool {
	return name == "ID"
}

func (id *ID) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		uid, err := ParseID(input)
		if err != nil {
			return err
		}
		*id = uid
	default:
		err = fmt.Errorf("wrong type for ID: %T", input)
	}
	return err
}

func (id ID) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, id.String()), nil
}

type Time struct {
	graphql.Time
}

func NewTime(t time.Time) Time {
	return Time{Time: graphql.Time{Time: t}}
}

func (t Time) ToGolangTime() time.Time {
	return t.Time.Time
}

func (t *Time) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		*t = NewTime(v)
		return nil
	default:
		return fmt.Errorf("could not decode SQL db value into scalar.Time field: %v", v)
	}
}

func (t Time) Value() (driver.Value, error) {
	return t.ToGolangTime(), nil
}
