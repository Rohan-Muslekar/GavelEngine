package rulesengine

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_SingleRule(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("age", 22)

	rule := NewRule(
		Condition{Fact: "age", Operator: "greaterThanInclusive", Value: 18},
		Event{Type: "adult", Params: map[string]interface{}{"msg": "is adult"}},
		WithName("age-check"),
	)
	engine.AddRule(rule)

	result, err := engine.Run(nil)
	require.NoError(t, err)
	require.Len(t, result.Events, 1)
	assert.Equal(t, "adult", result.Events[0].Type)
	require.Len(t, result.RuleResults, 1)
	assert.True(t, result.RuleResults[0].Success)
}

func TestEngine_MultipleRules_PriorityOrder(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("x", 1)

	var order []string
	for _, p := range []int{1, 10, 5} {
		name := fmt.Sprintf("rule-p%d", p)
		n := name
		rule := NewRule(
			Condition{Fact: "x", Operator: "equal", Value: 1},
			Event{Type: n},
			WithName(n),
			WithPriorityForRule(p),
			WithOnSuccess(func(event Event, almanac *Almanac, rr *RuleResult) error {
				order = append(order, n)
				return nil
			}),
		)
		engine.AddRule(rule)
	}

	_, err := engine.Run(nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"rule-p10", "rule-p5", "rule-p1"}, order)
}

func TestEngine_RuleFailure(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("x", 1)

	failureCalled := false
	rule := NewRule(
		Condition{Fact: "x", Operator: "equal", Value: 999},
		Event{Type: "never"},
		WithName("fail-rule"),
		WithOnFailure(func(event Event, almanac *Almanac, rr *RuleResult) error {
			failureCalled = true
			return nil
		}),
	)
	engine.AddRule(rule)

	result, err := engine.Run(nil)
	require.NoError(t, err)
	assert.Empty(t, result.Events)
	require.Len(t, result.RuleResults, 1)
	assert.False(t, result.RuleResults[0].Success)
	assert.True(t, failureCalled)
}

func TestEngine_RuleChaining(t *testing.T) {
	engine := NewEngine()

	rule1 := NewRule(
		Condition{Fact: "flag", Operator: "equal", Value: true},
		Event{Type: "rule1-event"},
		WithName("rule1"),
		WithPriorityForRule(10),
		WithOnSuccess(func(event Event, almanac *Almanac, rr *RuleResult) error {
			almanac.AddRuntimeFact("rule1Passed", true)
			return nil
		}),
	)
	engine.AddRule(rule1)

	rule2 := NewRule(
		Condition{Fact: "rule1Passed", Operator: "equal", Value: true},
		Event{Type: "rule2-event"},
		WithName("rule2"),
		WithPriorityForRule(1),
	)
	engine.AddRule(rule2)

	result, err := engine.Run(map[string]interface{}{"flag": true})
	require.NoError(t, err)
	assert.Len(t, result.Events, 2)
	assert.Equal(t, "rule1-event", result.Events[0].Type)
	assert.Equal(t, "rule2-event", result.Events[1].Type)
}

func TestEngine_EventEmission(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("x", 1)

	rule := NewRule(
		Condition{Fact: "x", Operator: "equal", Value: 1},
		Event{Type: "match", Params: map[string]interface{}{"detail": "found"}},
		WithName("emit-rule"),
	)
	engine.AddRule(rule)

	result, err := engine.Run(nil)
	require.NoError(t, err)
	require.Len(t, result.Events, 1)
	assert.Equal(t, "match", result.Events[0].Type)
	assert.Equal(t, "found", result.Events[0].Params["detail"])
}

func TestEngine_MultipleFactsInCondition(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("a", 10)
	engine.AddFact("b", 20)
	engine.AddFact("c", 30)

	rule := NewRule(
		Condition{All: []Condition{
			{Fact: "a", Operator: "equal", Value: 10},
			{Fact: "b", Operator: "equal", Value: 20},
			{Fact: "c", Operator: "equal", Value: 30},
		}},
		Event{Type: "all-match"},
		WithName("multi-fact"),
	)
	engine.AddRule(rule)

	result, err := engine.Run(nil)
	require.NoError(t, err)
	assert.Len(t, result.Events, 1)
}

func TestEngine_Stop(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("x", 1)

	rule1 := NewRule(
		Condition{Fact: "x", Operator: "equal", Value: 1},
		Event{Type: "first"},
		WithName("rule1"),
		WithPriorityForRule(10),
		WithOnSuccess(func(event Event, almanac *Almanac, rr *RuleResult) error {
			engine.Stop()
			return nil
		}),
	)
	rule2 := NewRule(
		Condition{Fact: "x", Operator: "equal", Value: 1},
		Event{Type: "second"},
		WithName("rule2"),
		WithPriorityForRule(1),
	)
	engine.AddRule(rule1)
	engine.AddRule(rule2)

	result, err := engine.Run(nil)
	require.NoError(t, err)
	assert.Len(t, result.Events, 1)
	assert.Equal(t, "first", result.Events[0].Type)
}

func TestEngine_CustomOperator(t *testing.T) {
	engine := NewEngine()
	engine.AddOperator("contains", func(factValue interface{}, conditionValue interface{}) bool {
		rv := reflect.ValueOf(factValue)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return false
		}
		for i := 0; i < rv.Len(); i++ {
			if reflect.DeepEqual(rv.Index(i).Interface(), conditionValue) {
				return true
			}
		}
		return false
	})

	engine.AddFact("colors", []interface{}{"red", "green", "blue"})
	rule := NewRule(
		Condition{Fact: "colors", Operator: "contains", Value: "blue"},
		Event{Type: "found-blue"},
		WithName("contains-rule"),
	)
	engine.AddRule(rule)

	result, err := engine.Run(nil)
	require.NoError(t, err)
	require.Len(t, result.Events, 1)
	assert.Equal(t, "found-blue", result.Events[0].Type)
}

func TestEngine_OperatorDecorator(t *testing.T) {
	engine := NewEngine()
	engine.AddOperatorDecorator("caseInsensitive", func(factValue interface{}, conditionValue interface{}, next OperatorFunc) bool {
		s1, ok1 := factValue.(string)
		s2, ok2 := conditionValue.(string)
		if ok1 && ok2 {
			return next(strings.ToLower(s1), strings.ToLower(s2))
		}
		return next(factValue, conditionValue)
	})

	engine.AddFact("username", "Alice")
	rule := NewRule(
		Condition{Fact: "username", Operator: "caseInsensitive:equal", Value: "alice"},
		Event{Type: "welcome"},
		WithName("ci-rule"),
	)
	engine.AddRule(rule)

	result, err := engine.Run(nil)
	require.NoError(t, err)
	require.Len(t, result.Events, 1)
	assert.Equal(t, "welcome", result.Events[0].Type)
}

func TestEngine_ConditionRef(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("age", 25)

	engine.SetCondition("is-adult", Condition{
		Fact: "age", Operator: "greaterThanInclusive", Value: 18,
	})

	rule := NewRule(
		Condition{ConditionRef: "is-adult"},
		Event{Type: "adult-event"},
		WithName("ref-rule"),
	)
	engine.AddRule(rule)

	result, err := engine.Run(nil)
	require.NoError(t, err)
	require.Len(t, result.Events, 1)
	assert.Equal(t, "adult-event", result.Events[0].Type)
}

func TestEvaluateCondition(t *testing.T) {
	t.Run("simple true", func(t *testing.T) {
		result, err := EvaluateCondition(
			Condition{Fact: "age", Operator: "gt", Value: 18},
			map[string]interface{}{"age": 25},
		)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("simple false", func(t *testing.T) {
		result, err := EvaluateCondition(
			Condition{Fact: "age", Operator: "gt", Value: 18},
			map[string]interface{}{"age": 10},
		)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("nested condition", func(t *testing.T) {
		result, err := EvaluateCondition(
			Condition{All: []Condition{
				{Fact: "age", Operator: "gt", Value: 18},
				{Fact: "score", Operator: "lt", Value: 50},
			}},
			map[string]interface{}{"age": 25, "score": 30},
		)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("operator alias", func(t *testing.T) {
		result, err := EvaluateCondition(
			Condition{Fact: "x", Operator: "lte", Value: 100},
			map[string]interface{}{"x": 100},
		)
		require.NoError(t, err)
		assert.True(t, result)
	})
}

func TestEvaluateCondition_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	errors := make([]error, 10)
	results := make([]bool, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			val := idx + 1
			result, err := EvaluateCondition(
				Condition{Fact: "x", Operator: "greaterThan", Value: 0},
				map[string]interface{}{"x": val},
			)
			errors[idx] = err
			results[idx] = result
		}(i)
	}
	wg.Wait()

	for i := 0; i < 10; i++ {
		assert.NoError(t, errors[i], "goroutine %d", i)
		assert.True(t, results[i], "goroutine %d", i)
	}
}
