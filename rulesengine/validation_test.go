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

func TestEngineValidate_StateErrors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(e *Engine)
		rule    *Rule
		wantMsg string
	}{
		{
			name:  "undefined fact",
			setup: func(e *Engine) {},
			rule: NewRule(
				Condition{Fact: "missing", Operator: "equal", Value: 1},
				Event{Type: "test"},
			),
			wantMsg: "undefined fact: missing",
		},
		{
			name: "undefined operator",
			setup: func(e *Engine) {
				e.AddFact("x", 1)
			},
			rule: NewRule(
				Condition{Fact: "x", Operator: "customOp", Value: 1},
				Event{Type: "test"},
			),
			wantMsg: "undefined operator: customOp",
		},
		{
			name: "undefined decorator",
			setup: func(e *Engine) {
				e.AddFact("x", 1)
			},
			rule: NewRule(
				Condition{Fact: "x", Operator: "myDec:equal", Value: 1},
				Event{Type: "test"},
			),
			wantMsg: "undefined operator decorator: myDec",
		},
		{
			name:  "undefined condition ref",
			setup: func(e *Engine) {},
			rule: NewRule(
				Condition{ConditionRef: "nonexistent"},
				Event{Type: "test"},
			),
			wantMsg: "undefined condition reference: nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine()
			tt.setup(engine)
			err := engine.AddRule(tt.rule)
			require.NoError(t, err)
			errs := engine.Validate()
			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, tt.wantMsg) {
					found = true
					break
				}
			}
			assert.True(t, found, "expected error containing %q, got: %v", tt.wantMsg, errs)
		})
	}
}

func TestEngineValidate_AllowUndefinedFacts(t *testing.T) {
	engine := NewEngine()
	engine.allowUndefinedFacts = true
	err := engine.AddRule(NewRule(
		Condition{Fact: "missing", Operator: "equal", Value: 1},
		Event{Type: "test"},
	))
	require.NoError(t, err)
	errs := engine.Validate()
	assert.Empty(t, errs)
}

func TestEngineValidate_AllowUndefinedConditions(t *testing.T) {
	engine := NewEngine()
	engine.allowUndefinedConditions = true
	err := engine.AddRule(NewRule(
		Condition{ConditionRef: "nonexistent"},
		Event{Type: "test"},
	))
	require.NoError(t, err)
	errs := engine.Validate()
	assert.Empty(t, errs)
}

func TestEngineValidate_ValidEngine(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("age", 18)
	err := engine.AddRule(NewRule(
		Condition{Fact: "age", Operator: "gte", Value: 18},
		Event{Type: "adult"},
	))
	require.NoError(t, err)
	errs := engine.Validate()
	assert.Empty(t, errs)
}

func TestAddRule_InvalidCondition(t *testing.T) {
	engine := NewEngine()
	rule := NewRule(
		Condition{},
		Event{Type: "test"},
	)
	err := engine.AddRule(rule)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid rule conditions")
	assert.Empty(t, engine.rules)
}

func TestAddRule_ValidCondition(t *testing.T) {
	engine := NewEngine()
	rule := NewRule(
		Condition{Fact: "x", Operator: "eq", Value: 1},
		Event{Type: "test"},
	)
	err := engine.AddRule(rule)
	require.NoError(t, err)
	assert.Len(t, engine.rules, 1)
}
