// rule.go
package rulesengine

import (
	"fmt"
	"strings"
)

// Rule represents a single rule with its conditions, event, priority, and callbacks.
type Rule struct {
	Conditions Condition
	Event      Event
	Priority   int
	Name       string
	OnSuccess  func(event Event, almanac *Almanac, ruleResult *RuleResult) error
	OnFailure  func(event Event, almanac *Almanac, ruleResult *RuleResult) error
}

// RuleResult holds metadata about a rule evaluation.
type RuleResult struct {
	Name    string
	Success bool
	// Additional details (like intermediate condition results) could be stored here.
}

// NewRule creates a new rule instance.
func NewRule(conditions Condition, event Event, options ...RuleOption) *Rule {
	rule := &Rule{
		Conditions: conditions,
		Event:      event,
		Priority:   1,
	}
	for _, opt := range options {
		opt(rule)
	}
	return rule
}

// RuleOption defines a type for options that modify a Rule.
type RuleOption func(*Rule)

// WithPriority sets the rule’s priority.
func WithPriorityForRule(priority int) RuleOption {
	return func(r *Rule) {
		r.Priority = priority
	}
}

// WithName sets the rule’s name.
func WithName(name string) RuleOption {
	return func(r *Rule) {
		r.Name = name
	}
}

// WithOnSuccess registers a callback for when the rule succeeds.
func WithOnSuccess(callback func(event Event, almanac *Almanac, rr *RuleResult) error) RuleOption {
	return func(r *Rule) {
		r.OnSuccess = callback
	}
}

// WithOnFailure registers a callback for when the rule fails.
func WithOnFailure(callback func(event Event, almanac *Almanac, rr *RuleResult) error) RuleOption {
	return func(r *Rule) {
		r.OnFailure = callback
	}
}

// Evaluate runs the rule’s conditions using the provided almanac and engine.
func (r *Rule) Evaluate(almanac *Almanac, engine *Engine) (bool, *RuleResult, error) {
	result, err := r.Conditions.Evaluate(almanac, engine)
	if err != nil {
		return false, nil, err
	}
	ruleResult := &RuleResult{
		Name:    r.Name,
		Success: result,
	}
	return result, ruleResult, nil
}

// Event represents an event triggered by a rule.
type Event struct {
	Type   string
	Params map[string]interface{}
}

// Condition represents a rule condition. It supports basic comparisons as well as nested boolean expressions.
type Condition struct {
	All          []Condition            `json:"all,omitempty"`
	Any          []Condition            `json:"any,omitempty"`
	Not          *Condition             `json:"not,omitempty"`
	Fact         string                 `json:"fact,omitempty"`
	Operator     string                 `json:"operator,omitempty"`
	Value        interface{}            `json:"value,omitempty"`
	Params       map[string]interface{} `json:"params,omitempty"`
	Path         string                 `json:"path,omitempty"`
	ConditionRef string                 `json:"condition,omitempty"` // For referencing a named condition.
}

// Evaluate evaluates the condition recursively.
func (c *Condition) Evaluate(almanac *Almanac, engine *Engine) (bool, error) {
	// If referencing a named condition.
	if c.ConditionRef != "" {
		cond, ok := engine.conditions[c.ConditionRef]
		if !ok {
			if engine.allowUndefinedConditions {
				return false, nil
			}
			return false, fmt.Errorf("undefined condition: %s", c.ConditionRef)
		}
		return cond.Evaluate(almanac, engine)
	}
	// Compound conditions.
	if len(c.All) > 0 {
		for _, cond := range c.All {
			res, err := cond.Evaluate(almanac, engine)
			if err != nil {
				return false, err
			}
			if !res {
				return false, nil
			}
		}
		return true, nil
	}
	if len(c.Any) > 0 {
		for _, cond := range c.Any {
			res, err := cond.Evaluate(almanac, engine)
			if err != nil {
				return false, err
			}
			if res {
				return true, nil
			}
		}
		return false, nil
	}
	if c.Not != nil {
		res, err := c.Not.Evaluate(almanac, engine)
		if err != nil {
			return false, err
		}
		return !res, nil
	}
	// Basic condition: evaluate fact using operator.
	if c.Fact != "" && c.Operator != "" {
		factValue, err := almanac.FactValue(c.Fact, c.Params, c.Path)
		if err != nil {
			return false, err
		}
		opFunc, err := resolveOperator(c.Operator, engine)
		if err != nil {
			return false, err
		}
		return opFunc(factValue, c.Value), nil
	}
	return false, fmt.Errorf("invalid condition")
}

// resolveOperator resolves an operator string (possibly with decorators) into an OperatorFunc.
func resolveOperator(operator string, engine *Engine) (OperatorFunc, error) {
	parts := splitOperator(operator)
	baseOpName := parts[len(parts)-1]
	baseOp, ok := engine.operators[baseOpName]
	if !ok {
		return nil, fmt.Errorf("undefined operator: %s", baseOpName)
	}
	opFunc := baseOp
	// Wrap with decorators (if any) in reverse order.
	for i := len(parts) - 2; i >= 0; i-- {
		decoratorName := parts[i]
		decorator, ok := engine.operatorDecorators[decoratorName]
		if !ok {
			return nil, fmt.Errorf("undefined operator decorator: %s", decoratorName)
		}
		nextOp := opFunc
		opFunc = func(factValue, conditionValue interface{}) bool {
			return decorator(factValue, conditionValue, nextOp)
		}
	}
	return opFunc, nil
}

func splitOperator(op string) []string {
	return strings.Split(op, ":")
}
