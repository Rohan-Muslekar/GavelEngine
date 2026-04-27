package rulesengine

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type Engine struct {
	mu                        sync.RWMutex
	facts                     map[string]*Fact
	rules                     []*Rule
	operators                 map[string]OperatorFunc
	operatorDecorators        map[string]OperatorDecorator
	conditions                map[string]Condition
	allowUndefinedFacts       bool
	allowUndefinedConditions  bool
	replaceFactsInEventParams bool
	stopRequested             atomic.Bool
	pathResolver              PathResolverFunc
}

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
		pathResolver:              DefaultPathResolver,
	}
	e.initOperators()
	return e
}

func (e *Engine) AddFact(id string, definition interface{}, options ...FactOption) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	var factFunc FactFunc
	var constant bool
	switch def := definition.(type) {
	case FactFunc:
		factFunc = def
		constant = false
	default:
		factFunc = func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
			return def, nil
		}
		constant = true
	}
	fact := &Fact{
		Id:         id,
		Fn:         factFunc,
		Cache:      true,
		Priority:   1,
		IsConstant: constant,
	}
	for _, opt := range options {
		opt(fact)
	}
	e.facts[id] = fact
	return nil
}

func (e *Engine) RemoveFact(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.facts, id)
}

func (e *Engine) AddRule(rule *Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	errs := ValidateCondition(&rule.Conditions)
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, ve := range errs {
			msgs[i] = ve.Error()
		}
		return fmt.Errorf("invalid rule conditions: %s", strings.Join(msgs, "; "))
	}
	e.rules = append(e.rules, rule)
	sort.Slice(e.rules, func(i, j int) bool {
		return e.rules[i].Priority > e.rules[j].Priority
	})
	return nil
}

func (e *Engine) RemoveRule(ruleName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	filtered := []*Rule{}
	for _, r := range e.rules {
		if r.Name != ruleName {
			filtered = append(filtered, r)
		}
	}
	e.rules = filtered
}

func (e *Engine) AddOperator(name string, op OperatorFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.operators[name] = op
}

func (e *Engine) RemoveOperator(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.operators, name)
}

func (e *Engine) AddOperatorDecorator(name string, decorator OperatorDecorator) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.operatorDecorators[name] = decorator
}

func (e *Engine) RemoveOperatorDecorator(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.operatorDecorators, name)
}

func (e *Engine) SetCondition(name string, cond Condition) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.conditions[name] = cond
}

func (e *Engine) RemoveCondition(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.conditions, name)
}

func (e *Engine) Stop() {
	e.stopRequested.Store(true)
}

func (e *Engine) Run(runtimeFacts map[string]interface{}, options ...RunOption) (*RunResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	cfg := &runConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	almanac := NewAlmanac(e, runtimeFacts)
	result := &RunResult{
		Almanac:            almanac,
		Events:             []Event{},
		FailureEvents:      []Event{},
		RuleResults:        []*RuleResult{},
		FailureRuleResults: []*RuleResult{},
	}

	for _, rule := range e.rules {
		if e.stopRequested.Load() {
			break
		}
		var passed bool
		var ruleResult *RuleResult
		var err error
		if cfg.trace {
			passed, ruleResult, err = rule.EvaluateWithTrace(almanac, e)
		} else {
			passed, ruleResult, err = rule.Evaluate(almanac, e)
		}
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
	e.operators["in"] = func(factValue, conditionValue interface{}) bool {
		return sliceContains(conditionValue, factValue)
	}
	e.operators["notIn"] = func(factValue, conditionValue interface{}) bool {
		return !sliceContains(conditionValue, factValue)
	}
	e.operators["contains"] = func(factValue, conditionValue interface{}) bool {
		if s, ok := factValue.(string); ok {
			if sub, ok := conditionValue.(string); ok {
				return strings.Contains(s, sub)
			}
			return false
		}
		return sliceContains(factValue, conditionValue)
	}
	e.operators["doesNotContain"] = func(factValue, conditionValue interface{}) bool {
		return !e.operators["contains"](factValue, conditionValue)
	}
	e.operators["matches"] = func(factValue, conditionValue interface{}) bool {
		s, ok1 := factValue.(string)
		pattern, ok2 := conditionValue.(string)
		if !ok1 || !ok2 {
			return false
		}
		matched, err := regexp.MatchString(pattern, s)
		if err != nil {
			return false
		}
		return matched
	}
	e.operators["lt"] = e.operators["lessThan"]
	e.operators["gt"] = e.operators["greaterThan"]
	e.operators["eq"] = e.operators["equal"]
	e.operators["ne"] = e.operators["notEqual"]
	e.operators["lte"] = e.operators["lessThanInclusive"]
	e.operators["gte"] = e.operators["greaterThanInclusive"]
}

func EvaluateCondition(condition Condition, facts map[string]interface{}) (bool, error) {
	engine := NewEngine()
	almanac := NewAlmanac(engine, facts)
	return condition.Evaluate(almanac, engine)
}

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
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(v).Int()), true
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(v).Uint()), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func sliceContains(slice, element interface{}) bool {
	rv := reflect.ValueOf(slice)
	kind := rv.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return false
	}
	for i := 0; i < rv.Len(); i++ {
		if reflect.DeepEqual(rv.Index(i).Interface(), element) {
			return true
		}
	}
	return false
}

type PathResolverFunc func(object interface{}, path string) interface{}

func DefaultPathResolver(object interface{}, path string) interface{} {
	if m, ok := object.(map[string]interface{}); ok {
		key := strings.TrimPrefix(path, ".")
		return m[key]
	}
	return nil
}

type RunResult struct {
	Events             []Event       `json:"events" bson:"events" xml:"events" yaml:"events"`
	FailureEvents      []Event       `json:"failureEvents" bson:"failureEvents" xml:"failureEvents" yaml:"failureEvents"`
	Almanac            *Almanac      `json:"-" bson:"-" xml:"-" yaml:"-"`
	RuleResults        []*RuleResult `json:"ruleResults" bson:"ruleResults" xml:"ruleResults" yaml:"ruleResults"`
	FailureRuleResults []*RuleResult `json:"failureRuleResults" bson:"failureRuleResults" xml:"failureRuleResults" yaml:"failureRuleResults"`
}

func (e *Engine) GetRulesAsJSON() []interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var rules []interface{}
	for _, rule := range e.rules {
		rules = append(rules, rule.ToJSON())
	}
	return rules
}
