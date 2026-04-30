package rulesengine

import "fmt"

func (e *Engine) Validate() []ValidationError {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var errs []ValidationError
	for i, rule := range e.rules {
		prefix := fmt.Sprintf("rules[%d]", i)

		structErrs := ValidateCondition(&rule.Conditions)
		for _, se := range structErrs {
			path := prefix
			if se.Path != "" {
				path = prefix + "." + se.Path
			}
			errs = append(errs, ValidationError{Path: path, Message: se.Message})
		}

		errs = append(errs, e.validateConditionState(&rule.Conditions, prefix)...)
	}
	return errs
}

func (e *Engine) validateConditionState(c *Condition, path string) []ValidationError {
	var errs []ValidationError

	if c.ConditionRef != "" {
		if !e.allowUndefinedConditions {
			if _, ok := e.conditions[c.ConditionRef]; !ok {
				errs = append(errs, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("undefined condition reference: %s", c.ConditionRef),
				})
			}
		}
		return errs
	}

	if c.Fact != "" && c.Operator != "" {
		if !e.allowUndefinedFacts {
			if _, ok := e.facts[c.Fact]; !ok {
				errs = append(errs, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("undefined fact: %s", c.Fact),
				})
			}
		}
		parts := splitOperator(c.Operator)
		baseName := parts[len(parts)-1]
		if _, ok := e.operators[baseName]; !ok {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("undefined operator: %s", baseName),
			})
		}
		for i := 0; i < len(parts)-1; i++ {
			if _, ok := e.operatorDecorators[parts[i]]; !ok {
				errs = append(errs, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("undefined operator decorator: %s", parts[i]),
				})
			}
		}
		return errs
	}

	for i, child := range c.All {
		errs = append(errs, e.validateConditionState(&child, fmt.Sprintf("%s.All[%d]", path, i))...)
	}
	for i, child := range c.Any {
		errs = append(errs, e.validateConditionState(&child, fmt.Sprintf("%s.Any[%d]", path, i))...)
	}
	if c.Not != nil {
		errs = append(errs, e.validateConditionState(c.Not, path+".Not")...)
	}

	return errs
}

type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	if e.Path == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

func ValidateCondition(c *Condition) []ValidationError {
	return validateCondition(c, "")
}

func validateCondition(c *Condition, path string) []ValidationError {
	if c == nil {
		return []ValidationError{{Path: path, Message: "condition is nil"}}
	}
	var errs []ValidationError

	hasAll := c.All != nil
	hasAny := c.Any != nil
	hasNot := c.Not != nil
	hasLeaf := c.Fact != "" || c.Operator != ""
	hasRef := c.ConditionRef != ""

	typeCount := 0
	if hasAll {
		typeCount++
	}
	if hasAny {
		typeCount++
	}
	if hasNot {
		typeCount++
	}
	if hasLeaf {
		typeCount++
	}
	if hasRef {
		typeCount++
	}

	if typeCount == 0 {
		errs = append(errs, ValidationError{Path: path, Message: "condition is empty"})
		return errs
	}

	if typeCount > 1 {
		errs = append(errs, ValidationError{Path: path, Message: "condition has multiple types set"})
	}

	if hasLeaf {
		if c.Fact == "" {
			errs = append(errs, ValidationError{Path: path, Message: "leaf condition missing Fact"})
		}
		if c.Operator == "" {
			errs = append(errs, ValidationError{Path: path, Message: "leaf condition missing Operator"})
		}
	}

	if hasAll {
		if len(c.All) == 0 {
			errs = append(errs, ValidationError{Path: path, Message: "All must have at least one child"})
		}
		for i, child := range c.All {
			childPath := fmt.Sprintf("All[%d]", i)
			if path != "" {
				childPath = path + "." + childPath
			}
			errs = append(errs, validateCondition(&child, childPath)...)
		}
	}

	if hasAny {
		if len(c.Any) == 0 {
			errs = append(errs, ValidationError{Path: path, Message: "Any must have at least one child"})
		}
		for i, child := range c.Any {
			childPath := fmt.Sprintf("Any[%d]", i)
			if path != "" {
				childPath = path + "." + childPath
			}
			errs = append(errs, validateCondition(&child, childPath)...)
		}
	}

	if hasNot {
		childPath := "Not"
		if path != "" {
			childPath = path + ".Not"
		}
		errs = append(errs, validateCondition(c.Not, childPath)...)
	}

	return errs
}
