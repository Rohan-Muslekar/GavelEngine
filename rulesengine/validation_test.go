package rulesengine

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCondition_Valid(t *testing.T) {
	tests := []struct {
		name      string
		condition *Condition
	}{
		{"leaf", &Condition{Fact: "age", Operator: "gte", Value: 18}},
		{"All", &Condition{All: []Condition{{Fact: "x", Operator: "eq", Value: 1}}}},
		{"Any", &Condition{Any: []Condition{{Fact: "x", Operator: "eq", Value: 1}}}},
		{"Not", &Condition{Not: &Condition{Fact: "x", Operator: "eq", Value: 1}}},
		{"ConditionRef", &Condition{ConditionRef: "my-condition"}},
		{"nested All/Any", &Condition{All: []Condition{
			{Any: []Condition{{Fact: "x", Operator: "eq", Value: 1}}},
		}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateCondition(tt.condition)
			assert.Empty(t, errs)
		})
	}
}

func TestValidateCondition_Errors(t *testing.T) {
	tests := []struct {
		name      string
		condition *Condition
		wantPath  string
		wantMsg   string
	}{
		{
			name:      "empty node",
			condition: &Condition{},
			wantPath:  "",
			wantMsg:   "empty",
		},
		{
			name:      "missing Fact",
			condition: &Condition{Operator: "eq"},
			wantPath:  "",
			wantMsg:   "missing Fact",
		},
		{
			name:      "missing Operator",
			condition: &Condition{Fact: "x"},
			wantPath:  "",
			wantMsg:   "missing Operator",
		},
		{
			name:      "empty All",
			condition: &Condition{All: []Condition{}},
			wantPath:  "",
			wantMsg:   "at least one child",
		},
		{
			name:      "empty Any",
			condition: &Condition{Any: []Condition{}},
			wantPath:  "",
			wantMsg:   "at least one child",
		},
		{
			name: "mixed compound and leaf",
			condition: &Condition{
				All:      []Condition{{Fact: "x", Operator: "eq", Value: 1}},
				Fact:     "y",
				Operator: "eq",
			},
			wantPath: "",
			wantMsg:  "multiple types",
		},
		{
			name: "nested error path All.Any",
			condition: &Condition{
				All: []Condition{
					{Fact: "x", Operator: "eq", Value: 1},
					{Any: []Condition{{Operator: "eq"}}},
				},
			},
			wantPath: "All[1].Any[0]",
			wantMsg:  "missing Fact",
		},
		{
			name: "nested error path All.Not",
			condition: &Condition{
				All: []Condition{
					{Not: &Condition{}},
				},
			},
			wantPath: "All[0].Not",
			wantMsg:  "empty",
		},
		{
			name:      "nil condition",
			condition: nil,
			wantPath:  "",
			wantMsg:   "nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateCondition(tt.condition)
			require.NotEmpty(t, errs, "expected validation errors")
			found := false
			for _, e := range errs {
				pathMatch := tt.wantPath == "" || e.Path == tt.wantPath
				msgMatch := strings.Contains(e.Message, tt.wantMsg)
				if pathMatch && msgMatch {
					found = true
					break
				}
			}
			assert.True(t, found, "expected error with path=%q containing %q, got: %v", tt.wantPath, tt.wantMsg, errs)
		})
	}
}
