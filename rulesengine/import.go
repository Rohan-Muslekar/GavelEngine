package rulesengine

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

func LoadRulesFromJSON(data []byte) ([]*Rule, error) {
	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}
	result := make([]*Rule, len(rules))
	for i := range rules {
		result[i] = &rules[i]
	}
	return result, nil
}

func LoadRulesFromYAML(data []byte) ([]*Rule, error) {
	var rules []Rule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, err
	}
	result := make([]*Rule, len(rules))
	for i := range rules {
		result[i] = &rules[i]
	}
	return result, nil
}

func (e *Engine) ExportRulesJSON() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return json.Marshal(e.rules)
}

func (e *Engine) ExportRulesYAML() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return yaml.Marshal(e.rules)
}
