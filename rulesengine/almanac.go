// almanac.go
package rulesengine

import (
	"encoding/json"
	"fmt"
)

// Almanac collects fact values (with caching), runtime facts, events, and rule results.
type Almanac struct {
	engine       *Engine
	runtimeFacts map[string]interface{}
	factCache    map[string]map[string]interface{}
	events       []Event
	ruleResults  []*RuleResult
}

// NewAlmanac creates a new almanac instance for a run.
func NewAlmanac(engine *Engine, runtimeFacts map[string]interface{}) *Almanac {
	if runtimeFacts == nil {
		runtimeFacts = make(map[string]interface{})
	}
	return &Almanac{
		engine:       engine,
		runtimeFacts: runtimeFacts,
		factCache:    make(map[string]map[string]interface{}),
		events:       []Event{},
		ruleResults:  []*RuleResult{},
	}
}

// FactValue computes (or retrieves from cache) the value of a fact.
// If a path is provided, the engine’s pathResolver is used.
func (a *Almanac) FactValue(factId string, params map[string]interface{}, path string) (interface{}, error) {
	// Check runtime facts first.
	if val, ok := a.runtimeFacts[factId]; ok {
		if path != "" {
			return a.engine.pathResolver(val, path), nil
		}
		return val, nil
	}
	cacheKey, err := generateCacheKey(params)
	if err != nil {
		return nil, err
	}
	if factCache, ok := a.factCache[factId]; ok {
		if value, ok := factCache[cacheKey]; ok {
			if path != "" {
				return a.engine.pathResolver(value, path), nil
			}
			return value, nil
		}
	} else {
		a.factCache[factId] = make(map[string]interface{})
	}
	// Retrieve fact from the engine.
	fact, ok := a.engine.facts[factId]
	if !ok {
		if a.engine.allowUndefinedFacts {
			return nil, nil
		}
		return nil, fmt.Errorf("undefined fact: %s", factId)
	}
	value, err := fact.Evaluate(params, a)
	if err != nil {
		return nil, err
	}
	a.factCache[factId][cacheKey] = value
	if path != "" {
		return a.engine.pathResolver(value, path), nil
	}
	return value, nil
}

// AddRuntimeFact sets a fact value at runtime.
func (a *Almanac) AddRuntimeFact(factId string, value interface{}) {
	a.runtimeFacts[factId] = value
}

// generateCacheKey creates a cache key for a given parameters map.
func generateCacheKey(params map[string]interface{}) (string, error) {
	if params == nil {
		return "nil", nil
	}
	b, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetEvents returns all events collected during the run.
func (a *Almanac) GetEvents() []Event {
	return a.events
}

// GetRuleResults returns all rule results from the run.
func (a *Almanac) GetRuleResults() []*RuleResult {
	return a.ruleResults
}

// Get runtime facts
func (a *Almanac) GetRuntimeFacts() map[string]interface{} {
	return a.runtimeFacts
}
