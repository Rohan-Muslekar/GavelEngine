// operators.go
package rulesengine

// OperatorFunc defines a function that compares a fact value to a condition value.
type OperatorFunc func(factValue interface{}, conditionValue interface{}) bool

// OperatorDecorator defines a decorator that wraps an OperatorFunc.
type OperatorDecorator func(factValue interface{}, conditionValue interface{}, next OperatorFunc) bool
