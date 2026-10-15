package agent

import (
	"encoding/json"
	"sort"

	"github.com/Tencent/WeKnora/internal/modelcontext"
	"github.com/Tencent/WeKnora/internal/types"
)

// argKeys returns the sorted list of top-level keys in a tool's argument
// map. Used when the audit avoids echoing raw arguments (e.g.
// database_query's SQL) but still wants to signal what was passed in.
func argKeys(args map[string]any) []string {
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// traceArgumentValue keeps valid model-emitted JSON structured in the audit
// while preserving malformed payloads verbatim for diagnosis.
func traceArgumentValue(raw string) interface{} {
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	return value
}

// buildToolSpanInput exposes both sides of the model-context boundary:
// model_arguments is exactly what the model emitted (including temporary
// handles), while resolved_arguments is the durable payload actually executed.
// The mapping itself never happens here; this only observes the registry's
// audit, which is what the modelcontext encode/decode tests assert on.
func buildToolSpanInput(tc types.LLMToolCall, resolvedArgs map[string]any, sensitive bool) map[string]interface{} {
	modelArguments := tc.ModelArguments
	if modelArguments == "" {
		modelArguments = tc.Function.Arguments
	}
	resolution := tc.ArgumentResolution
	if resolution == "" {
		resolution = modelcontext.ArgumentResolutionUnchanged
	}
	if sensitive {
		modelArgKeys := []string(nil)
		if parsed, ok := traceArgumentValue(modelArguments).(map[string]interface{}); ok {
			modelArgKeys = argKeys(parsed)
		}
		return map[string]interface{}{
			"tool_call_id":            tc.ID,
			"model_arg_keys":          modelArgKeys,
			"resolved_arg_keys":       argKeys(resolvedArgs),
			"argument_resolution":     resolution,
			"unresolved_handle_count": len(tc.UnresolvedHandles),
			"args_redacted":           true,
		}
	}
	return map[string]interface{}{
		"tool_call_id":        tc.ID,
		"model_arguments":     traceArgumentValue(modelArguments),
		"resolved_arguments":  resolvedArgs,
		"argument_resolution": resolution,
		"unresolved_handles":  tc.UnresolvedHandles,
	}
}
