package rulesengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON_RoundTrip(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("age", 18)
	engine.AddRule(NewRule(
		Condition{Fact: "age", Operator: "gte", Value: 18},
		Event{Type: "adult", Params: map[string]interface{}{"msg": "welcome"}},
		WithName("age-check"),
		WithPriorityForRule(10),
	))

	data, err := engine.ExportRulesJSON()
	require.NoError(t, err)

	rules, err := LoadRulesFromJSON(data)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "age-check", rules[0].Name)
	assert.Equal(t, 10, rules[0].Priority)
	assert.Equal(t, "age", rules[0].Conditions.Fact)
	assert.Equal(t, "gte", rules[0].Conditions.Operator)
	assert.Equal(t, float64(18), rules[0].Conditions.Value)
	assert.Equal(t, "adult", rules[0].Event.Type)
	assert.Equal(t, "welcome", rules[0].Event.Params["msg"])
}

func TestYAML_RoundTrip(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("age", 18)
	engine.AddRule(NewRule(
		Condition{Fact: "age", Operator: "gte", Value: 18},
		Event{Type: "adult", Params: map[string]interface{}{"msg": "welcome"}},
		WithName("age-check"),
		WithPriorityForRule(10),
	))

	data, err := engine.ExportRulesYAML()
	require.NoError(t, err)

	rules, err := LoadRulesFromYAML(data)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "age-check", rules[0].Name)
	assert.Equal(t, 10, rules[0].Priority)
	assert.Equal(t, "age", rules[0].Conditions.Fact)
	assert.Equal(t, "gte", rules[0].Conditions.Operator)
	assert.Equal(t, 18, rules[0].Conditions.Value)
	assert.Equal(t, "adult", rules[0].Event.Type)
	assert.Equal(t, "welcome", rules[0].Event.Params["msg"])
}

func TestLoadRulesFromJSON_InvalidInput(t *testing.T) {
	_, err := LoadRulesFromJSON([]byte("not json"))
	require.Error(t, err)
}

func TestLoadRulesFromYAML_InvalidInput(t *testing.T) {
	_, err := LoadRulesFromYAML([]byte("{{{"))
	require.Error(t, err)
}

func TestLoadRulesFromJSON_EmptyArray(t *testing.T) {
	rules, err := LoadRulesFromJSON([]byte("[]"))
	require.NoError(t, err)
	assert.Empty(t, rules)
}

func TestLoadRulesFromJSON_SingleRule(t *testing.T) {
	data := []byte(`[{"name":"test","priority":1,"conditions":{"fact":"x","operator":"eq","value":1},"event":{"type":"match"}}]`)
	rules, err := LoadRulesFromJSON(data)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "test", rules[0].Name)
	assert.Equal(t, "eq", rules[0].Conditions.Operator)
}
