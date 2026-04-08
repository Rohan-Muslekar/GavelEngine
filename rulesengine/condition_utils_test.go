package rulesengine

import (
	"fmt"
	"testing"
)

func TestCountLeafConditions(t *testing.T) {
	tests := []struct {
		name string
		cond *Condition
		want int
	}{
		{"nil", nil, 0},
		{"single leaf", &Condition{Fact: "age", Operator: "gt", Value: 18}, 1},
		{"all with 2 leaves", &Condition{All: []Condition{
			{Fact: "age", Operator: "gt", Value: 18},
			{Fact: "score", Operator: "lt", Value: 50},
		}}, 2},
		{"nested", &Condition{All: []Condition{
			{Fact: "age", Operator: "gt", Value: 18},
			{Any: []Condition{
				{Fact: "score", Operator: "lt", Value: 50},
				{Fact: "score", Operator: "gt", Value: 90},
			}},
		}}, 3},
		{"not", &Condition{Not: &Condition{Fact: "active", Operator: "eq", Value: false}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountLeafConditions(tt.cond)
			if got != tt.want {
				t.Errorf("CountLeafConditions() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMaxDepth(t *testing.T) {
	tests := []struct {
		name string
		cond *Condition
		want int
	}{
		{"nil", nil, 0},
		{"single leaf", &Condition{Fact: "age", Operator: "gt", Value: 18}, 1},
		{"all with leaves", &Condition{All: []Condition{
			{Fact: "a", Operator: "gt", Value: 1},
			{Fact: "b", Operator: "lt", Value: 2},
		}}, 2},
		{"depth 3", &Condition{All: []Condition{
			{Fact: "a", Operator: "gt", Value: 1},
			{Any: []Condition{
				{Fact: "b", Operator: "lt", Value: 2},
				{All: []Condition{
					{Fact: "c", Operator: "eq", Value: 3},
					{Fact: "d", Operator: "ne", Value: 4},
				}},
			}},
		}}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxDepth(tt.cond)
			if got != tt.want {
				t.Errorf("MaxDepth() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWalkLeaves(t *testing.T) {
	cond := &Condition{All: []Condition{
		{Fact: "a", Operator: "gt", Value: 1},
		{Any: []Condition{
			{Fact: "b", Operator: "lt", Value: 2},
			{Fact: "c", Operator: "eq", Value: 3},
		}},
	}}

	var facts []string
	err := WalkLeaves(cond, func(leaf *Condition) error {
		facts = append(facts, leaf.Fact)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkLeaves returned error: %v", err)
	}
	if len(facts) != 3 || facts[0] != "a" || facts[1] != "b" || facts[2] != "c" {
		t.Errorf("WalkLeaves visited %v, want [a b c]", facts)
	}

	// Test early exit on error
	err = WalkLeaves(cond, func(leaf *Condition) error {
		if leaf.Fact == "b" {
			return fmt.Errorf("stop at b")
		}
		return nil
	})
	if err == nil || err.Error() != "stop at b" {
		t.Errorf("WalkLeaves should have stopped with error, got: %v", err)
	}
}

func TestEvaluateCondition(t *testing.T) {
	// Simple: age > 18
	result, err := EvaluateCondition(
		Condition{Fact: "age", Operator: "gt", Value: 18},
		map[string]interface{}{"age": 25},
	)
	if err != nil {
		t.Fatalf("EvaluateCondition error: %v", err)
	}
	if !result {
		t.Error("Expected true for age=25 > 18")
	}

	// Nested: (age > 18 AND score < 50) — should be true
	result, err = EvaluateCondition(
		Condition{All: []Condition{
			{Fact: "age", Operator: "gt", Value: 18},
			{Fact: "score", Operator: "lt", Value: 50},
		}},
		map[string]interface{}{"age": 25, "score": 30},
	)
	if err != nil {
		t.Fatalf("EvaluateCondition error: %v", err)
	}
	if !result {
		t.Error("Expected true for (age=25>18 AND score=30<50)")
	}

	// Short aliases: lt, gt, eq, ne
	result, err = EvaluateCondition(
		Condition{Fact: "revenue", Operator: "lt", Value: 100.0},
		map[string]interface{}{"revenue": 42.5},
	)
	if err != nil {
		t.Fatalf("EvaluateCondition error with alias: %v", err)
	}
	if !result {
		t.Error("Expected true for revenue=42.5 lt 100")
	}
}
