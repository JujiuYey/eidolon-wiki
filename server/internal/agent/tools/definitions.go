package tools

// maxFunctionNameLength is the maximum length for a tool/function name
// imposed by the OpenAI API.
const maxFunctionNameLength = 64

// Tool names constants
const (
	ToolThinking  = "thinking"
	ToolTodoWrite = "todo_write"
	// Knowledge retrieval surface. search_knowledge covers semantic, keyword
	// and hybrid retrieval over chunk-indexed knowledge bases; read_document
	// reads a document's metadata and chunks (by page, by chunk handle, or by
	// an in-document text search); list_documents browses one knowledge base.
	ToolSearchKnowledge     = "search_knowledge"
	ToolReadDocument        = "read_document"
	ToolListDocuments       = "list_documents"
	ToolQueryKnowledgeGraph = "query_knowledge_graph"
	ToolSearchConversations = "search_conversations"
	// ToolWebSearch / ToolWebFetch are retired: the web search & fetch tools
	// were removed from this deployment. The names stay so stored agent
	// allowlists (AllowedTools) that name them keep decoding; they are always
	// filtered out before tool registration.
	ToolWebSearch = "web_search"
	ToolWebFetch  = "web_fetch"
	// Wiki-related tools (only available when wiki KBs are in scope)
	ToolWikiReadPage    = "wiki_read_page"
	ToolWikiWritePage   = "wiki_write_page"
	ToolWikiReplaceText = "wiki_replace_text"
	ToolWikiRenamePage  = "wiki_rename_page"
	ToolWikiDeletePage  = "wiki_delete_page"
	ToolWikiSearch      = "wiki_search"
	ToolWikiFlagIssue   = "wiki_flag_issue"
	ToolWikiReadIssue   = "wiki_read_issue"
	ToolWikiUpdateIssue = "wiki_update_issue"
)

// AvailableTool defines a simple tool metadata used by settings APIs.
type AvailableTool struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// AvailableToolDefinitions returns the list of tools exposed to the UI.
// Keep this in sync with registered tools in this package.
func AvailableToolDefinitions() []AvailableTool {
	return []AvailableTool{
		{Name: ToolThinking, Label: "思考", Description: "动态和反思性的问题解决思考工具"},
		{Name: ToolTodoWrite, Label: "制定计划", Description: "创建结构化的研究计划"},
		{Name: ToolSearchKnowledge, Label: "检索知识库", Description: "语义、关键词或混合检索知识库分块"},
		{Name: ToolReadDocument, Label: "阅读文档", Description: "读取文档元数据与分块内容，支持分页和文内查找"},
		{Name: ToolListDocuments, Label: "浏览文档列表", Description: "分页列出知识库中的文档"},
		{Name: ToolQueryKnowledgeGraph, Label: "查询知识图谱", Description: "从知识图谱中查询关系"},
		{
			Name:        ToolSearchConversations,
			Label:       "回顾历史对话",
			Description: "在用户自己的历史会话中查找之前聊过的内容",
		},
		{Name: ToolWikiReadPage, Label: "读取Wiki页面", Description: "读取指定的Wiki页面内容"},
		{Name: ToolWikiSearch, Label: "搜索Wiki", Description: "在Wiki中搜索页面"},
		{Name: ToolWikiFlagIssue, Label: "标记Wiki问题", Description: "标记页面中存在的事实错误或合并冲突问题"},
		{Name: ToolWikiWritePage, Label: "创建/覆盖Wiki", Description: "创建新页面或完全覆盖已有页面"},
		{Name: ToolWikiReplaceText, Label: "局部替换Wiki", Description: "替换Wiki页面中的特定文本"},
		{Name: ToolWikiRenamePage, Label: "重命名Wiki", Description: "重命名Wiki页面并自动更新关联链接"},
		{Name: ToolWikiDeletePage, Label: "删除Wiki", Description: "删除Wiki页面并自动清理关联死链"},
		{Name: ToolWikiReadIssue, Label: "查看Wiki问题", Description: "查看特定的Wiki页面问题详情"},
		{Name: ToolWikiUpdateIssue, Label: "更新Wiki问题状态", Description: "更新特定的Wiki页面问题状态"},
	}
}

// DefaultAllowedTools returns the default allowed tools list.
func DefaultAllowedTools() []string {
	return []string{
		ToolSearchKnowledge,
		ToolReadDocument,
		ToolListDocuments,
		// Looking up what this user asked before is only ever a read of their
		// own history, and it is what lets "上次你给我的那个配置" resolve at all
		// without stuffing every past conversation into the context window.
		ToolSearchConversations,
		// Graph, SQL, data analysis and explicit planning are opt-in. Existing
		// agents keep their explicit allowlists; domain presets select extras.
	}
}

// Retired tool identifiers are retained only for decoding existing histories
// and ignoring obsolete allowlist entries. They have no implementation or
// registration path and are never offered in model tool schemas. The sandbox
// and skill surface (read_file, shell_exec, write/edit_sandbox_file,
// execute_skill_script, read_skill, read_sandbox_file) was removed with the
// sandbox subsystem and has no successor.
const (
	// The pre-consolidation knowledge retrieval surface. knowledge_search and
	// grep_chunks were folded into search_knowledge (mode=semantic|keyword|
	// hybrid); list_knowledge_chunks, get_document_info and
	// wiki_read_source_doc were folded into read_document.
	LegacyToolKnowledgeSearch     = "knowledge_search"
	LegacyToolGrepChunks          = "grep_chunks"
	LegacyToolListKnowledgeChunks = "list_knowledge_chunks"
	LegacyToolGetDocumentInfo     = "get_document_info"
	LegacyToolWikiReadSourceDoc   = "wiki_read_source_doc"

	// Long-term memory search, removed with the memory subsystem. Stored
	// histories still replay its results and stale allowlists still name it.
	ToolSearchMemory = "search_memory"
)

// legacyToolSuccessors maps retired allowlist entries to the tool that now
// provides the capability. Stored agent configurations, presets and API
// callers keep working without a data migration: NormalizeAllowedTools maps
// them at registration time.
var legacyToolSuccessors = map[string]string{
	LegacyToolKnowledgeSearch:     ToolSearchKnowledge,
	LegacyToolGrepChunks:          ToolSearchKnowledge,
	LegacyToolListKnowledgeChunks: ToolReadDocument,
	LegacyToolGetDocumentInfo:     ToolReadDocument,
	LegacyToolWikiReadSourceDoc:   ToolReadDocument,
}

// SuccessorToolName returns the current tool that replaces a retired
// allowlist entry, or name itself when it is not retired.
func SuccessorToolName(name string) string {
	if successor, ok := legacyToolSuccessors[name]; ok {
		return successor
	}
	return name
}

// IsLegacyRetrievalTool reports whether name is a retired knowledge retrieval
// tool that NormalizeAllowedTools rewrites to its successor.
func IsLegacyRetrievalTool(name string) bool {
	_, ok := legacyToolSuccessors[name]
	return ok
}

// NormalizeAllowedTools rewrites retired tool names in an allowlist to their
// successors and drops duplicates while preserving first-seen order.
func NormalizeAllowedTools(allowed []string) []string {
	if len(allowed) == 0 {
		return allowed
	}
	seen := make(map[string]struct{}, len(allowed))
	out := make([]string, 0, len(allowed))
	for _, name := range allowed {
		name = SuccessorToolName(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// RetiredToolReplacement tells the model how to replace a removed tool.
// Empty when name was never a WeKnora tool.
func RetiredToolReplacement(name string) string {
	switch name {
	case "execute_skill_script", "read_skill", "read_sandbox_file", "read_file",
		"shell_exec", "list_sandbox_files", "write_sandbox_file", "edit_sandbox_file",
		"write_skill_file", "edit_skill_file":
		return "the sandbox and skill tooling was removed from this deployment; no equivalent tool exists"
	case "discover_mcp_tools", "call_mcp_tool":
		return "external MCP tool integration was removed from this deployment; " +
			"use the built-in knowledge and wiki tools instead"
	case ToolSearchMemory:
		return "long-term memory was removed from this deployment; no equivalent tool exists"
	case LegacyToolKnowledgeSearch:
		return "knowledge_search is no longer available; use search_knowledge(query=..., mode=\"semantic\"|\"hybrid\")"
	case LegacyToolGrepChunks:
		return "grep_chunks is no longer available; use search_knowledge(query=..., mode=\"keyword\") for exact " +
			"terms, or read_document(id=dN, query=...) to search inside one document"
	case LegacyToolListKnowledgeChunks:
		return "list_knowledge_chunks is no longer available; use read_document(id=dN or cN, offset=..., limit=...)"
	case LegacyToolGetDocumentInfo:
		return "get_document_info is no longer available; read_document(id=dN) returns the document metadata " +
			"with its first page"
	case LegacyToolWikiReadSourceDoc:
		return "wiki_read_source_doc is no longer available; use read_document(id=dN, query=... or offset=...)"
	default:
		return ""
	}
}
