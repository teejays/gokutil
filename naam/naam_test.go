package naam

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewName(t *testing.T) {

	tests := []struct {
		name  string
		words string
		want  Name
	}{
		{
			name:  "1",
			words: "alpha_bravo",
			want:  Name{words: "alpha_bravo"},
		},
		{
			name:  "2",
			words: "alphaBravo",
			want:  Name{words: "alpha_bravo"},
		},
		{
			name:  "3",
			words: "AlphaBravo",
			want:  Name{words: "alpha_bravo"},
		},
		{
			name:  "4",
			words: "Alpha_bravo",
			want:  Name{words: "alpha_bravo"},
		},
		{
			name:  "5",
			words: "AlphaBravo_Charlie",
			want:  Name{words: "alpha_bravo_charlie"},
		},
		{
			name:  "6",
			words: "PastOrganizationsIDs",
			want:  Name{words: "past_organizations_ids"},
		},
		{
			name:  "7",
			words: "user__x__name",
			want:  Name{words: "user__x__name"},
		},
		{
			name:  "8",
			words: "PersonName",
			want:  Name{words: "person_name"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.words)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCamel(t *testing.T) {
	tests := []struct {
		name string
		in   Name
		want Name
	}{
		{
			name: "1",
			in:   New("person_name"),
			want: New("person_names"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.Pluralize()
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestPluralize(t *testing.T) {
	tests := []struct {
		name string
		in   Name
		want Name
	}{
		{
			name: "1",
			in:   New("person_name"),
			want: New("person_names"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.Pluralize()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTrimThenPluralize(t *testing.T) {
	tests := []struct {
		name    string
		in      Name
		trimStr string
		want1   Name
		want2   Name
	}{
		{
			name:    "1",
			in:      New("person_name_with_meta"),
			trimStr: "with_meta",
			want1:   New("person_name"),
			want2:   New("person_names"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1 := tt.in.TrimSuffixString(tt.trimStr)
			require.Equal(t, tt.want1, got1, "Comparison 1")
			got2 := got1.Pluralize()
			require.Equal(t, tt.want2, got2, "Comparison 2")
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		want      Name
		wantErr   bool
		matchErr  error
		wantPanic bool
	}{
		{
			name:    "Sample Test",
			data:    []byte(`Sample App`),
			want:    Name{words: "sample_app"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n Name
			err := n.UnmarshalJSON(tt.data)
			if tt.wantErr {
				if assert.Error(t, err) && tt.matchErr != nil && errors.Is(err, tt.matchErr) {
					assert.FailNowf(t, "Received Error does not match expected Error", "expected: %v, got: %v", tt.matchErr, err)
				}
				return
			}
			if assert.NoError(t, err) {
				assert.Equal(t, tt.want, n)
			}
		})
	}
}
