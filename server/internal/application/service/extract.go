package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	chatpipeline "github.com/Tencent/WeKnora/internal/application/service/chat_pipeline"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// NewChunkExtractTask creates a new chunk extract task. It returns
// (enqueued, err): enqueued is true only when a task was actually placed on
// the queue. When NEO4J is disabled the call is a no-op and returns
// (false, nil) — callers that seeded a pending-subtask counter for this chunk
// MUST release that slot, otherwise the parent knowledge stays stuck in
// "finalizing" forever (the graph subtask it's waiting on was never enqueued).
func NewChunkExtractTask(
	ctx context.Context,
	client interfaces.TaskEnqueuer,
	tenantID uint64,
	chunkID string,
	modelID string,
	knowledgeID string,
	attempt int,
	chunkIndex int,
) (bool, error) {
	if strings.ToLower(os.Getenv("NEO4J_ENABLE")) != "true" {
		logger.Warn(ctx, "NEO4J is not enabled, skip chunk extract task")
		return false, nil
	}
	taskPayload := types.ExtractChunkPayload{
		TenantID:    tenantID,
		ChunkID:     chunkID,
		ModelID:     modelID,
		KnowledgeID: knowledgeID,
		Attempt:     attempt,
		ChunkIndex:  chunkIndex,
	}
	payload, err := json.Marshal(taskPayload)
	if err != nil {
		return false, err
	}
	task := asynq.NewTask(types.TypeChunkExtract, payload,
		asynq.Queue(types.QueueGraph), asynq.MaxRetry(3), asynq.Timeout(30*time.Minute))
	info, err := client.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "failed to enqueue task: %v", err)
		return false, fmt.Errorf("failed to enqueue task: %v", err)
	}
	logger.Infof(ctx, "enqueued task: id=%s queue=%s chunk=%s", info.ID, info.Queue, chunkID)
	return true, nil
}

// ChunkExtractService is a service for extracting chunks
type ChunkExtractService struct {
	template          *types.PromptTemplateStructured
	modelService      interfaces.ModelService
	knowledgeBaseRepo interfaces.KnowledgeBaseRepository
	knowledgeRepo     interfaces.KnowledgeRepository
	chunkRepo         interfaces.ChunkRepository
	graphEngine       interfaces.RetrieveGraphRepository
	// spanTracker records this graph-extract task's subspan under the
	// parent attempt's postprocess stage so the trace viewer shows real
	// per-chunk graph extraction time rather than the upstream's enqueue.
	spanTracker SpanTracker
}

// NewChunkExtractService creates a new chunk extract service
func NewChunkExtractService(
	config *config.Config,
	modelService interfaces.ModelService,
	knowledgeBaseRepo interfaces.KnowledgeBaseRepository,
	knowledgeRepo interfaces.KnowledgeRepository,
	chunkRepo interfaces.ChunkRepository,
	graphEngine interfaces.RetrieveGraphRepository,
	spanTracker SpanTracker,
) interfaces.TaskHandler {
	return &ChunkExtractService{
		template:          config.ExtractManager.ExtractGraph,
		modelService:      modelService,
		knowledgeBaseRepo: knowledgeBaseRepo,
		knowledgeRepo:     knowledgeRepo,
		chunkRepo:         chunkRepo,
		graphEngine:       graphEngine,
		spanTracker:       spanTracker,
	}
}

func (s *ChunkExtractService) tracker() SpanTracker {
	if s.spanTracker == nil {
		return noopSpanTracker{}
	}
	return s.spanTracker
}

// Handle handles the chunk extraction task
func (s *ChunkExtractService) Handle(ctx context.Context, t *asynq.Task) error {
	var p types.ExtractChunkPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logger.Errorf(ctx, "failed to unmarshal task payload: %v", err)
		return err
	}
	ctx = logger.WithRequestID(ctx, uuid.New().String())
	ctx = logger.WithField(ctx, "extract", p.ChunkID)
	ctx = context.WithValue(ctx, types.TenantIDContextKey, p.TenantID)

	// A newer attempt (re-upload / edit / reparse) has superseded this one:
	// skip before opening the span or registering the FinalizeSubtask defer.
	// The chunk this task references was deleted by the new attempt's cleanup,
	// and decrementing here would drain the new attempt's counter.
	if attemptSuperseded(ctx, s.tracker(), p.KnowledgeID, p.Attempt) {
		logger.Infof(ctx, "graph extract: attempt %d superseded for %s, skipping stale enrichment",
			p.Attempt, p.KnowledgeID)
		return nil
	}

	// Open a postprocess subspan keyed by chunk ordinal so the trace
	// shows real per-chunk graph extraction time. Skipped silently when
	// upstream didn't pass the parent attempt (legacy in-flight tasks)
	// or when the postprocess stage span isn't found.
	var gSpan *Span
	if p.KnowledgeID != "" && p.Attempt > 0 {
		parent := s.tracker().LookupStage(ctx, p.KnowledgeID, p.Attempt, types.StagePostProcess)
		if parent != nil {
			gSpan = s.tracker().BeginSubSpan(ctx, parent,
				fmt.Sprintf("postprocess.graph.chunk[%d]", p.ChunkIndex),
				types.SpanKindSubSpan,
				types.JSONMap{
					"chunk_id":    p.ChunkID,
					"chunk_index": p.ChunkIndex,
					"model_id":    p.ModelID,
				})
		}
	}
	var handleErr error
	graphOut := types.JSONMap{}
	defer func() {
		// Decrement the parent's enrichment counter on terminal exit so a
		// completed (or terminally-failed) per-chunk extract releases its
		// slot in pending_subtasks_count. KnowledgeID is the new (post-#? )
		// payload field; legacy in-flight tasks without it are skipped.
		finalizeSubtaskDetached(ctx, s.knowledgeRepo, p.KnowledgeID,
			fmt.Sprintf("graph_chunk[%d]", p.ChunkIndex),
			handleErr, false, isFinalAsynqAttempt(ctx))
		if gSpan == nil {
			return
		}
		if handleErr != nil {
			s.tracker().FailSpan(ctx, gSpan, "GRAPH_EXTRACT_FAILED", handleErr.Error(), handleErr)
		} else {
			s.tracker().EndSpan(ctx, gSpan, graphOut)
		}
	}()

	// Short-circuit when the parent knowledge has been cancelled / deleted.
	// Each graph extract is per-chunk and runs one LLM call — the most
	// expensive enrichment fan-out in the pipeline. Skipping on cancel
	// is the whole point of the finalizing-state machinery above.
	if p.KnowledgeID != "" && s.knowledgeRepo != nil {
		if k, kerr := s.knowledgeRepo.GetKnowledgeByIDOnly(ctx, p.KnowledgeID); kerr == nil && k != nil {
			switch k.ParseStatus {
			case types.ParseStatusCancelled, types.ParseStatusDeleting:
				logger.Infof(ctx, "graph extract: knowledge %s aborted (%s), skipping chunk %s",
					p.KnowledgeID, k.ParseStatus, p.ChunkID)
				graphOut["skipped"] = "knowledge_" + k.ParseStatus
				return nil
			}
		}
	}

	chunk, err := s.chunkRepo.GetChunkByID(ctx, p.TenantID, p.ChunkID)
	if err != nil {
		logger.Errorf(ctx, "failed to get chunk: %v", err)
		handleErr = err
		return err
	}
	// Capture chunk content shape on output — lets traces answer "WHAT
	// did the LLM call see?" without joining back to the chunk store.
	// Preview is truncated to keep span rows reasonable.
	if gSpan != nil {
		graphOut["chunk_chars"] = len([]rune(chunk.Content))
		graphOut["chunk_preview"] = previewText(chunk.Content, 200)
	}
	kb, err := s.knowledgeBaseRepo.GetKnowledgeBaseByID(ctx, chunk.KnowledgeBaseID)
	if err != nil {
		logger.Errorf(ctx, "failed to get knowledge base: %v", err)
		handleErr = err
		return err
	}

	var processOverrides *types.KnowledgeProcessOverrides
	knowledgeID := p.KnowledgeID
	if knowledgeID == "" {
		knowledgeID = chunk.KnowledgeID
	}
	if knowledgeID != "" && s.knowledgeRepo != nil {
		if k, kerr := s.knowledgeRepo.GetKnowledgeByIDOnly(ctx, knowledgeID); kerr == nil && k != nil {
			processOverrides, _ = k.ProcessOverrides()
		}
	}
	extractCfg := ResolveProcessConfig(kb, processOverrides).ExtractConfig
	if !extractCfg.Enabled {
		logger.Warnf(ctx, "extract config not enabled")
		graphOut["skipped"] = "extract_disabled"
		return nil
	}

	chatModel, err := s.modelService.GetChatModel(ctx, p.ModelID)
	if err != nil {
		logger.Errorf(ctx, "failed to get chat model: %v", err)
		handleErr = err
		return err
	}

	template := &types.PromptTemplateStructured{
		Description: types.AppendCustomPromptInstructions(
			s.template.Description, extractCfg.CustomInstructions, "graph_extraction"),
		Tags: extractCfg.Tags,
		Examples: []types.GraphData{
			{
				Text:     extractCfg.Text,
				Node:     extractCfg.Nodes,
				Relation: extractCfg.Relations,
			},
		},
	}
	extractor := chatpipeline.NewExtractor(chatModel, template)
	graph, err := extractor.Extract(ctx, chunk.Content)
	if err != nil {
		handleErr = err
		return err
	}

	chunk, err = s.chunkRepo.GetChunkByID(ctx, p.TenantID, p.ChunkID)
	if err != nil {
		logger.Warnf(ctx, "graph ignore chunk %s: %v", p.ChunkID, err)
		graphOut["skipped"] = "chunk_disappeared"
		return nil
	}

	for _, node := range graph.Node {
		node.Chunks = []string{chunk.ID}
	}
	if err = s.graphEngine.AddGraph(ctx,
		types.NameSpace{KnowledgeBase: chunk.KnowledgeBaseID, Knowledge: chunk.KnowledgeID},
		[]*types.GraphData{graph},
	); err != nil {
		logger.Errorf(ctx, "failed to add graph: %v", err)
		handleErr = err
		return err
	}
	graphOut["nodes_added"] = len(graph.Node)
	graphOut["relations_added"] = len(graph.Relation)
	// Capture a couple of sample nodes/relations so the trace viewer can
	// answer "what did the LLM actually extract?" without round-tripping
	// to the graph store. Cap to two each — anything more bloats span
	// rows and the full graph is queryable elsewhere.
	if len(graph.Node) > 0 {
		samples := graph.Node
		if len(samples) > 2 {
			samples = samples[:2]
		}
		names := make([]string, 0, len(samples))
		for _, n := range samples {
			names = append(names, n.Name)
		}
		graphOut["sample_nodes"] = names
	}
	if len(graph.Relation) > 0 {
		samples := graph.Relation
		if len(samples) > 2 {
			samples = samples[:2]
		}
		out := make([]string, 0, len(samples))
		for _, r := range samples {
			out = append(out, fmt.Sprintf("%s --[%s]--> %s", r.Node1, r.Type, r.Node2))
		}
		graphOut["sample_relations"] = out
	}
	return nil
}
