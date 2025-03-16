package rulesengine

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

type EngineManager struct {
	engines sync.Map
}

func NewEngineManager() *EngineManager {
	return &EngineManager{
		engines: sync.Map{},
	}
}

func (em *EngineManager) CreateEngine(name string) *Engine {
	engine := NewEngine()
	em.engines.Store(name, engine)
	return engine
}

func (em *EngineManager) GetEngine(name string) *Engine {
	if engine, ok := em.engines.Load(name); ok {
		return engine.(*Engine)
	}
	log.Error().Msg(fmt.Sprintf("Engine %s not found", name))
	return nil
}

func (em *EngineManager) DeleteEngine(name string) {
	em.engines.Delete(name)
}

func (em *EngineManager) GetEngines() map[string]*Engine {
	engines := make(map[string]*Engine)
	em.engines.Range(func(key, value interface{}) bool {
		engines[key.(string)] = value.(*Engine)
		return true
	})
	return engines
}
