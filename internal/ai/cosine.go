package ai

import (
	"math"
	"sort"

	"github.com/digitalghost404/inkandbone/internal/db"
)

func decodeFloat32Slice(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		bits := uint32(b[i*4]) |
			uint32(b[i*4+1])<<8 |
			uint32(b[i*4+2])<<16 |
			uint32(b[i*4+3])<<24
		out[i] = math.Float32frombits(bits)
	}
	return out
}

// CosineSimilarity returns the cosine similarity between two equal-length vectors.
func CosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// TopK returns the k chunks with the highest cosine similarity to queryEmb.
// Chunks without embeddings are skipped.
func TopK(chunks []db.RulebookChunk, queryEmb []float32, k int) []db.RulebookChunk {
	type scored struct {
		chunk db.RulebookChunk
		score float32
	}
	var candidates []scored
	for _, c := range chunks {
		if len(c.Embedding) == 0 {
			continue
		}
		emb := decodeFloat32Slice(c.Embedding)
		if len(emb) != len(queryEmb) {
			continue
		}
		candidates = append(candidates, scored{chunk: c, score: CosineSimilarity(emb, queryEmb)})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	result := make([]db.RulebookChunk, 0, k)
	for i := 0; i < k && i < len(candidates); i++ {
		result = append(result, candidates[i].chunk)
	}
	return result
}
