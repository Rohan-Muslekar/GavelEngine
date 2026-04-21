package rulesengine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountLeafConditions(t *testing.T) {
	tests := []struct {
		name string
		cond *Condition
		want int
	}{
		{"nil", nil, 0},
		{"single leaf", &Condition{Fact: "age", Operator: "gt", Value: 18}, 1},
		{"All with 3 leaves", &Condition{All: []Condition{
			{Fact: "a", Operator: "gt", Value: 1},
			{Fact: "b", Operator: "lt", Value: 2},
			{Fact: "c", Operator: "eq", Value: 3},
		}}, 3},
		{"Any with 2 leaves", &Condition{Any: []Condition{
			{Fact: "a", Operator: "gt", Value: 1},
			{Fact: "b", Operator: "lt", Value: 2},
		}}, 2},
		{"nested All[Any[leaf,leaf], leaf]", &Condition{All: []Condition{
			{Any: []Condition{
				{Fact: "a", Operator: "gt", Value: 1},
				{Fact: "b", Operator: "lt", Value: 2},
			}},
			{Fact: "c", Operator: "eq", Value: 3},
		}}, 3},
		{"Not[leaf]", &Condition{Not: &Condition{Fact: "active", Operator: "eq", Value: false}}, 1},
		{"empty All", &Condition{All: []Condition{}}, 0},
		{"empty Any", &Condition{Any: []Condition{}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CountLeafConditions(tt.cond))
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
		{"All[leaf, leaf]", &Condition{All: []Condition{
			{Fact: "a", Operator: "gt", Value: 1},
			{Fact: "b", Operator: "lt", Value: 2},
		}}, 2},
		{"Any[All[leaf,leaf], leaf]", &Condition{Any: []Condition{
			{All: []Condition{
				{Fact: "a", Operator: "gt", Value: 1},
				{Fact: "b", Operator: "lt", Value: 2},
			}},
			{Fact: "c", Operator: "eq", Value: 3},
		}}, 3},
		{"Not[leaf]", &Condition{Not: &Condition{Fact: "x", Operator: "eq", Value: 1}}, 2},
		{"depth 4", &Condition{All: []Condition{
			{Fact: "a", Operator: "gt", Value: 1},
			{Any: []Condition{
				{Fact: "b", Operator: "lt", Value: 2},
				{All: []Condition{
					{Fact: "c", Operator: "eq", Value: 3},
					{Fact: "d", Operator: "ne", Value: 4},
				}},
			}},
		}}, 4},
		{"empty All", &Condition{All: []Condition{}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MaxDepth(tt.cond))
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
		{Not: &Condition{Fact: "d", Operator: "ne", Value: 4}},
	}}

	var facts []string
	err := WalkLeaves(cond, func(leaf *Condition) error {
		facts = append(facts, leaf.Fact)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c", "d"}, facts)
	assert.Equal(t, CountLeafConditions(cond), len(facts))

	for i := 0; i < 10; i++ {
		var run []string
		WalkLeaves(cond, func(leaf *Condition) error {
			run = append(run, leaf.Fact)
			return nil
		})
		assert.Equal(t, facts, run, "run %d should match", i)
	}
}

func TestWalkLeaves_ErrorStopsWalk(t *testing.T) {
	cond := &Condition{All: []Condition{
		{Fact: "a", Operator: "gt", Value: 1},
		{Fact: "b", Operator: "lt", Value: 2},
		{Fact: "c", Operator: "eq", Value: 3},
	}}

	visited := 0
	err := WalkLeaves(cond, func(leaf *Condition) error {
		visited++
		if leaf.Fact == "b" {
			return fmt.Errorf("stop at b")
		}
		return nil
	})
	assert.EqualError(t, err, "stop at b")
	assert.Equal(t, 2, visited)
}
