// engine.go
package rulesengine

import (
	"reflect"
	"sort"
	"strings"
)

// Engine stores the facts, rules, operator functions and other options.
type Engine struct {
	facts                     map[string]*Fact
	rules                     []*Rule
	operators                 map[string]OperatorFunc
	operatorDecorators        map[string]OperatorDecorator
	conditions                map[string]Condition
	allowUndefinedFacts       bool
	allowUndefinedConditions  bool
	replaceFactsInEventParams bool
	stopRequested             bool
	pathResolver              PathResolverFunc
}

// NewEngine creates a new engine instance with sensible defaults.
func NewEngine() *Engine {
	e := &Engine{
		facts:                     make(map[string]*Fact),
		rules:                     []*Rule{},
		operators:                 make(map[string]OperatorFunc),
		operatorDecorators:        make(map[string]OperatorDecorator),
		conditions:                make(map[string]Condition),
		allowUndefinedFacts:       false,
		allowUndefinedConditions:  false,
		replaceFactsInEventParams: false,
		stopRequested:             false,
		pathResolver:              DefaultPathResolver,
	}
	e.initOperators()
	return e
}

// AddFact adds a fact (constant or function) to the engine.
func (e *Engine) AddFact(id string, definition interface{}, options ...FactOption) error {
	var factFunc FactFunc
	var constant bool
	switch def := definition.(type) {
	case FactFunc:
		factFunc = def
		constant = false
	default:
		// Wrap constant value in a function.
		factFunc = func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
			return def, nil
		}
		constant = true
	}
	fact := &Fact{
		Id:         id,
		Fn:         factFunc,
		Cache:      true, // default: cache result
		Priority:   1,
		IsConstant: constant,
	}
	for _, opt := range options {
		opt(fact)
	}
	e.facts[id] = fact
	return nil
}

// RemoveFact removes a fact by its id.
func (e *Engine) RemoveFact(id string) {
	delete(e.facts, id)
}

// AddRule adds a rule to the engine and sorts rules by priority (higher first).
func (e *Engine) AddRule(rule *Rule) {
	e.rules = append(e.rules, rule)
	sort.Slice(e.rules, func(i, j int) bool {
		return e.rules[i].Priority > e.rules[j].Priority
	})
}

// RemoveRule removes rules by matching the rule’s Name.
func (e *Engine) RemoveRule(ruleName string) {
	filtered := []*Rule{}
	for _, r := range e.rules {
		if r.Name != ruleName {
			filtered = append(filtered, r)
		}
	}
	e.rules = filtered
}

// AddOperator registers a new operator.
func (e *Engine) AddOperator(name string, op OperatorFunc) {
	e.operators[name] = op
}

// RemoveOperator removes a registered operator.
func (e *Engine) RemoveOperator(name string) {
	delete(e.operators, name)
}

// AddOperatorDecorator registers a new operator decorator.
func (e *Engine) AddOperatorDecorator(name string, decorator OperatorDecorator) {
	e.operatorDecorators[name] = decorator
}

// RemoveOperatorDecorator removes an operator decorator.
func (e *Engine) RemoveOperatorDecorator(name string) {
	delete(e.operatorDecorators, name)
}

// SetCondition sets a named condition that can be referenced by rules.
func (e *Engine) SetCondition(name string, cond Condition) {
	e.conditions[name] = cond
}

// RemoveCondition removes a named condition.
func (e *Engine) RemoveCondition(name string) {
	delete(e.conditions, name)
}

// Stop requests that the engine stop evaluating further rules.
func (e *Engine) Stop() {
	e.stopRequested = true
}

// Run executes all the rules in the engine using the provided runtime facts.
// It returns a RunResult with events and rule results.
func (e *Engine) Run(runtimeFacts map[string]interface{}) (*RunResult, error) {
	almanac := NewAlmanac(e, runtimeFacts)
	result := &RunResult{
		Almanac:            almanac,
		Events:             []Event{},
		FailureEvents:      []Event{},
		RuleResults:        []*RuleResult{},
		FailureRuleResults: []*RuleResult{},
	}

	for _, rule := range e.rules {
		if e.stopRequested {
			break
		}
		passed, ruleResult, err := rule.Evaluate(almanac, e)
		if err != nil {
			return nil, err
		}
		result.RuleResults = append(result.RuleResults, ruleResult)
		if passed {
			result.Events = append(result.Events, rule.Event)
			if rule.OnSuccess != nil {
				if err := rule.OnSuccess(rule.Event, almanac, ruleResult); err != nil {
					return nil, err
				}
			}
		} else {
			result.FailureRuleResults = append(result.FailureRuleResults, ruleResult)
			result.FailureEvents = append(result.FailureEvents, rule.Event)
			if rule.OnFailure != nil {
				if err := rule.OnFailure(rule.Event, almanac, ruleResult); err != nil {
					return nil, err
				}
			}
		}
	}
	return result, nil
}

// initOperators registers the built-in operators.

func (e *Engine) initOperators() {
	e.operators["equal"] = func(factValue, conditionValue interface{}) bool {
		if s1, ok := factValue.(string); ok {
			if s2, ok := conditionValue.(string); ok {
				return s1 == s2
			}
		}
		return reflect.DeepEqual(factValue, conditionValue)
	}
	e.operators["notEqual"] = func(factValue, conditionValue interface{}) bool {
		return !e.operators["equal"](factValue, conditionValue)
	}
	e.operators["lessThan"] = func(factValue, conditionValue interface{}) bool {
		return compare(factValue, conditionValue) < 0
	}
	e.operators["lessThanInclusive"] = func(factValue, conditionValue interface{}) bool {
		return compare(factValue, conditionValue) <= 0
	}
	e.operators["greaterThan"] = func(factValue, conditionValue interface{}) bool {
		return compare(factValue, conditionValue) > 0
	}
	e.operators["greaterThanInclusive"] = func(factValue, conditionValue interface{}) bool {
		return compare(factValue, conditionValue) >= 0
	}
	// Additional operators (e.g. for arrays) can be added here.
}

// compare compares two values as numbers (or as strings if not numbers).
func compare(a, b interface{}) int {
	fa, ok1 := toFloat64(a)
	fb, ok2 := toFloat64(b)
	if ok1 && ok2 {
		switch {
		case fa < fb:
			return -1
		case fa > fb:
			return 1
		default:
			return 0
		}
	}
	sa, ok1 := a.(string)
	sb, ok2 := b.(string)
	if ok1 && ok2 {
		if sa < sb {
			return -1
		} else if sa > sb {
			return 1
		} else {
			return 0
		}
	}
	return 0
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
}

// PathResolverFunc defines how to extract a value given a path.
type PathResolverFunc func(object interface{}, path string) interface{}

// DefaultPathResolver is a simple implementation that supports top-level access.
// For a field ".foo", it returns object["foo"] if object is a map.
func DefaultPathResolver(object interface{}, path string) interface{} {
	if m, ok := object.(map[string]interface{}); ok {
		key := strings.TrimPrefix(path, ".")
		return m[key]
	}
	return nil
}

// RunResult holds the result of an engine run.
type RunResult struct {
	Events             []Event
	FailureEvents      []Event
	Almanac            *Almanac
	RuleResults        []*RuleResult
	FailureRuleResults []*RuleResult
}
