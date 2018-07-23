package template

import (
	"errors"
	"log"

	"github.com/crcc/jsonp/engine"
)

const (
	LoggerKey            = "logger"
	EngineKey            = "engine"
	CollectingContentKey = "collectingContent"
	ExpanderConfigKey    = "expanderConfig"
	RenderConfigKey      = "renderConfig"
)

func GetLogger(ctx engine.Context) *log.Logger {
	return ctx.Get(LoggerKey).(*log.Logger)
}

func GetEngine(ctx engine.Context) *SDocEngine {
	return ctx.Get(EngineKey).(*SDocEngine)
}

func GetExpanderConfig(ctx engine.Context) engine.Exp {
	return ctx.Get(ExpanderConfigKey).(engine.Exp)
}

func GetRenderConfig(ctx engine.Context) engine.Exp {
	return ctx.Get(RenderConfigKey).(engine.Exp)
}

func GetCollectingContent(ctx engine.Context) bool {
	val := ctx.Get(CollectingContentKey)
	if val == nil {
		return false
	}
	if v, ok := val.(bool); !ok {
		return false
	} else {
		return v
	}
}

const (
	RootDocRedexName   = "document"
	RootValueRedexName = "value"
)

var (
	ErrInvalidRootExpander = errors.New("Invalid Root Expander")
	ErrMissingRootRender   = errors.New("Missing Root Render")
	ErrMissingRootExpander = errors.New("Missing Root Expander")
)

type SDocEngine struct {
	parser engine.Parser
	logger *log.Logger
}
