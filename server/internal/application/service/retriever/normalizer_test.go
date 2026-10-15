package retriever

import (
	"context"
	"math"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestEngineAwareNormalizer_KeywordPassthrough(t *testing.T) {
	t.Parallel()
	n := EngineAwareNormalizer{}
	cases := []float64{-12.5, 0, 0.7, 1, 27.3, math.Inf(1)}
	for _, s := range cases {
		// Any keyword retriever, any engine — must be identity.
		got := n.Normalize(context.Background(), s, types.KeywordsRetrieverType,
			types.QdrantRetrieverEngineType)
		if got != s {
			// math.IsNaN special handling for the Inf case.
			if math.IsInf(s, 1) && math.IsInf(got, 1) {
				continue
			}
			t.Fatalf("keyword passthrough expected %v, got %v", s, got)
		}
	}
}

// Qdrant reports cosine similarity like every other engine (IP or COSINE over
// L2-normalized embeddings; L2 is converted in the driver), so the value
// passes through clamp01 unchanged apart from the [0, 1] envelope guard.
func TestEngineAwareNormalizer_QdrantIsCosine(t *testing.T) {
	t.Parallel()
	n := EngineAwareNormalizer{}
	cases := []struct {
		score float64
		want  float64
	}{
		{-0.5, 0},
		{0, 0},
		{0.5, 0.5},
		{1.0, 1.0},
		{1.5, 1.0},
	}
	for _, tc := range cases {
		got := n.Normalize(context.Background(), tc.score,
			types.VectorRetrieverType, types.QdrantRetrieverEngineType)
		if math.Abs(got-tc.want) > 1e-9 {
			t.Fatalf("qdrant score=%v: want %v, got %v", tc.score, tc.want, got)
		}
	}
}

func TestEngineAwareNormalizer_UnitInterval(t *testing.T) {
	t.Parallel()
	n := EngineAwareNormalizer{}
	// Both remaining engines report cosine similarity that is theoretically
	// [-1, 1] but, for the IR-normalized embeddings WeKnora targets (BGE /
	// OpenAI / Cohere / sentence-transformers), the observed range stays in
	// [0, 1]; the silent floor at clamp01(<0) → 0 is the defense for
	// embedding models that violate that assumption.
	cases := []struct {
		score float64
		want  float64
	}{
		{-0.1, 0},
		{0, 0},
		{0.25, 0.25},
		{0.999, 0.999},
		{1, 1},
		{1.5, 1},
	}
	for _, engine := range []types.RetrieverEngineType{
		types.PostgresRetrieverEngineType,
		types.QdrantRetrieverEngineType,
	} {
		for _, tc := range cases {
			got := n.Normalize(context.Background(), tc.score,
				types.VectorRetrieverType, engine)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("unit[%s] score=%v: want %v, got %v",
					engine, tc.score, tc.want, got)
			}
		}
	}
}

// TestEngineAwareNormalizer_QdrantCosinePassthrough pins the passthrough
// policy: a cosine of 0.5 must stay 0.5 (not be rescaled to (0.5 + 1) / 2 =
// 0.75), so multi-store merges rank hits from both engines on the same
// scale.
func TestEngineAwareNormalizer_QdrantCosinePassthrough(t *testing.T) {
	t.Parallel()
	n := EngineAwareNormalizer{}
	cases := []struct {
		name  string
		score float64
		want  float64
	}{
		{"cos=0 orthogonal IR-clamped to 0", 0, 0},
		{"cos=0.25 IR-typical low passes through", 0.25, 0.25},
		{"cos=0.5 IR-typical mid passes through (not 0.75)", 0.5, 0.5},
		{"cos=0.75 IR-typical high passes through", 0.75, 0.75},
		{"cos=1 identical passes through as 1", 1, 1},
		// A driver should never hand the normalizer a negative cosine, but
		// the clamp guards the downstream sort comparator either way.
		{"negative defensive clamps to 0", -0.5, 0},
		// Engine drift past 1 clamps to 1.
		{"drift past 1 clamps to 1", 1.0001, 1},
		// Float-edge cases handled by clamp01.
		{"+Inf clamps to 1", math.Inf(1), 1},
		{"-Inf clamps to 0", math.Inf(-1), 0},
		{"NaN clamps to 0", math.NaN(), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := n.Normalize(context.Background(), tc.score,
				types.VectorRetrieverType, types.QdrantRetrieverEngineType)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("qdrant score=%v: want %v, got %v",
					tc.score, tc.want, got)
			}
		})
	}
}

func TestEngineAwareNormalizer_Unknown_ClampsAndDoesNotPanic(t *testing.T) {
	t.Parallel()
	n := EngineAwareNormalizer{}
	got := n.Normalize(context.Background(), 0.42,
		types.VectorRetrieverType, types.RetrieverEngineType("nosuch"))
	if got != 0.42 {
		t.Fatalf("unknown engine passthrough-clamp: want 0.42, got %v", got)
	}
	got = n.Normalize(context.Background(), 5.0,
		types.VectorRetrieverType, types.RetrieverEngineType("nosuch"))
	if got != 1.0 {
		t.Fatalf("unknown engine clamp on >1: want 1, got %v", got)
	}
}

func TestEngineAwareNormalizer_NilCtx_DoesNotPanic(t *testing.T) {
	// nil ctx must not panic. Normalize is IO-free by contract — it never
	// reads from ctx and never logs. The unknown-engine warning is
	// emitted by the caller (retrieveFromStores), where ctx is always
	// live.
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Normalize panicked on nil ctx: %v", r)
		}
	}()
	_ = EngineAwareNormalizer{}.Normalize(
		nil, 0.5, types.VectorRetrieverType,
		types.RetrieverEngineType("nosuch"),
	)
}

func TestEngineAwareNormalizer_NaNAndInf(t *testing.T) {
	t.Parallel()
	n := EngineAwareNormalizer{}
	// NaN → 0 (so SortFunc does not get a comparator-poisoning value).
	got := n.Normalize(context.Background(), math.NaN(),
		types.VectorRetrieverType, types.PostgresRetrieverEngineType)
	if got != 0 {
		t.Fatalf("NaN clamp: want 0, got %v", got)
	}
	// +Inf → 1.
	got = n.Normalize(context.Background(), math.Inf(1),
		types.VectorRetrieverType, types.PostgresRetrieverEngineType)
	if got != 1 {
		t.Fatalf("+Inf clamp: want 1, got %v", got)
	}
	// -Inf → 0.
	got = n.Normalize(context.Background(), math.Inf(-1),
		types.VectorRetrieverType, types.PostgresRetrieverEngineType)
	if got != 0 {
		t.Fatalf("-Inf clamp: want 0, got %v", got)
	}
	// NaN on the other engine too.
	got = n.Normalize(context.Background(), math.NaN(),
		types.VectorRetrieverType, types.QdrantRetrieverEngineType)
	if got != 0 {
		t.Fatalf("NaN cosine: want 0, got %v", got)
	}
}

func TestClamp01(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want float64
	}{
		{-1, 0},
		{0, 0},
		{0.5, 0.5},
		{1, 1},
		{2, 1},
		{math.NaN(), 0},
		{math.Inf(1), 1},
		{math.Inf(-1), 0},
	}
	for _, tc := range cases {
		got := clamp01(tc.in)
		if math.Abs(got-tc.want) > 1e-9 {
			// math.IsNaN comparison would have already failed via the
			// equality above; this branch covers the rest.
			t.Fatalf("clamp01(%v): want %v, got %v", tc.in, tc.want, got)
		}
	}
}

// TestEngineAwareNormalizer_InterfaceSatisfied is a compile-time check that
// catches accidental breakage of the ScoreNormalizer interface via the
// package-scope var assertion in normalizer.go. The runtime body is a
// no-op; failure would surface at build time.
func TestEngineAwareNormalizer_InterfaceSatisfied(t *testing.T) {
	var _ ScoreNormalizer = EngineAwareNormalizer{}
}
