package rulesengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateWithTrace_Leaf(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("age", 25)
	almanac := NewAlmanac(engine, nil)

	cond := &Condition{Fact: "age", Operator: "gte", Value: 18}
	result, trace, err := cond.EvaluateWithTrace(almanac, engine)
	require.NoError(t, err)

	assert.True(t, result)
	assert.True(t, trace.Result)
	assert.Equal(t, 25, trace.FactValue)
	assert.Nil(t, trace.Children)
	assert.Equal(t, "age", trace.Condition.Fact)
	assert.Equal(t, "gte", trace.Condition.Operator)
}

func TestEvaluateWithTrace_AllShortCircuit(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("x", 5)
	almanac := NewAlmanac(engine, nil)

	cond := &Condition{
		All: []Condition{
			{Fact: "x", Operator: "gte", Value: 1},
			{Fact: "x", Operator: "gte", Value: 10},
			{Fact: "x", Operator: "gte", Value: 3},
		},
	}
	result, trace, err := cond.EvaluateWithTrace(almanac, engine)
	require.NoError(t, err)

	assert.False(t, result)
	assert.False(t, trace.Result)
	require.Len(t, trace.Children, 2)
	assert.True(t, trace.Children[0].Result)
	assert.False(t, trace.Children[1].Result)
}

func TestEvaluateWithTrace_AnyShortCircuit(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("x", 5)
	almanac := NewAlmanac(engine, nil)

	cond := &Condition{
		Any: []Condition{
			{Fact: "x", Operator: "gte", Value: 10},
			{Fact: "x", Operator: "gte", Value: 3},
			{Fact: "x", Operator: "gte", Value: 1},
		},
	}
	result, trace, err := cond.EvaluateWithTrace(almanac, engine)
	require.NoError(t, err)

	assert.True(t, result)
	assert.True(t, trace.Result)
	require.Len(t, trace.Children, 2)
	assert.False(t, trace.Children[0].Result)
	assert.True(t, trace.Children[1].Result)
}

func TestEvaluateWithTrace_Not(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("x", 5)
	almanac := NewAlmanac(engine, nil)

	cond := &Condition{
		Not: &Condition{Fact: "x", Operator: "gte", Value: 10},
	}
	result, trace, err := cond.EvaluateWithTrace(almanac, engine)
	require.NoError(t, err)

	assert.True(t, result)
	assert.True(t, trace.Result)
	require.Len(t, trace.Children, 1)
	assert.False(t, trace.Children[0].Result)
	assert.Equal(t, 5, trace.Children[0].FactValue)
}

func TestEvaluateWithTrace_ConditionRef(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("age", 25)
	engine.SetCondition("isAdult", Condition{Fact: "age", Operator: "gte", Value: 18})
	almanac := NewAlmanac(engine, nil)

	cond := &Condition{ConditionRef: "isAdult"}
	result, trace, err := cond.EvaluateWithTrace(almanac, engine)
	require.NoError(t, err)

	assert.True(t, result)
	assert.True(t, trace.Result)
	require.Len(t, trace.Children, 1)
	assert.Equal(t, 25, trace.Children[0].FactValue)
}

func TestRun_WithTrace(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("age", 25)
	engine.AddRule(NewRule(
		Condition{Fact: "age", Operator: "gte", Value: 18},
		Event{Type: "adult"},
		WithName("age-check"),
	))

	result, err := engine.Run(nil, WithTrace())
	require.NoError(t, err)
	require.Len(t, result.RuleResults, 1)
	require.NotNil(t, result.RuleResults[0].Trace)
	assert.True(t, result.RuleResults[0].Trace.Result)
	assert.Equal(t, 25, result.RuleResults[0].Trace.FactValue)
	assert.Equal(t, "age", result.RuleResults[0].Trace.Condition.Fact)
}

func TestRun_WithoutTrace(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("age", 25)
	engine.AddRule(NewRule(
		Condition{Fact: "age", Operator: "gte", Value: 18},
		Event{Type: "adult"},
		WithName("age-check"),
	))

	result, err := engine.Run(nil)
	require.NoError(t, err)
	require.Len(t, result.RuleResults, 1)
	assert.Nil(t, result.RuleResults[0].Trace)
}
