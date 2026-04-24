package rulesengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeafCondition_Operators(t *testing.T) {
	tests := []struct {
		name     string
		fact     interface{}
		operator string
		value    interface{}
		want     bool
	}{
		{"equal int match", 10, "equal", 10, true},
		{"equal int mismatch", 10, "equal", 20, false},
		{"equal float64 match", 3.14, "equal", 3.14, true},
		{"equal float64 mismatch", 3.14, "equal", 2.71, false},
		{"equal string match", "hello", "equal", "hello", true},
		{"equal string mismatch", "hello", "equal", "world", false},
		{"notEqual int", 10, "notEqual", 20, true},
		{"notEqual int same", 10, "notEqual", 10, false},
		{"notEqual string", "hello", "notEqual", "world", true},
		{"lessThan int true", 5, "lessThan", 10, true},
		{"lessThan int false", 10, "lessThan", 5, false},
		{"lessThan int equal", 10, "lessThan", 10, false},
		{"lessThan float64", 1.5, "lessThan", 2.5, true},
		{"lessThan string", "apple", "lessThan", "banana", true},
		{"greaterThan int true", 10, "greaterThan", 5, true},
		{"greaterThan int false", 5, "greaterThan", 10, false},
		{"greaterThan float64", 9.9, "greaterThan", 1.1, true},
		{"greaterThan string", "banana", "greaterThan", "apple", true},
		{"lte int less", 5, "lessThanInclusive", 10, true},
		{"lte int equal", 10, "lessThanInclusive", 10, true},
		{"lte int greater", 15, "lessThanInclusive", 10, false},
		{"gte int greater", 10, "greaterThanInclusive", 5, true},
		{"gte int equal", 10, "greaterThanInclusive", 10, true},
		{"gte int less", 5, "greaterThanInclusive", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateCondition(
				Condition{Fact: "x", Operator: tt.operator, Value: tt.value},
				map[string]interface{}{"x": tt.fact},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestLeafCondition_OperatorAliases(t *testing.T) {
	tests := []struct {
		alias    string
		fullName string
	}{
		{"eq", "equal"},
		{"ne", "notEqual"},
		{"lt", "lessThan"},
		{"gt", "greaterThan"},
		{"lte", "lessThanInclusive"},
		{"gte", "greaterThanInclusive"},
	}

	facts := map[string]interface{}{"x": 10}
	value := 20

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			aliasResult, err := EvaluateCondition(
				Condition{Fact: "x", Operator: tt.alias, Value: value},
				facts,
			)
			require.NoError(t, err)

			fullResult, err := EvaluateCondition(
				Condition{Fact: "x", Operator: tt.fullName, Value: value},
				facts,
			)
			require.NoError(t, err)

			assert.Equal(t, fullResult, aliasResult, "%s should behave like %s", tt.alias, tt.fullName)
		})
	}
}

func TestLeafCondition_TypeCoercion(t *testing.T) {
	tests := []struct {
		name     string
		fact     interface{}
		operator string
		value    interface{}
		want     bool
	}{
		{"int vs float64 equal", 100, "equal", 100.0, false},
		{"float64 vs int equal", 100.0, "equal", 100, false},
		{"int vs float64 lessThan", 99, "lessThan", 100.0, true},
		{"float64 vs int greaterThan", 100.5, "greaterThan", 100, true},
		{"string vs int not equal", "100", "equal", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateCondition(
				Condition{Fact: "x", Operator: tt.operator, Value: tt.value},
				map[string]interface{}{"x": tt.fact},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestLeafCondition_NilAndZero(t *testing.T) {
	tests := []struct {
		name     string
		fact     interface{}
		operator string
		value    interface{}
		want     bool
	}{
		{"nil equal nil", nil, "equal", nil, true},
		{"nil equal 0", nil, "equal", 0, false},
		{"0 equal 0", 0, "equal", 0, true},
		{"0 greaterThan 0", 0, "greaterThan", 0, false},
		{"0 lessThan 1", 0, "lessThan", 1, true},
		{"empty string equal empty", "", "equal", "", true},
		{"empty string notEqual nonempty", "", "notEqual", "hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine()
			engine.AddFact("x", tt.fact)
			almanac := NewAlmanac(engine, nil)

			cond := Condition{Fact: "x", Operator: tt.operator, Value: tt.value}
			result, err := cond.Evaluate(almanac, engine)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestLeafCondition_UnknownOperator(t *testing.T) {
	_, err := EvaluateCondition(
		Condition{Fact: "x", Operator: "nonexistent", Value: 1},
		map[string]interface{}{"x": 1},
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undefined operator")
}

func TestLeafCondition_UndefinedFact(t *testing.T) {
	t.Run("default errors", func(t *testing.T) {
		_, err := EvaluateCondition(
			Condition{Fact: "missing", Operator: "equal", Value: 1},
			map[string]interface{}{},
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined fact")
	})

	t.Run("allowUndefinedFacts returns no error", func(t *testing.T) {
		engine := NewEngine()
		engine.allowUndefinedFacts = true
		almanac := NewAlmanac(engine, map[string]interface{}{})

		cond := Condition{Fact: "missing", Operator: "equal", Value: nil}
		result, err := cond.Evaluate(almanac, engine)
		require.NoError(t, err)
		assert.True(t, result)
	})
}

func TestAllCondition(t *testing.T) {
	facts := map[string]interface{}{"a": 10, "b": 20}

	t.Run("all true", func(t *testing.T) {
		result, err := EvaluateCondition(Condition{
			All: []Condition{
				{Fact: "a", Operator: "equal", Value: 10},
				{Fact: "b", Operator: "equal", Value: 20},
			},
		}, facts)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("one false", func(t *testing.T) {
		result, err := EvaluateCondition(Condition{
			All: []Condition{
				{Fact: "a", Operator: "equal", Value: 10},
				{Fact: "b", Operator: "equal", Value: 999},
			},
		}, facts)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("empty children", func(t *testing.T) {
		cond := Condition{All: []Condition{}}
		engine := NewEngine()
		almanac := NewAlmanac(engine, facts)
		_, err := cond.Evaluate(almanac, engine)
		assert.Error(t, err)
	})
}

func TestAnyCondition(t *testing.T) {
	facts := map[string]interface{}{"a": 10, "b": 20}

	t.Run("one true", func(t *testing.T) {
		result, err := EvaluateCondition(Condition{
			Any: []Condition{
				{Fact: "a", Operator: "equal", Value: 999},
				{Fact: "b", Operator: "equal", Value: 20},
			},
		}, facts)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("all false", func(t *testing.T) {
		result, err := EvaluateCondition(Condition{
			Any: []Condition{
				{Fact: "a", Operator: "equal", Value: 999},
				{Fact: "b", Operator: "equal", Value: 999},
			},
		}, facts)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("empty children", func(t *testing.T) {
		cond := Condition{Any: []Condition{}}
		engine := NewEngine()
		almanac := NewAlmanac(engine, facts)
		_, err := cond.Evaluate(almanac, engine)
		assert.Error(t, err)
	})
}

func TestNotCondition(t *testing.T) {
	facts := map[string]interface{}{"x": 10}

	t.Run("negates true to false", func(t *testing.T) {
		result, err := EvaluateCondition(Condition{
			Not: &Condition{Fact: "x", Operator: "equal", Value: 10},
		}, facts)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("negates false to true", func(t *testing.T) {
		result, err := EvaluateCondition(Condition{
			Not: &Condition{Fact: "x", Operator: "equal", Value: 999},
		}, facts)
		require.NoError(t, err)
		assert.True(t, result)
	})
}

func TestNestedConditions(t *testing.T) {
	facts := map[string]interface{}{"a": 10, "b": 20, "c": 30}

	t.Run("All[Any[leaf,leaf], leaf]", func(t *testing.T) {
		result, err := EvaluateCondition(Condition{
			All: []Condition{
				{Any: []Condition{
					{Fact: "a", Operator: "equal", Value: 999},
					{Fact: "b", Operator: "equal", Value: 20},
				}},
				{Fact: "c", Operator: "equal", Value: 30},
			},
		}, facts)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("Any[All[leaf,leaf], Not[leaf]]", func(t *testing.T) {
		result, err := EvaluateCondition(Condition{
			Any: []Condition{
				{All: []Condition{
					{Fact: "a", Operator: "equal", Value: 999},
					{Fact: "b", Operator: "equal", Value: 20},
				}},
				{Not: &Condition{Fact: "c", Operator: "equal", Value: 30}},
			},
		}, facts)
		require.NoError(t, err)
		assert.False(t, result)
	})
}

func TestOperator_In(t *testing.T) {
	tests := []struct {
		name  string
		fact  interface{}
		value interface{}
		want  bool
	}{
		{"int in slice", 2, []interface{}{1, 2, 3}, true},
		{"int not in slice", 4, []interface{}{1, 2, 3}, false},
		{"string in slice", "b", []interface{}{"a", "b", "c"}, true},
		{"empty slice", 1, []interface{}{}, false},
		{"non-slice condition", 1, "not-a-slice", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateCondition(
				Condition{Fact: "x", Operator: "in", Value: tt.value},
				map[string]interface{}{"x": tt.fact},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOperator_NotIn(t *testing.T) {
	tests := []struct {
		name  string
		fact  interface{}
		value interface{}
		want  bool
	}{
		{"int not in slice", 4, []interface{}{1, 2, 3}, true},
		{"int in slice", 2, []interface{}{1, 2, 3}, false},
		{"non-slice condition", 1, "not-a-slice", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateCondition(
				Condition{Fact: "x", Operator: "notIn", Value: tt.value},
				map[string]interface{}{"x": tt.fact},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOperator_Contains_String(t *testing.T) {
	tests := []struct {
		name  string
		fact  interface{}
		value interface{}
		want  bool
	}{
		{"substring found", "hello world", "world", true},
		{"substring not found", "hello world", "xyz", false},
		{"empty substring", "hello", "", true},
		{"empty string fact", "", "hello", false},
		{"non-string condition value", "hello", 123, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateCondition(
				Condition{Fact: "x", Operator: "contains", Value: tt.value},
				map[string]interface{}{"x": tt.fact},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOperator_Contains_Slice(t *testing.T) {
	tests := []struct {
		name  string
		fact  interface{}
		value interface{}
		want  bool
	}{
		{"element in slice", []interface{}{"a", "b", "c"}, "b", true},
		{"element not in slice", []interface{}{"a", "b", "c"}, "d", false},
		{"int element in slice", []interface{}{1, 2, 3}, 2, true},
		{"type mismatch element", []interface{}{1, 2, 3}, "2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateCondition(
				Condition{Fact: "x", Operator: "contains", Value: tt.value},
				map[string]interface{}{"x": tt.fact},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOperator_DoesNotContain(t *testing.T) {
	tests := []struct {
		name  string
		fact  interface{}
		value interface{}
		want  bool
	}{
		{"string without substring", "hello", "xyz", true},
		{"string with substring", "hello world", "world", false},
		{"slice without element", []interface{}{1, 2}, 3, true},
		{"slice with element", []interface{}{1, 2, 3}, 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateCondition(
				Condition{Fact: "x", Operator: "doesNotContain", Value: tt.value},
				map[string]interface{}{"x": tt.fact},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOperator_Matches(t *testing.T) {
	tests := []struct {
		name  string
		fact  interface{}
		value interface{}
		want  bool
	}{
		{"valid match", "user@example.com", `^[a-z]+@[a-z]+\.[a-z]+$`, true},
		{"no match", "invalid", `^[0-9]+$`, false},
		{"invalid regex", "test", `[invalid`, false},
		{"non-string fact", 123, `^[0-9]+$`, false},
		{"non-string pattern", "test", 123, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateCondition(
				Condition{Fact: "x", Operator: "matches", Value: tt.value},
				map[string]interface{}{"x": tt.fact},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDeepNesting(t *testing.T) {
	facts := map[string]interface{}{"x": 5}

	leaf := Condition{Fact: "x", Operator: "equal", Value: 5}
	cond := leaf
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			cond = Condition{Any: []Condition{cond}}
		} else {
			cond = Condition{All: []Condition{cond}}
		}
	}

	result, err := EvaluateCondition(cond, facts)
	require.NoError(t, err)
	assert.True(t, result)

	depth := MaxDepth(&cond)
	assert.Equal(t, 7, depth)
}
