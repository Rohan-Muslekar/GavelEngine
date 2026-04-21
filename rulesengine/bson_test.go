package rulesengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestBSON_LeafCondition_RoundTrip(t *testing.T) {
	original := Condition{
		Fact:     "temperature",
		Operator: "greaterThan",
		Value:    36.5,
		Params:   map[string]interface{}{"unit": "celsius"},
	}

	data, err := bson.Marshal(original)
	require.NoError(t, err)

	var decoded Condition
	err = bson.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Fact, decoded.Fact)
	assert.Equal(t, original.Operator, decoded.Operator)
	assert.Equal(t, original.Value, decoded.Value)
	assert.Equal(t, "celsius", decoded.Params["unit"])
}

func TestBSON_NestedCondition_RoundTrip(t *testing.T) {
	original := Condition{
		All: []Condition{
			{Any: []Condition{
				{Fact: "a", Operator: "eq", Value: int32(1)},
				{Fact: "b", Operator: "lt", Value: int32(10)},
			}},
			{Not: &Condition{Fact: "c", Operator: "equal", Value: "blocked"}},
		},
	}

	data, err := bson.Marshal(original)
	require.NoError(t, err)

	var decoded Condition
	err = bson.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.Len(t, decoded.All, 2)
	require.Len(t, decoded.All[0].Any, 2)
	assert.Equal(t, "a", decoded.All[0].Any[0].Fact)
	assert.Equal(t, "b", decoded.All[0].Any[1].Fact)
	require.NotNil(t, decoded.All[1].Not)
	assert.Equal(t, "c", decoded.All[1].Not.Fact)
}

func TestBSON_Params_MixedTypes(t *testing.T) {
	original := Condition{
		Fact:     "x",
		Operator: "eq",
		Value:    int32(1),
		Params: map[string]interface{}{
			"str":    "hello",
			"num":    int32(42),
			"flt":    3.14,
			"flag":   true,
			"nested": map[string]interface{}{"key": "val"},
		},
	}

	data, err := bson.Marshal(original)
	require.NoError(t, err)

	var decoded Condition
	err = bson.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "hello", decoded.Params["str"])
	assert.Equal(t, int32(42), decoded.Params["num"])
	assert.Equal(t, 3.14, decoded.Params["flt"])
	assert.Equal(t, true, decoded.Params["flag"])
}

func TestBSON_OmitEmpty(t *testing.T) {
	tests := []struct {
		name        string
		cond        Condition
		absentField string
	}{
		{
			"leaf omits all/any/not",
			Condition{Fact: "x", Operator: "eq", Value: int32(1)},
			"all",
		},
		{
			"All-only omits fact/operator",
			Condition{All: []Condition{{Fact: "x", Operator: "eq", Value: int32(1)}}},
			"fact",
		},
		{
			"no params omits params",
			Condition{Fact: "x", Operator: "eq", Value: int32(1)},
			"params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := bson.Marshal(tt.cond)
			require.NoError(t, err)

			var raw bson.M
			err = bson.Unmarshal(data, &raw)
			require.NoError(t, err)

			_, exists := raw[tt.absentField]
			assert.False(t, exists, "field %q should be omitted", tt.absentField)
		})
	}
}

func TestBSON_NilFields(t *testing.T) {
	original := Condition{
		Fact:     "x",
		Operator: "eq",
		Value:    int32(1),
	}

	data, err := bson.Marshal(original)
	require.NoError(t, err)

	var decoded Condition
	err = bson.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.Not)
	assert.Nil(t, decoded.Params)
	assert.Nil(t, decoded.All)
	assert.Nil(t, decoded.Any)
}
