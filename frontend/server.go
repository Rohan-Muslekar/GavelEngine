// frontend/server.go
package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/Rohan-Muslekar/GavelEngine/rulesengine" // Update this import path
	"github.com/gin-gonic/gin"
)

// Global engine manager
var engineManager *rulesengine.EngineManager

// Store for function facts (since we can't send them directly as JSON)
var functionFacts = struct {
	sync.RWMutex
	facts map[string]map[string]rulesengine.FactFunc
}{facts: make(map[string]map[string]rulesengine.FactFunc)}

// ConditionJSON is used for JSON binding when receiving conditions
type ConditionJSON struct {
	All          []ConditionJSON        `json:"all,omitempty"`
	Any          []ConditionJSON        `json:"any,omitempty"`
	Not          *ConditionJSON         `json:"not,omitempty"`
	Fact         string                 `json:"fact,omitempty"`
	Operator     string                 `json:"operator,omitempty"`
	Value        interface{}            `json:"value,omitempty"`
	Params       map[string]interface{} `json:"params,omitempty"`
	Path         string                 `json:"path,omitempty"`
	ConditionRef string                 `json:"condition,omitempty"`
}

// Convert JSON condition to engine condition
func (c *ConditionJSON) ToEngineCondition() rulesengine.Condition {
	condition := rulesengine.Condition{
		Fact:         c.Fact,
		Operator:     c.Operator,
		Value:        c.Value,
		Params:       c.Params,
		Path:         c.Path,
		ConditionRef: c.ConditionRef,
	}

	if len(c.All) > 0 {
		condition.All = make([]rulesengine.Condition, len(c.All))
		for i, cond := range c.All {
			condition.All[i] = cond.ToEngineCondition()
		}
	}

	if len(c.Any) > 0 {
		condition.Any = make([]rulesengine.Condition, len(c.Any))
		for i, cond := range c.Any {
			condition.Any[i] = cond.ToEngineCondition()
		}
	}

	if c.Not != nil {
		not := c.Not.ToEngineCondition()
		condition.Not = &not
	}

	return condition
}

func main() {
	// Initialize engine manager
	engineManager = rulesengine.NewEngineManager()

	// Create Gin router
	router := gin.Default()

	// Serve static files
	router.Static("/static", "./frontend/static")
	router.StaticFile("/", "./frontend/static/index.html")

	// API endpoints
	api := router.Group("/api")
	{
		// Engine management
		api.GET("/engines", listEngines)
		api.POST("/engines", createEngine)
		api.GET("/engines/:name", getEngine)
		api.DELETE("/engines/:name", deleteEngine)

		// Fact management
		api.GET("/engines/:name/facts", listFacts)
		api.POST("/engines/:name/facts", addFact)
		api.DELETE("/engines/:name/facts/:id", removeFact)

		// Rule management
		api.GET("/engines/:name/rules", listRules)
		api.POST("/engines/:name/rules", addRule)
		api.GET("/engines/:name/rules/:ruleName", getRule)
		api.DELETE("/engines/:name/rules/:ruleName", removeRule)

		// Engine execution
		api.POST("/engines/:name/run", runEngine)

		// Predefined facts
		api.GET("/predefined-facts", getPredefinedFacts)
	}

	fmt.Println("Server running on http://localhost:8080")
	router.Run(":8080")
}

// List all engines
func listEngines(c *gin.Context) {
	engines := engineManager.GetEngines()
	names := make([]string, 0, len(engines))
	for name := range engines {
		names = append(names, name)
	}

	c.JSON(http.StatusOK, gin.H{"engines": names})
}

// Create a new engine
func createEngine(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	engineManager.CreateEngine(req.Name)

	// Initialize function facts map for this engine
	functionFacts.Lock()
	functionFacts.facts[req.Name] = make(map[string]rulesengine.FactFunc)
	functionFacts.Unlock()

	c.JSON(http.StatusCreated, gin.H{"name": req.Name, "status": "created"})
}

// Get engine details
func getEngine(c *gin.Context) {
	name := c.Param("name")
	engine := engineManager.GetEngine(name)
	if engine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Engine not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"name": name})
}

// Delete an engine
func deleteEngine(c *gin.Context) {
	name := c.Param("name")
	engineManager.DeleteEngine(name)

	// Clean up function facts for this engine
	functionFacts.Lock()
	delete(functionFacts.facts, name)
	functionFacts.Unlock()

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// List facts for an engine
func listFacts(c *gin.Context) {
	name := c.Param("name")
	engine := engineManager.GetEngine(name)
	if engine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Engine not found"})
		return
	}

	// In a real implementation, you'd need to add a method to get facts from an engine
	// For now, return a sample list
	facts := []map[string]interface{}{
		{
			"id":         "age",
			"isConstant": false,
			"cache":      true,
		},
		{
			"id":         "score",
			"isConstant": false,
			"cache":      true,
		},
	}
	c.JSON(http.StatusOK, gin.H{"facts": facts})
}

// Add a fact to an engine
func addFact(c *gin.Context) {
	name := c.Param("name")
	engine := engineManager.GetEngine(name)
	if engine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Engine not found"})
		return
	}

	var req struct {
		ID          string      `json:"id" binding:"required"`
		Type        string      `json:"type" binding:"required"` // "constant" or "function"
		Value       interface{} `json:"value"`
		Description string      `json:"description"` // For function facts
		Cache       bool        `json:"cache"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var factOpts []rulesengine.FactOption
	if !req.Cache {
		factOpts = append(factOpts, rulesengine.WithNoCache())
	}

	if req.Type == "constant" {
		err := engine.AddFact(req.ID, req.Value, factOpts...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if req.Type == "function" {
		// For function facts, we'll register a function that returns a runtime fact
		// This is a simplified approach - in a real implementation, you'd need a more
		// sophisticated way to define and register function facts
		factFunc := func(params map[string]interface{}, almanac *rulesengine.Almanac) (interface{}, error) {
			if val, ok := almanac.GetRuntimeFacts()[req.ID]; ok {
				return val, nil
			}
			return nil, nil
		}

		err := engine.AddFact(req.ID, rulesengine.FactFunc(factFunc), factOpts...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Store the function for later use
		functionFacts.Lock()
		if _, ok := functionFacts.facts[name]; !ok {
			functionFacts.facts[name] = make(map[string]rulesengine.FactFunc)
		}
		functionFacts.facts[name][req.ID] = factFunc
		functionFacts.Unlock()
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid fact type. Must be 'constant' or 'function'"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": req.ID, "status": "created"})
}

// Remove a fact from an engine
func removeFact(c *gin.Context) {
	name := c.Param("name")
	engine := engineManager.GetEngine(name)
	if engine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Engine not found"})
		return
	}

	id := c.Param("id")
	engine.RemoveFact(id)

	// Remove from function facts if it exists
	functionFacts.Lock()
	if engineFacts, ok := functionFacts.facts[name]; ok {
		delete(engineFacts, id)
	}
	functionFacts.Unlock()

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// List rules for an engine
func listRules(c *gin.Context) {
	name := c.Param("name")
	engine := engineManager.GetEngine(name)
	if engine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Engine not found"})
		return
	}

	rules := engine.GetRulesAsJSON()
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// Add a rule to an engine
func addRule(c *gin.Context) {
	name := c.Param("name")
	engine := engineManager.GetEngine(name)
	if engine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Engine not found"})
		return
	}

	var req struct {
		Name       string            `json:"name" binding:"required"`
		Priority   int               `json:"priority"`
		Conditions ConditionJSON     `json:"conditions" binding:"required"`
		Event      rulesengine.Event `json:"event" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	condition := req.Conditions.ToEngineCondition()
	rule := rulesengine.NewRule(condition, req.Event,
		rulesengine.WithName(req.Name),
		rulesengine.WithPriorityForRule(req.Priority))

	engine.AddRule(rule)
	c.JSON(http.StatusCreated, gin.H{"name": req.Name, "status": "created"})
}

// Get rule details
func getRule(c *gin.Context) {
	name := c.Param("name")
	engine := engineManager.GetEngine(name)
	if engine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Engine not found"})
		return
	}

	ruleName := c.Param("ruleName")
	// In a real implementation, you'd need to add a method to get a specific rule
	// For now, return a sample rule
	rule := map[string]interface{}{
		"name":     ruleName,
		"priority": 10,
		"conditions": map[string]interface{}{
			"all": []map[string]interface{}{
				{
					"fact":     "age",
					"operator": "greaterThanInclusive",
					"value":    18,
				},
				{
					"fact":     "age",
					"operator": "lessThanInclusive",
					"value":    25,
				},
			},
		},
		"event": map[string]interface{}{
			"type": "young-adult",
			"params": map[string]interface{}{
				"message": "User is a young adult",
			},
		},
	}
	c.JSON(http.StatusOK, gin.H{"rule": rule})
}

// Remove a rule from an engine
func removeRule(c *gin.Context) {
	name := c.Param("name")
	engine := engineManager.GetEngine(name)
	if engine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Engine not found"})
		return
	}

	ruleName := c.Param("ruleName")
	engine.RemoveRule(ruleName)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// Run an engine with runtime facts
func runEngine(c *gin.Context) {
	name := c.Param("name")
	engine := engineManager.GetEngine(name)
	if engine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Engine not found"})
		return
	}

	var runtimeFacts map[string]interface{}
	if err := c.ShouldBindJSON(&runtimeFacts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := engine.Run(runtimeFacts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert result to JSON-friendly format
	events := make([]map[string]interface{}, len(result.Events))
	for i, event := range result.Events {
		events[i] = map[string]interface{}{
			"type":   event.Type,
			"params": event.Params,
		}
	}

	ruleResults := make([]map[string]interface{}, len(result.RuleResults))
	for i, rr := range result.RuleResults {
		ruleResults[i] = map[string]interface{}{
			"name":    rr.Name,
			"success": rr.Success,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"events":      events,
		"ruleResults": ruleResults,
	})
}

// Get predefined facts
func getPredefinedFacts(c *gin.Context) {
	// This endpoint provides a list of predefined facts that could be used
	// In a real implementation, this might come from a database or config
	predefinedFacts := []map[string]interface{}{
		{
			"id":          "age",
			"description": "User's age in years",
			"usage":       "Add as function and provide in runtime facts",
		},
		{
			"id":          "score",
			"description": "Numeric score value",
			"usage":       "Add as function and provide in runtime facts",
		},
		{
			"id":          "userName",
			"description": "User's name",
			"usage":       "Add as function and provide in runtime facts",
		},
	}
	c.JSON(http.StatusOK, gin.H{"facts": predefinedFacts})
}
