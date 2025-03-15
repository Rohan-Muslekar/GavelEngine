// engine_test.go
package rulesengine

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEngineSimpleRule(t *testing.T) {
	engine := NewEngine()
	// Add fact "age" which retrieves a runtime fact "userAge"
	err := engine.AddFact("age", func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
		if userAge, ok := almanac.runtimeFacts["userAge"]; ok {
			return userAge, nil
		}
		return 0, nil
	})

	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}
	// Define rule: age >= 18 and age <= 25
	cond := Condition{
		All: []Condition{
			{
				Fact:     "age",
				Operator: "greaterThanInclusive",
				Value:    18,
			},
			{
				Fact:     "age",
				Operator: "lessThanInclusive",
				Value:    25,
			},
		},
	}
	event := Event{
		Type: "young-adult",
		Params: map[string]interface{}{
			"message": "User is a young adult",
		},
	}
	rule := NewRule(cond, event, WithName("age-rule"))
	engine.AddRule(rule)

	// Run engine with userAge = 22.
	runtimeFacts := map[string]interface{}{
		"userAge": 22,
	}
	result, err := engine.Run(runtimeFacts)
	if err != nil {
		t.Fatalf("Engine run failed: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Type != "young-adult" {
		t.Fatalf("Expected event type 'young-adult', got %s", result.Events[0].Type)
	}
}

func TestNestedConditions(t *testing.T) {
	engine := NewEngine()
	// Fact "score" returns the runtime fact "score"
	err := engine.AddFact("score", func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
		if val, ok := almanac.runtimeFacts["score"]; ok {
			return val, nil
		}
		return 0, nil
	})

	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}

	// Condition: score > 50 AND (score < 70 OR score > 90)
	cond := Condition{
		All: []Condition{
			{
				Fact:     "score",
				Operator: "greaterThan",
				Value:    50,
			},
			{
				Any: []Condition{
					{
						Fact:     "score",
						Operator: "lessThan",
						Value:    70,
					},
					{
						Fact:     "score",
						Operator: "greaterThan",
						Value:    90,
					},
				},
			},
		},
	}
	event := Event{
		Type: "special-score",
		Params: map[string]interface{}{
			"info": "Score meets criteria",
		},
	}
	rule := NewRule(cond, event, WithName("score-rule"))
	engine.AddRule(rule)

	// Test with score = 65 (should pass).
	runtimeFacts := map[string]interface{}{
		"score": 65,
	}
	result, err := engine.Run(runtimeFacts)
	if err != nil {
		t.Fatalf("Engine run failed: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("Expected 1 event for score 65, got %d", len(result.Events))
	}

	// Test with score = 80 (should fail).
	runtimeFacts = map[string]interface{}{
		"score": 80,
	}
	result, err = engine.Run(runtimeFacts)
	if err != nil {
		t.Fatalf("Engine run failed: %v", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("Expected 0 events for score 80, got %d", len(result.Events))
	}
}

func TestRuleChaining(t *testing.T) {
	engine := NewEngine()
	// Rule 1: if flag is true, add runtime fact "rule1Passed" = true.
	cond1 := Condition{
		Fact:     "flag",
		Operator: "equal",
		Value:    true,
	}
	event1 := Event{
		Type:   "rule1-event",
		Params: map[string]interface{}{},
	}
	rule1 := NewRule(cond1, event1, WithName("rule1"), WithPriorityForRule(10),
		WithOnSuccess(func(event Event, almanac *Almanac, rr *RuleResult) error {
			almanac.AddRuntimeFact("rule1Passed", true)
			return nil
		}))
	engine.AddRule(rule1)

	// Rule 2: if rule1Passed is true, then fire rule2 event.
	cond2 := Condition{
		Fact:     "rule1Passed",
		Operator: "equal",
		Value:    true,
	}
	event2 := Event{
		Type:   "rule2-event",
		Params: map[string]interface{}{},
	}
	rule2 := NewRule(cond2, event2, WithName("rule2"), WithPriorityForRule(1))
	engine.AddRule(rule2)

	runtimeFacts := map[string]interface{}{
		"flag": true,
	}
	result, err := engine.Run(runtimeFacts)
	if err != nil {
		t.Fatalf("Engine run failed: %v", err)
	}
	// Expect both rule1 and rule2 events.
	if len(result.Events) != 2 {
		t.Fatalf("Expected 2 events from rule chaining, got %d", len(result.Events))
	}
}

func TestAsyncFact(t *testing.T) {
	engine := NewEngine()
	// Simulate an asynchronous fact by sleeping.
	err := engine.AddFact("delayed", FactFunc(func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "done", nil
	}))
	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}
	cond := Condition{
		Fact:     "delayed",
		Operator: "equal",
		Value:    "done",
	}
	event := Event{
		Type:   "delayed-event",
		Params: map[string]interface{}{},
	}
	rule := NewRule(cond, event, WithName("delayed-rule"))
	engine.AddRule(rule)

	result, err := engine.Run(nil)
	if err != nil {
		t.Fatalf("Engine run failed: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if len(result.Events) != 1 {
		t.Fatalf("Expected delayed event to trigger, got %d", len(result.Events))
	}
}

func TestContainsOperator(t *testing.T) {
	engine := NewEngine()

	// Register a custom "contains" operator.
	// This operator expects factValue to be a slice or array and
	// returns true if any element equals the condition value.
	engine.AddOperator("contains", func(factValue interface{}, conditionValue interface{}) bool {
		rv := reflect.ValueOf(factValue)
		// Ensure that factValue is a slice or array.
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return false
		}
		for i := 0; i < rv.Len(); i++ {
			if reflect.DeepEqual(rv.Index(i).Interface(), conditionValue) {
				return true
			}
		}
		return false
	})

	// Add a fact "colors" that returns an array of colors.
	err := engine.AddFact("colors", FactFunc(func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
		// Return as a slice of interface{} for generic handling.
		return []interface{}{"red", "green", "blue"}, nil
	}))
	if err != nil {
		t.Fatalf("Failed to add fact 'colors': %v", err)
	}

	// Define a rule condition that checks if the fact "colors" contains "blue".
	cond := Condition{
		Fact:     "colors",
		Operator: "contains",
		Value:    "blue",
	}
	event := Event{
		Type: "contains-event",
		Params: map[string]interface{}{
			"msg": "The array contains blue",
		},
	}
	rule := NewRule(cond, event, WithName("contains-rule"))
	engine.AddRule(rule)

	// Run the engine (no runtime facts required).
	result, err := engine.Run(nil)
	if err != nil {
		t.Fatalf("Engine run failed: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Type != "contains-event" {
		t.Fatalf("Expected event type 'contains-event', got %s", result.Events[0].Type)
	}
}

func TestCaseInsensitiveOperator(t *testing.T) {
	engine := NewEngine()

	// Register the "caseInsensitive" decorator.
	// This decorator converts both values to lowercase (if they are strings)
	// and then calls the underlying operator.
	engine.AddOperatorDecorator("caseInsensitive", func(factValue interface{}, conditionValue interface{}, next OperatorFunc) bool {
		s1, ok1 := factValue.(string)
		s2, ok2 := conditionValue.(string)
		if ok1 && ok2 {
			return next(strings.ToLower(s1), strings.ToLower(s2))
		}
		// If values are not strings, fall back to the default behavior.
		return next(factValue, conditionValue)
	})

	// Add a fact "username" that returns "Alice".
	err := engine.AddFact("username", FactFunc(func(params map[string]interface{}, almanac *Almanac) (interface{}, error) {
		return "Alice", nil
	}))
	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}

	// Create a rule that uses the "caseInsensitive:equal" operator.
	// It should compare the fact "username" to "alice" in a case-insensitive manner.
	cond := Condition{
		Fact:     "username",
		Operator: "caseInsensitive:equal",
		Value:    "alice",
	}
	event := Event{
		Type: "welcome-event",
		Params: map[string]interface{}{
			"message": "Welcome, Alice!",
		},
	}
	rule := NewRule(cond, event, WithName("welcome-rule"))
	engine.AddRule(rule)

	// Run the engine.
	result, err := engine.Run(nil)
	if err != nil {
		t.Fatalf("Engine run failed: %v", err)
	}

	// Expect the welcome event to be triggered.
	if len(result.Events) != 1 {
		t.Fatalf("Expected 1 event from caseInsensitive operator test, got %d", len(result.Events))
	}
	if result.Events[0].Type != "welcome-event" {
		t.Fatalf("Expected event type 'welcome-event', got %s", result.Events[0].Type)
	}
}
