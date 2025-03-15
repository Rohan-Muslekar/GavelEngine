// fact.go
package rulesengine

// FactFunc defines a function to compute a fact’s value.
type FactFunc func(params map[string]interface{}, almanac *Almanac) (interface{}, error)

// Fact represents a fact that may be a constant or computed via a function.
type Fact struct {
	Id         string
	Fn         FactFunc
	Cache      bool
	Priority   int
	IsConstant bool
}

// FactOption allows customization of a fact.
type FactOption func(*Fact)

// WithNoCache disables caching for a fact.
func WithNoCache() FactOption {
	return func(f *Fact) {
		f.Cache = false
	}
}

// WithPriority sets the priority for a fact.
func WithPriorityForFact(priority int) FactOption {
	return func(f *Fact) {
		f.Priority = priority
	}
}

// Evaluate executes the fact function.
func (f *Fact) Evaluate(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
	value, err := f.Fn(params, almanac)
	return value, err
}
