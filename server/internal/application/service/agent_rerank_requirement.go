package service

import (
	"github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/types"
)

// agentRequiresRerankModel returns true when the agent's configured scope and
// tools will actually invoke the reranker at runtime. An agent whose knowledge
// base scope is explicitly disabled cannot run search_knowledge, so it does not
// need a rerank model even if that tool remains in AllowedTools.
//
// This mirrors the runtime check in session_agent_qa.go: only
// `search_knowledge` with an enabled knowledge-base scope uses the reranker.
// Wiki-first agents (wiki_search / wiki_read_page / …) never call it and
// therefore don't need a rerank model configured, even when knowledge bases
// are attached.
//
// When AllowedTools is empty the runtime falls back to
// tools.DefaultAllowedTools(), which includes search_knowledge, so we treat
// that case as requiring the reranker.
func agentRequiresRerankModel(agent *types.CustomAgent) bool {
	if agent == nil {
		return false
	}
	if agent.Config.KBSelectionMode == "none" {
		return false
	}
	allowed := agent.Config.AllowedTools
	if len(allowed) == 0 {
		allowed = tools.DefaultAllowedTools()
	}
	for _, t := range allowed {
		if tools.SuccessorToolName(t) == tools.ToolSearchKnowledge {
			return true
		}
	}
	return false
}
