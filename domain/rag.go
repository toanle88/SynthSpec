package domain

import "math"

// VectorNode represents a single text chunk with its vector embedding and source metadata
type VectorNode struct {
	Text     string    `json:"text"`
	FilePath string    `json:"file_path"`
	Vector   []float32 `json:"vector"`
}

// CosineSimilarity computes the cosine similarity between two vector embeddings
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}
