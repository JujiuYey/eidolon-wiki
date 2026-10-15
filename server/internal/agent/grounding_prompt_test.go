package agent

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGroundingGuidanceSurvivesTemplateSelection(t *testing.T) {
	data, err := os.ReadFile("../../config/prompt_templates/agent_system_prompt.yaml")
	require.NoError(t, err)
	var file struct {
		Templates []config.PromptTemplate `yaml:"templates"`
	}
	require.NoError(t, yaml.Unmarshal(data, &file))
	cfg := &config.Config{PromptTemplates: &config.PromptTemplatesConfig{AgentSystemPrompt: file.Templates}}
	for _, tc := range []struct {
		name   string
		kbs    []*KnowledgeBaseInfo
		custom string
	}{
		{name: "pure"},
		{name: "rag", kbs: []*KnowledgeBaseInfo{{ID: "kb", Capabilities: []string{"chunks"}}}},
		{name: "saved custom", custom: "Create the requested slides using the selected skill."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prompt := BuildSystemPromptWithOptions(tc.kbs, false, &BuildSystemPromptOptions{Config: cfg}, tc.custom)
			require.Contains(t, prompt, "consult relevant available sources before drafting unsupported content")
			require.Contains(t, prompt, "presentations, reports, tutorials, and technical instructions")
			require.Contains(t, prompt, "Reading a generator's instructions or successfully running its script does not verify the subject matter")
			require.Contains(t, prompt, "translation or formatting of supplied content do not require research")
			require.Contains(t, prompt, "If relevant sources are unavailable or searches leave gaps")
			if tc.custom != "" {
				require.True(t, strings.HasPrefix(prompt, tc.custom))
			}
		})
	}
	// The legacy builder cannot infer tools, but still needs the evidence policy.
	require.Contains(t, BuildSystemPrompt(nil, false, "Custom"), "Content grounding")
}

func TestGroundingUsesRegistryInsteadOfConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name       string
		registered []string
		webFlag    bool
		want       []string
		absent     []string
	}{
		{
			"non-knowledge tools only with stale flags",
			[]string{tools.ToolTodoWrite, "pptx_generator"},
			true,
			nil,
			[]string{
				"Available knowledge tools:", "web_search is available", "web_fetch is available",
				"search them before answering",
			},
		},
		{
			"rag",
			[]string{tools.ToolSearchKnowledge, tools.ToolReadDocument},
			false,
			[]string{
				"Available knowledge tools: search_knowledge, read_document",
				"search them before answering a question about a topic, even one that reads like general knowledge",
			},
			[]string{"wiki_search", "web_search is available"},
		},
		{
			"wiki",
			[]string{tools.ToolWikiReadPage, tools.ToolWikiSearch},
			false,
			[]string{"Available knowledge tools: wiki_search, wiki_read_page"},
			[]string{"search_knowledge", "web_search is available"},
		},
		{
			"retired web tools stay unadvertised",
			[]string{tools.ToolWebSearch, tools.ToolWebFetch},
			false,
			nil,
			[]string{"Available knowledge tools:", "web_search is available", "web_fetch is available"},
		},
		{
			"no tools", nil, false, nil,
			[]string{"Available knowledge tools:", "web_search is available", "web_fetch is available"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := newTestEngine(t, &mockChat{})
			engine.config.WebSearchEnabled = tc.webFlag
			engine.config.AllowedTools = []string{tools.ToolSearchKnowledge, tools.ToolWebSearch}
			engine.toolRegistry = tools.NewToolRegistry()
			for _, name := range tc.registered {
				engine.toolRegistry.RegisterTool(newCountingTool(name))
			}
			prompt := engine.buildSystemPrompt(context.Background())
			require.Contains(t, prompt, "Content grounding")
			for _, want := range tc.want {
				require.Contains(t, prompt, want)
			}
			for _, absent := range tc.absent {
				require.NotContains(t, prompt, absent)
			}
		})
	}
}
