package rulesengine

import "fmt"

type TraceNode struct {
	Condition Condition    `json:"condition" bson:"condition" xml:"condition" yaml:"condition"`
	Result    bool         `json:"result" bson:"result" xml:"result" yaml:"result"`
	FactValue interface{}  `json:"factValue,omitempty" bson:"factValue,omitempty" xml:"factValue,omitempty" yaml:"factValue,omitempty"`
	Children  []*TraceNode `json:"children,omitempty" bson:"children,omitempty" xml:"children,omitempty" yaml:"children,omitempty"`
}

func (c *Condition) EvaluateWithTrace(almanac *Almanac, engine *Engine) (bool, *TraceNode, error) {
	trace := &TraceNode{
		Condition: *c,
	}

	if c.ConditionRef != "" {
		cond, ok := engine.conditions[c.ConditionRef]
		if !ok {
			if engine.allowUndefinedConditions {
				trace.Result = false
				return false, trace, nil
			}
			return false, nil, fmt.Errorf("undefined condition: %s", c.ConditionRef)
		}
		result, childTrace, err := cond.EvaluateWithTrace(almanac, engine)
		if err != nil {
			return false, nil, err
		}
		trace.Result = result
		trace.Children = []*TraceNode{childTrace}
		return result, trace, nil
	}

	if len(c.All) > 0 {
		trace.Children = make([]*TraceNode, 0, len(c.All))
		for _, cond := range c.All {
			result, childTrace, err := cond.EvaluateWithTrace(almanac, engine)
			if err != nil {
				return false, nil, err
			}
			trace.Children = append(trace.Children, childTrace)
			if !result {
				trace.Result = false
				return false, trace, nil
			}
		}
		trace.Result = true
		return true, trace, nil
	}

	if len(c.Any) > 0 {
		trace.Children = make([]*TraceNode, 0, len(c.Any))
		for _, cond := range c.Any {
			result, childTrace, err := cond.EvaluateWithTrace(almanac, engine)
			if err != nil {
				return false, nil, err
			}
			trace.Children = append(trace.Children, childTrace)
			if result {
				trace.Result = true
				return true, trace, nil
			}
		}
		trace.Result = false
		return false, trace, nil
	}

	if c.Not != nil {
		result, childTrace, err := c.Not.EvaluateWithTrace(almanac, engine)
		if err != nil {
			return false, nil, err
		}
		trace.Result = !result
		trace.Children = []*TraceNode{childTrace}
		return !result, trace, nil
	}

	if c.Fact != "" && c.Operator != "" {
		factValue, err := almanac.FactValue(c.Fact, c.Params, c.Path)
		if err != nil {
			return false, nil, err
		}
		opFunc, err := resolveOperator(c.Operator, engine)
		if err != nil {
			return false, nil, err
		}
		result := opFunc(factValue, c.Value)
		trace.Result = result
		trace.FactValue = factValue
		return result, trace, nil
	}

	return false, nil, fmt.Errorf("invalid condition")
}

type RunOption func(*runConfig)

type runConfig struct {
	trace bool
}

func WithTrace() RunOption {
	return func(c *runConfig) { c.trace = true }
}

func (r *Rule) EvaluateWithTrace(almanac *Almanac, engine *Engine) (bool, *RuleResult, error) {
	result, trace, err := r.Conditions.EvaluateWithTrace(almanac, engine)
	if err != nil {
		return false, nil, err
	}
	ruleResult := &RuleResult{
		Name:    r.Name,
		Success: result,
		Trace:   trace,
	}
	return result, ruleResult, nil
}
