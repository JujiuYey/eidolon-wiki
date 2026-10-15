package types

import (
	"os"
	"path/filepath"

	"github.com/go-ego/gse"
)

// jiebaSegmenter segments text with jieba-style Chinese word segmentation.
// Upstream WeKnora bound the CGO gojieba library here; slim replaces it with
// gse, a pure-Go port of jieba, so the binary builds without a C toolchain.
// Only CutForSearch is used across the codebase (query tokenization for
// full-text search and near-duplicate chunk comparison).
type jiebaSegmenter struct {
	seg *gse.Segmenter
}

// Jieba is the process-wide segmenter. When dictionary loading failed at
// startup (e.g. JIEBA_DICT_DIR pointing at a broken directory), segmentation
// degrades to returning the raw text so callers still receive one token
// instead of panicking on a nil segmenter.
var Jieba = newJieba()

func newJieba() *jiebaSegmenter {
	seg, err := loadSegmenter()
	if err != nil {
		return &jiebaSegmenter{}
	}
	return &jiebaSegmenter{seg: seg}
}

func loadSegmenter() (*gse.Segmenter, error) {
	dictDir := os.Getenv("JIEBA_DICT_DIR")
	if dictDir == "" {
		// No custom dictionaries: gse ships an embedded Chinese dictionary.
		seg, err := gse.New()
		if err != nil {
			return nil, err
		}
		return &seg, nil
	}
	// Custom dictionaries use the same jieba-format files upstream gojieba
	// consumed, so existing deployments keep working unchanged.
	seg, err := gse.New(
		filepath.Join(dictDir, "jieba.dict.utf8"),
		filepath.Join(dictDir, "hmm_model.utf8"),
		filepath.Join(dictDir, "user.dict.utf8"),
		filepath.Join(dictDir, "idf.utf8"),
		filepath.Join(dictDir, "stop_words.utf8"),
	)
	if err != nil {
		return nil, err
	}
	return &seg, nil
}

// CutForSearch segments text in search-engine mode: coarser-grained than
// accurate mode, mixing full words with their sub-words to raise recall.
func (j *jiebaSegmenter) CutForSearch(text string, hmm bool) []string {
	if j == nil || j.seg == nil {
		return []string{text}
	}
	return j.seg.CutSearch(text, hmm)
}
