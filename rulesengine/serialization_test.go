package rulesengine

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCondition_YAMLRoundTrip(t *testing.T) {
	original := Condition{
		All: []Condition{
			{Fact: "age", Operator: "gte", Value: 18},
			{Not: &Condition{Fact: "banned", Operator: "equal", Value: true}},
		},
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var decoded Condition
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.Len(t, decoded.All, 2)
	assert.Equal(t, "age", decoded.All[0].Fact)
	assert.Equal(t, "gte", decoded.All[0].Operator)
	assert.Equal(t, 18, decoded.All[0].Value)
	require.NotNil(t, decoded.All[1].Not)
	assert.Equal(t, "banned", decoded.All[1].Not.Fact)
	assert.Equal(t, true, decoded.All[1].Not.Value)
}

func TestRule_YAMLRoundTrip(t *testing.T) {
	original := Rule{
		Name:     "age-check",
		Priority: 10,
		Conditions: Condition{
			Fact: "age", Operator: "gte", Value: 18,
		},
		Event: Event{Type: "adult", Params: map[string]interface{}{"msg": "welcome"}},
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var decoded Rule
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "age-check", decoded.Name)
	assert.Equal(t, 10, decoded.Priority)
	assert.Equal(t, "age", decoded.Conditions.Fact)
	assert.Equal(t, "gte", decoded.Conditions.Operator)
	assert.Equal(t, "adult", decoded.Event.Type)
	assert.Equal(t, "welcome", decoded.Event.Params["msg"])
	assert.Nil(t, decoded.OnSuccess)
	assert.Nil(t, decoded.OnFailure)
}

func TestRunResult_YAMLRoundTrip(t *testing.T) {
	original := RunResult{
		Events: []Event{
			{Type: "match", Params: map[string]interface{}{"score": 95}},
		},
		FailureEvents: []Event{},
		RuleResults: []*RuleResult{
			{Name: "rule-1", Success: true},
		},
		FailureRuleResults: []*RuleResult{},
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var decoded RunResult
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.Len(t, decoded.Events, 1)
	assert.Equal(t, "match", decoded.Events[0].Type)
	assert.Equal(t, 95, decoded.Events[0].Params["score"])
	require.Len(t, decoded.RuleResults, 1)
	assert.Equal(t, "rule-1", decoded.RuleResults[0].Name)
	assert.True(t, decoded.RuleResults[0].Success)
	assert.Nil(t, decoded.Almanac)
}

func TestRuleResult_XMLRoundTrip(t *testing.T) {
	original := RuleResult{
		Name:    "test-rule",
		Success: true,
	}

	data, err := xml.Marshal(original)
	require.NoError(t, err)

	var decoded RuleResult
	err = xml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "test-rule", decoded.Name)
	assert.True(t, decoded.Success)
}
