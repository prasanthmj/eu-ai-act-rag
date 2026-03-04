package pipeline

import "github.com/prasanthmj/eu-ai-act-rag/rag"

// ClassifyResult is the output of Stage 1 (System Classifier).
type ClassifyResult struct {
	Domain             string   `json:"domain"`
	RiskTiers          []string `json:"risk_tiers"`
	Reasoning          string   `json:"reasoning"`
	NeedsProfiling     bool     `json:"needs_profiling"`
	ExceptionCandidate bool     `json:"exception_candidate"`
}

// RetrievedChunk is a search result tagged with the retrieval hop number.
type RetrievedChunk struct {
	rag.SearchResult
	Hop int `json:"hop"`
}

// Obligation is a single compliance obligation.
type Obligation struct {
	Article  string `json:"article"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Priority string `json:"priority"` // "MANDATORY" or "RECOMMENDED"
	Deadline string `json:"deadline"`
}

// MapperResult is the output of Stage 3 (Obligation Mapper).
type MapperResult struct {
	RiskTier            string       `json:"risk_tier"`
	ClassificationBasis []string     `json:"classification_basis"`
	ExceptionApplicable bool         `json:"exception_applicable"`
	ExceptionReasoning  string       `json:"exception_reasoning"`
	Obligations         []Obligation `json:"obligations"`
}

// ScorerResult is the output of Stage 4 (Confidence Scorer).
type ScorerResult struct {
	OverallConfidence float64  `json:"overall_confidence"`
	AmbiguityFlags   []string `json:"ambiguity_flags"`
}

// PipelineResult is the complete output of all 5 stages.
type PipelineResult struct {
	Classification ClassifyResult   `json:"classification"`
	Chunks         []RetrievedChunk `json:"retrieved_chunks"`
	Obligations    MapperResult     `json:"obligations"`
	Confidence     ScorerResult     `json:"confidence"`
	Checklist      string           `json:"checklist"`
}
