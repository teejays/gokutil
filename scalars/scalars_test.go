package scalars

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestID(t *testing.T) {

	// Harcoded UUID
	idStr1 := "123e4567-e89b-12d3-a456-426614174000"

	// Test NewID
	t.Run("NewID", func(t *testing.T) {
		id := NewID()
		assert.False(t, id.IsEmpty())
	})

	// Test ParseID
	t.Run("ParseID", func(t *testing.T) {
		id, err := ParseID(idStr1)
		assert.NoError(t, err)
		assert.False(t, id.IsEmpty())
		assert.Equal(t, idStr1, id.String())
	})

	// Test String
	t.Run("String", func(t *testing.T) {
		id := NewID()
		assert.NotEmpty(t, id.String())
	})

	// Test IsEmpty
	t.Run("IsEmpty", func(t *testing.T) {
		id := ID{}
		assert.True(t, id.IsEmpty())
	})

	// Test ImplementsGraphQLType
	t.Run("ImplementsGraphQLType", func(t *testing.T) {
		id := NewID()
		assert.True(t, id.ImplementsGraphQLType("ID"))
	})

	// Test UnmarshalGraphQL
	t.Run("UnmarshalGraphQL", func(t *testing.T) {
		id := ID{}
		err := id.UnmarshalGraphQL(idStr1)
		assert.NoError(t, err)
	})
}
