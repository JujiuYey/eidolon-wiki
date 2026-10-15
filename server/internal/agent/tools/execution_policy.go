package tools

// CanRunConcurrently is an explicit allowlist of built-in reads. Mutations,
// arbitrary code, and unknown tools form execution barriers.
func CanRunConcurrently(name string) bool {
	switch name {
	case ToolSearchKnowledge, ToolReadDocument, ToolListDocuments,
		ToolQueryKnowledgeGraph, ToolSearchConversations,
		ToolWikiSearch, ToolWikiReadPage, ToolWikiReadIssue:
		return true
	default:
		return false
	}
}
