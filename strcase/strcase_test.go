package strcase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSnake(t *testing.T) {

	tests := []struct {
		name string
		s    string
		want string
	}{
		{"PastOrganizationsIDs", "PastOrganizationsIDs", "past_organizations_ids"},
		{"Snaked_CamelCase", "Snaked___CamelCase", "snaked__camel_case"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSnake(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToSnakedCamel(t *testing.T) {

	tests := []struct {
		name string
		s    string
		want string
	}{
		{"Underscores within CamelCase", "past___organizations_ids", "Past_OrganizationsIDs"},
		{"Underscores within CamelCase - 2", "drug___name", "Drug_Name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSnakedCamel(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpperizeAcronym(t *testing.T) {

	tests := []struct {
		name string
		s    string
		want string
	}{
		// TODO: Add test cases.
		{"Simple", "id", "ID"},
		{"Suffix", "org_id", "org_ID"},
		{"Prefix", "id_org", "ID_org"},
		{"Middle", "org_id_org", "org_id_org"},
		{"Plural", "org_ids", "org_IDs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UpperizeAcronym(tt.s); got != tt.want {
				t.Errorf("UpperizeAcronym() = %v, want %v", got, tt.want)
			}
		})
	}
}
