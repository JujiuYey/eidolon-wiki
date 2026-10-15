package types

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"
)

// DefaultMaxContextTokens is the context window assumed for a model that does
// not declare one. 200k matches typical windows on current chat/VLM models
// (GPT-4.1, Claude, Gemini, Qwen, etc.).
//
// It is still below the largest windows on the market. Guessing high is
// the dangerous direction: compaction never fires, and the first sign of
// trouble is the provider rejecting the request. Guessing low only costs some
// history that would have fit.
const DefaultMaxContextTokens = 200000

// AgentMaxContextTokens picks the context window for one agent run. An
// explicit agent setting wins, then whatever the model declares, then the
// assumed default.
func AgentMaxContextTokens(configured, modelContextWindow int) int {
	if configured > 0 {
		return configured
	}
	if modelContextWindow > 0 {
		return modelContextWindow
	}
	return DefaultMaxContextTokens
}

// DefaultSmartReasoningMaxCompletionTokens is the per-round budget for
// smart-reasoning agents. 4096 matches typical OpenAI-compatible provider
// defaults and is enough for ordinary tool-call JSON.
const DefaultSmartReasoningMaxCompletionTokens = 4096

// DefaultQuickAnswerMaxCompletionTokens is the RAG answer budget.
const DefaultQuickAnswerMaxCompletionTokens = 2048

// DefaultMaxCompletionTokens is the unset-config default for one agent:
// quick-answer 2048, smart-reasoning 4096.
func DefaultMaxCompletionTokens(agentMode string) int {
	if agentMode == AgentModeSmartReasoning {
		return DefaultSmartReasoningMaxCompletionTokens
	}
	return DefaultQuickAnswerMaxCompletionTokens
}

// AgentRoundMaxCompletionTokens returns the completion-token budget for one
// ReAct LLM round. Zero (unset) uses DefaultSmartReasoningMaxCompletionTokens.
// An explicit configured value is honored as-is.
func AgentRoundMaxCompletionTokens(configured int) int {
	if configured > 0 {
		return configured
	}
	return DefaultSmartReasoningMaxCompletionTokens
}

// UnlimitedMaxIterations is the stored MaxIterations value meaning the ReAct
// loop has no round cap. Zero still means "unset, apply default" so agents
// that omit the field keep their current behaviour.
const UnlimitedMaxIterations = -1

// AgentConfig represents the full agent configuration (used at tenant level and runtime)
// This includes all configuration parameters for agent execution
type AgentConfig struct {
	// MaxIterations caps ReAct rounds. Zero is unset (filled with a default);
	// a negative value is unlimited — the loop runs until the model stops,
	// the user cancels, or another guard fires. See UnlimitedMaxIterations.
	MaxIterations  int      `json:"max_iterations"`
	AllowedTools   []string `json:"allowed_tools"`           // List of allowed tool names
	Temperature    float64  `json:"temperature"`             // LLM temperature for agent
	KnowledgeBases []string `json:"knowledge_bases"`         // Accessible knowledge base IDs
	KnowledgeIDs   []string `json:"knowledge_ids"`           // Accessible knowledge IDs (individual documents)
	SystemPrompt   string   `json:"system_prompt,omitempty"` // Unified system prompt (uses web_search_status placeholder for dynamic behavior)
	// Deprecated: Use SystemPrompt instead. Kept so stored agent configs
	// predating the unified prompt keep decoding.
	SystemPromptWebEnabled  string `json:"system_prompt_web_enabled,omitempty"`  // Deprecated: prompt when web search was enabled
	SystemPromptWebDisabled string `json:"system_prompt_web_disabled,omitempty"` // Deprecated: prompt when web search was disabled
	UseCustomSystemPrompt   bool   `json:"use_custom_system_prompt"`             // Whether to use custom system prompt instead of default
	// Web search settings are retired: the web search/fetch toolset was
	// removed from this deployment. The fields stay so stored agent configs
	// keep decoding; nothing reads them at runtime anymore.
	WebSearchEnabled    bool          `json:"web_search_enabled"`               // retired: web search tool flag
	WebSearchMaxResults int           `json:"web_search_max_results"`           // retired: max web search results
	WebSearchProviderID string        `json:"web_search_provider_id,omitempty"` // retired: web search provider ID
	MultiTurnEnabled    bool          `json:"multi_turn_enabled"`               // Whether multi-turn conversation is enabled
	SearchTargets       SearchTargets `json:"-"`                                // Pre-computed unified search targets (runtime only)
	// Whether to enable thinking mode (for models that support extended thinking)
	Thinking *bool `json:"thinking"`
	// ReasoningEffort is the graded thinking level; empty falls back to Thinking.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	// Whether final answers include knowledge/web source citations. Nil defaults to true.
	CitationEnabled *bool `json:"citation_enabled"`
	// Whether to retrieve knowledge base only when explicitly mentioned with @ (default: false)
	RetrieveKBOnlyWhenMentioned bool `json:"retrieve_kb_only_when_mentioned"`

	// Whether to retain retrieval history (like wiki_read_page results) across turns (default: false)
	RetainRetrievalHistory bool `json:"retain_retrieval_history"`

	// Runtime-only fields (not persisted)
	ChatModelSupportsVision bool   `json:"-"` // Resolved model capability, never supplied by tool input.
	VLMModelID              string `json:"-"` // VLM model ID for tool images, resolved from CustomAgent config.
	// QuestionOrigin is the knowledge source of a suggested question the user
	// picked, already checked to be inside KnowledgeBases (runtime only;
	// rendered into runtime_context as a retrieval hint).
	QuestionOrigin *QuestionOrigin `json:"-"`
	// WritableKBIDs are the SearchTargets KBs this caller may modify (its own
	// workspace's). Search targets only need read access, so tools that write
	// (wiki pages and issues) are limited to this set; empty means read-only.
	// Runtime only, derived per turn.
	WritableKBIDs []string `json:"-"`
	// LLM call timeout in seconds (default: 120). Controls the maximum time for a single LLM call.
	LLMCallTimeout int `json:"llm_call_timeout,omitempty"`

	// Maximum completion tokens for each ReAct LLM round. Zero means
	// DefaultMaxCompletionTokens for this agent's mode. Explicit values are
	// sent as-is.
	MaxCompletionTokens int `json:"max_completion_tokens,omitempty"`

	// Maximum character length for tool output (default: 16000).
	// Outputs exceeding this limit are truncated with head + tail preservation.
	MaxToolOutputChars int `json:"max_tool_output_chars,omitempty"`

	// Maximum context window tokens for the agent. Zero means "use the
	// model's context_window, or DefaultMaxContextTokens (200000)".
	MaxContextTokens int `json:"max_context_tokens,omitempty"`

	// ContextTokenScale is the provider's tokens per estimated token that the
	// session's last calibrated turn measured (TokenUsage.ContextTokenScale).
	// The engine starts its estimator at this scale. Zero means uncalibrated.
	// Runtime only.
	ContextTokenScale float64 `json:"-"`

	// How much recent conversation a compaction keeps verbatim. Zero means
	// compaction.DefaultKeepRecentTokens, scaled down on small windows. This
	// is the knob that decides how often compaction runs: whatever the context
	// grew to, it comes out of a compaction at roughly this size.
	CompactionKeepRecentTokens int `json:"compaction_keep_recent_tokens,omitempty"`

	// Whether to execute independent tool calls in parallel (default: false).
	// When enabled and the LLM returns multiple tool calls, they run concurrently via errgroup.
	ParallelToolCalls bool `json:"parallel_tool_calls,omitempty"`
}

// UnlimitedIterations reports whether the ReAct loop has no round cap.
func (c *AgentConfig) UnlimitedIterations() bool {
	return c != nil && c.MaxIterations < 0
}

// CitationsEnabled preserves citation output for legacy runtime configs that
// predate the setting and therefore have a nil CitationEnabled value.
func (c *AgentConfig) CitationsEnabled() bool {
	return c == nil || c.CitationEnabled == nil || *c.CitationEnabled
}

// SessionAgentConfig represents session-level agent configuration
// Sessions only store Enabled and KnowledgeBases; other configs are read from Tenant at runtime
type SessionAgentConfig struct {
	AgentModeEnabled bool `json:"agent_mode_enabled"` // Whether agent mode is enabled for this session
	// WebSearchEnabled is retired: web search is removed from this
	// deployment. The field stays so stored session configs keep decoding.
	WebSearchEnabled bool     `json:"web_search_enabled"` // retired: web search flag
	KnowledgeBases   []string `json:"knowledge_bases"`    // Accessible knowledge base IDs for this session
	KnowledgeIDs     []string `json:"knowledge_ids"`      // Accessible knowledge IDs (individual documents) for this session
}

// Value implements driver.Valuer interface for AgentConfig
func (c AgentConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner interface for AgentConfig
func (c *AgentConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return nil
	}
	return json.Unmarshal(b, c)
}

// Value implements driver.Valuer interface for SessionAgentConfig
func (c SessionAgentConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner interface for SessionAgentConfig
func (c *SessionAgentConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return nil
	}
	return json.Unmarshal(b, c)
}

// ResolveSystemPrompt returns the prompt template. It uses the unified
// SystemPrompt field, falling back to the deprecated non-web variant for
// agents stored before the unified prompt existed (web search is retired, so
// the web-enabled variant is never selected anymore).
func (c *AgentConfig) ResolveSystemPrompt() string {
	if c == nil {
		return ""
	}

	if c.SystemPrompt != "" {
		return c.SystemPrompt
	}

	if c.SystemPromptWebDisabled != "" {
		return c.SystemPromptWebDisabled
	}

	return ""
}

// Tool defines the interface that all agent tools must implement
type Tool interface {
	// Name returns the unique identifier for this tool
	Name() string

	// Description returns a human-readable description of what the tool does
	Description() string

	// Parameters returns the JSON Schema for the tool's parameters
	Parameters() json.RawMessage

	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error)
}

// Cleanable is an optional interface that tools can implement to release resources.
// Tools implementing this interface will have their Cleanup method called during
// registry cleanup (e.g., at the end of an agent session).
type Cleanable interface {
	Cleanup(ctx context.Context)
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	// OutputFiles holds sandbox references for this live result only. History
	// uses the final answer's persistent resource references instead.
	// A non-nil empty slice means output inspection found no eligible changes;
	// nil means no output inspection result is available.
	OutputFiles []string               `json:"-"`
	Success     bool                   `json:"success"`          // Whether the tool executed successfully
	Output      string                 `json:"output"`           // Human-readable output
	Data        map[string]interface{} `json:"data,omitempty"`   // Structured data for programmatic use
	Error       string                 `json:"error,omitempty"`  // Error message if execution failed
	Images      []string               `json:"images,omitempty"` // Base64 data URIs from tool results
}

// ToolCall represents a single tool invocation within an agent step
type ToolCall struct {
	ID               string                 `json:"id"`                          // Function call ID from LLM
	Name             string                 `json:"name"`                        // Tool name
	Args             map[string]interface{} `json:"args"`                        // Tool arguments
	Result           *ToolResult            `json:"result"`                      // Execution result (contains Output)
	Reflection       string                 `json:"reflection,omitempty"`        // Agent's reflection on this tool call result (if enabled)
	Duration         int64                  `json:"duration"`                    // Execution time in milliseconds
	ProviderMetadata ToolCallMetadata       `json:"provider_metadata,omitempty"` // Provider-specific tool-call state for replay
}

// PipelineToolCallIDPrefix marks a persisted tool call the model never made.
// The fast-answer (KnowledgeQA) pipeline records its retrieval stages as tool
// calls so a reloaded conversation can redraw the same timeline it showed while
// streaming. History replay must skip them: asking the model to account for
// calls it never issued, against tools it may not even have, breaks the
// request protocol.
const PipelineToolCallIDPrefix = "ragpipe-"

// IsPipelineToolCallID reports whether a tool call was synthesized by the
// fast-answer pipeline rather than requested by the model.
func IsPipelineToolCallID(id string) bool {
	return strings.HasPrefix(id, PipelineToolCallIDPrefix)
}

// AgentStep represents one iteration of the ReAct loop
type AgentStep struct {
	Iteration int    `json:"iteration"` // Iteration number (0-indexed)
	Thought   string `json:"thought"`   // LLM's reasoning/thinking (Think phase)
	// UserMessagesBefore records consumed steer rows in delivery order, before
	// this model response. Unlike timestamps, this remains unambiguous on replay.
	UserMessagesBefore []string `json:"user_messages_before,omitempty"`
	// IntermediateAnswer preserves a plain answer followed by a loop-end steer.
	// The canonical final answer is still stored in Message.Content.
	IntermediateAnswer bool `json:"intermediate_answer,omitempty"`
	// ReasoningContent stores the OpenAI-protocol reasoning_content emitted by the
	// model in this round. Persisted on AgentStep so cross-turn replay can put it
	// back on the assistant message — required by MiMo / DeepSeek V3.2+ thinking
	// mode, ignored by providers that don't recognize the field.
	ReasoningContent string `json:"reasoning_content,omitempty"`
	// ReasoningSignature / ReasoningMetadata are the provider artifacts that
	// must accompany ReasoningContent on replay (Anthropic signatures,
	// OpenAI Responses encrypted reasoning items).
	ReasoningSignature string           `json:"reasoning_signature,omitempty"`
	ReasoningMetadata  ProviderMetadata `json:"reasoning_metadata,omitempty"`
	ToolCalls          []ToolCall       `json:"tool_calls"` // Tools called in this step (Act phase)
	Timestamp          time.Time        `json:"timestamp"`  // When this step occurred
	// Truncated marks the round the completion-token cap cut off. It rides in
	// the agent_steps JSON so a reloaded, shared or re-opened conversation can
	// still show that the answer stops mid-sentence by design, rather than
	// looking finished. Live streaming carries the same fact on the answer
	// event; this is what survives the round trip.
	Truncated bool `json:"truncated,omitempty"`
}

// GetObservations returns observations from all tool calls in this step
// This is a convenience method to maintain backward compatibility
func (s *AgentStep) GetObservations() []string {
	observations := make([]string, 0, len(s.ToolCalls))
	for _, tc := range s.ToolCalls {
		if tc.Result != nil && tc.Result.Output != "" {
			observations = append(observations, tc.Result.Output)
		}
		if tc.Reflection != "" {
			observations = append(observations, "Reflection: "+tc.Reflection)
		}
	}
	return observations
}

// AgentState tracks the execution state of an agent across iterations
type AgentState struct {
	PendingSteerMessages []string        `json:"-"`
	CurrentRound         int             `json:"current_round"`  // Current round number
	RoundSteps           []AgentStep     `json:"round_steps"`    // All steps taken so far in the current round
	IsComplete           bool            `json:"is_complete"`    // Whether agent has finished
	FinalAnswer          string          `json:"final_answer"`   // The final answer to the query
	KnowledgeRefs        []*SearchResult `json:"knowledge_refs"` // Collected knowledge references
	TurnUsage            TokenUsage      `json:"turn_usage"`     // LLM usage accumulated across this turn
}

// FunctionDefinition represents a function definition for LLM function calling
type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}
