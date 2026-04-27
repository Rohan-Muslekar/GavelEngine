package rulesengine

import "fmt"

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
