package rulesengine

// CountLeafConditions returns the number of leaf (fact-based) conditions in the tree.
func CountLeafConditions(c *Condition) int {
	if c == nil {
		return 0
	}
	// Leaf condition
	if c.Fact != "" {
		return 1
	}
	count := 0
	for i := range c.All {
		count += CountLeafConditions(&c.All[i])
	}
	for i := range c.Any {
		count += CountLeafConditions(&c.Any[i])
	}
	if c.Not != nil {
		count += CountLeafConditions(c.Not)
	}
	return count
}

// MaxDepth returns the maximum nesting depth of the condition tree.
// A single leaf condition has depth 1. An All/Any/Not wrapper adds 1.
func MaxDepth(c *Condition) int {
	if c == nil {
		return 0
	}
	// Leaf condition
	if c.Fact != "" {
		return 1
	}
	maxChild := 0
	for i := range c.All {
		if d := MaxDepth(&c.All[i]); d > maxChild {
			maxChild = d
		}
	}
	for i := range c.Any {
		if d := MaxDepth(&c.Any[i]); d > maxChild {
			maxChild = d
		}
	}
	if c.Not != nil {
		if d := MaxDepth(c.Not); d > maxChild {
			maxChild = d
		}
	}
	if maxChild > 0 || len(c.All) > 0 || len(c.Any) > 0 || c.Not != nil {
		return maxChild + 1
	}
	return 0
}

// WalkLeaves calls fn for each leaf condition (one with a Fact field set) in the tree.
// If fn returns an error, traversal stops and the error is returned.
func WalkLeaves(c *Condition, fn func(leaf *Condition) error) error {
	if c == nil {
		return nil
	}
	if c.Fact != "" {
		return fn(c)
	}
	for i := range c.All {
		if err := WalkLeaves(&c.All[i], fn); err != nil {
			return err
		}
	}
	for i := range c.Any {
		if err := WalkLeaves(&c.Any[i], fn); err != nil {
			return err
		}
	}
	if c.Not != nil {
		return WalkLeaves(c.Not, fn)
	}
	return nil
}
