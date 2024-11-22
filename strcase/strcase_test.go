package strcase

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestCase struct {
	Input      string
	Plural     string
	PascalCase string
	CamelCase  string
	SnakeCase  string
	KebabCase  string
	TitleCase  string
}

var testCases = []TestCase{
	{
		Input:      "id",
		Plural:     "ids",
		PascalCase: "ID",
		CamelCase:  "id",
		SnakeCase:  "id",
		KebabCase:  "id",
		TitleCase:  "ID",
	},
	{
		Input:      "ids",
		Plural:     "ids",
		PascalCase: "IDs",
		CamelCase:  "ids",
		SnakeCase:  "ids",
		KebabCase:  "ids",
		TitleCase:  "IDs",
	},
	{
		Input:      "first_id",
		Plural:     "first_ids",
		PascalCase: "FirstID",
		CamelCase:  "firstID",
		SnakeCase:  "first_id",
		KebabCase:  "first-id",
		TitleCase:  "First ID",
	},
	{
		Input:      "first_ids",
		Plural:     "first_ids",
		PascalCase: "FirstIDs",
		CamelCase:  "firstIDs",
		SnakeCase:  "first_ids",
		KebabCase:  "first-ids",
		TitleCase:  "First IDs",
	},
	{
		Input:      "first_id_last",
		Plural:     "first_id_lasts",
		PascalCase: "FirstIDLast",
		CamelCase:  "firstIDLast",
		SnakeCase:  "first_id_last",
		KebabCase:  "first-id-last",
		TitleCase:  "First ID Last",
	},
	{
		Input:      "first_ids_last",
		Plural:     "first_ids_lasts",
		PascalCase: "FirstIDsLast",
		CamelCase:  "firstIDsLast",
		SnakeCase:  "first_ids_last",
		KebabCase:  "first-ids-last",
		TitleCase:  "First IDs Last",
	},
	{
		Input:      "parent___first_ids_last",
		Plural:     "parent___first_ids_lasts",
		PascalCase: "Parent_FirstIDsLast",
		CamelCase:  "parent_firstIDsLast",
		SnakeCase:  "parent__first_ids_last",
		KebabCase:  "parent--first-ids-last",
		TitleCase:  "Parent - First IDs Last",
	},
	{
		Input:      "common",
		Plural:     "commons",
		PascalCase: "Common",
		CamelCase:  "common",
		SnakeCase:  "common",
		KebabCase:  "common",
		TitleCase:  "Common",
	},
	// {
	// 	Input:      `"common"`,
	// 	Plural:     `"commons"`,
	// 	PascalCase: "Common",
	// 	CamelCase:  "common",
	// 	SnakeCase:  "common",
	// 	KebabCase:  "common",
	// 	TitleCase:  "Common",
	// },
	{
		Input:      "blah json",
		Plural:     "blah jsons",
		PascalCase: "BlahJSON",
		CamelCase:  "blahJSON",
		SnakeCase:  "blah_json",
		KebabCase:  "blah-json",
		TitleCase:  "Blah JSON",
	},
}

func TestMain(t *testing.T) {
	for _, tc := range testCases {
		t.Run("Pluralize: "+tc.Input, func(t *testing.T) {
			assert.Equalf(t, tc.Plural, MustPluralize(tc.Input), "Input: %s", tc.Input)
		})
		t.Run("PascalCase: "+tc.Input, func(t *testing.T) {
			assert.Equalf(t, tc.PascalCase, ToPascal(tc.Input), "Input: %s", tc.Input)
		})
		t.Run("CamelCase: "+tc.Input, func(t *testing.T) {
			assert.Equalf(t, tc.CamelCase, ToCamel(tc.Input), "Input: %s", tc.Input)
		})
		t.Run("SnakeCase: "+tc.Input, func(t *testing.T) {
			assert.Equalf(t, tc.SnakeCase, ToSnake(tc.Input), "Input: %s", tc.Input)
		})
		t.Run("KebabCase: "+tc.Input, func(t *testing.T) {
			assert.Equalf(t, tc.KebabCase, ToKebab(tc.Input), "Input: %s", tc.Input)
		})
		t.Run("TitleCase: "+tc.Input, func(t *testing.T) {
			assert.Equalf(t, tc.TitleCase, ToTitle(tc.Input), "Input: %s", tc.Input)
		})
	}
}

func TestUpperizeAcronymWord(t *testing.T) {

	tests := []struct {
		s    string
		want string
	}{
		// TODO: Add test cases.
		{"id", "ID"},
		{"ids", "IDs"},
		{"sid", "sid"},
		{"sids", "sids"},
		{"sid_", "sid_"},
		{"_sid", "_sid"},
		{"_sid_", "_sid_"},
		{"https", "HTTPS"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := UpperizeAcronymWord(tt.s); got != tt.want {
				t.Errorf("got = %s, want = %s", got, tt.want)
			}
		})
	}
}

func TestUpperizeAcronyms(t *testing.T) {

	tests := []struct {
		s    string
		want string
	}{
		// TODO: Add test cases.
		{"id", "ID"},
		{"first_id", "first_ID"},
		{"first_id_last", "first_ID_last"},
		{"id_last", "ID_last"},
		{"first_ids", "first_IDs"},
		{"first_ids_last", "first_IDs_last"},
		{"ids_last", "IDs_last"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := UpperizeAcronyms(tt.s, "_"); got != tt.want {
				t.Errorf("got = %s, want = %s", got, tt.want)
			}
		})
	}
}

func TestToPascalWord(t *testing.T) {

	tests := []struct {
		s    string
		want string
	}{
		{"first", "First"},
		{"id", "ID"},
		{"ids", "IDs"},
		{"maids", "Maids"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test #%d", i), func(t *testing.T) {
			got := ToPascalWord(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test Pluralize
func TestPluralize(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{"id", "ids"},
		{"org_id", "org_ids"},
		{"id_org", "id_orgs"},
		{"hello_sheep", "hello_sheep"},
		{"some_process", "some_processes"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := MustPluralize(tt.s); got != tt.want {
				t.Errorf("got = %s, want %s", got, tt.want)
			}
		})
	}
}
