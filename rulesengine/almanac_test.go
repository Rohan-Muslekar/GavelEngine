package rulesengine

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlmanac_RuntimeFactPriority(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("x", FactFunc(func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
		return "from-engine", nil
	}))
	almanac := NewAlmanac(engine, map[string]interface{}{"x": "from-runtime"})

	val, err := almanac.FactValue("x", nil, "")
	require.NoError(t, err)
	assert.Equal(t, "from-runtime", val)
}

func TestAlmanac_FactCaching(t *testing.T) {
	var callCount int32
	engine := NewEngine()
	engine.AddFact("x", FactFunc(func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
		atomic.AddInt32(&callCount, 1)
		return 42, nil
	}))
	almanac := NewAlmanac(engine, nil)

	val1, err := almanac.FactValue("x", nil, "")
	require.NoError(t, err)
	val2, err := almanac.FactValue("x", nil, "")
	require.NoError(t, err)

	assert.Equal(t, 42, val1)
	assert.Equal(t, 42, val2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestAlmanac_CacheKeyByParams(t *testing.T) {
	var callCount int32
	engine := NewEngine()
	engine.AddFact("x", FactFunc(func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
		atomic.AddInt32(&callCount, 1)
		if v, ok := params["key"]; ok {
			return v, nil
		}
		return "default", nil
	}))
	almanac := NewAlmanac(engine, nil)

	val1, err := almanac.FactValue("x", map[string]interface{}{"key": "a"}, "")
	require.NoError(t, err)
	val2, err := almanac.FactValue("x", map[string]interface{}{"key": "b"}, "")
	require.NoError(t, err)

	assert.Equal(t, "a", val1)
	assert.Equal(t, "b", val2)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}

func TestAlmanac_NoCacheOption(t *testing.T) {
	var callCount int32
	engine := NewEngine()
	engine.AddFact("x", FactFunc(func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
		atomic.AddInt32(&callCount, 1)
		return 42, nil
	}), WithNoCache())
	almanac := NewAlmanac(engine, nil)

	almanac.FactValue("x", nil, "")
	almanac.FactValue("x", nil, "")

	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}

func TestAlmanac_DynamicFact(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("multiply", FactFunc(func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
		a, _ := params["a"].(float64)
		b, _ := params["b"].(float64)
		return a * b, nil
	}))
	almanac := NewAlmanac(engine, nil)

	val, err := almanac.FactValue("multiply", map[string]interface{}{"a": 3.0, "b": 4.0}, "")
	require.NoError(t, err)
	assert.Equal(t, 12.0, val)
}

func TestAlmanac_UndefinedFact_Default(t *testing.T) {
	engine := NewEngine()
	almanac := NewAlmanac(engine, nil)

	_, err := almanac.FactValue("missing", nil, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undefined fact")
}

func TestAlmanac_UndefinedFact_Allowed(t *testing.T) {
	engine := NewEngine()
	engine.allowUndefinedFacts = true
	almanac := NewAlmanac(engine, nil)

	val, err := almanac.FactValue("missing", nil, "")
	require.NoError(t, err)
	assert.Nil(t, val)
}

func TestAlmanac_PathResolution(t *testing.T) {
	engine := NewEngine()
	engine.AddFact("user", map[string]interface{}{
		"name": "Alice",
		"age":  30,
	})
	almanac := NewAlmanac(engine, nil)

	val, err := almanac.FactValue("user", nil, ".name")
	require.NoError(t, err)
	assert.Equal(t, "Alice", val)
}

func TestAlmanac_AddRuntimeFact(t *testing.T) {
	engine := NewEngine()
	almanac := NewAlmanac(engine, nil)

	almanac.AddRuntimeFact("x", 99)
	val, err := almanac.FactValue("x", nil, "")
	require.NoError(t, err)
	assert.Equal(t, 99, val)
}
