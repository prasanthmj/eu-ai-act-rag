package pipeline

// Score runs Stage 4: computes a heuristic confidence score (no LLM call).
func Score(classification *ClassifyResult, chunks []RetrievedChunk, mapper *MapperResult) *ScorerResult {
	result := &ScorerResult{}

	// Factor 1: Average annex similarity score (Hop 1 results)
	var annexScoreSum float64
	var annexCount int
	for _, c := range chunks {
		if c.Hop == 1 && c.Score > 0 {
			annexScoreSum += float64(c.Score)
			annexCount++
		}
	}
	avgAnnexScore := 0.0
	if annexCount > 0 {
		avgAnnexScore = annexScoreSum / float64(annexCount)
	}

	// Factor 2: Strong Annex III match (any hop-1 score > 0.75)
	hasStrongMatch := false
	for _, c := range chunks {
		if c.Hop == 1 && c.Score > 0.75 {
			hasStrongMatch = true
			break
		}
	}

	// Factor 3: Corroboration — how many retrieved articles back up the obligations
	articleDocIDs := map[string]bool{}
	for _, c := range chunks {
		if c.Hop == 2 {
			articleDocIDs[c.DocID] = true
		}
	}
	citedArticles := 0
	for _, ob := range mapper.Obligations {
		// Normalize "Article 9" to "article_9"
		normalized := "article_" + extractNumber(ob.Article)
		if articleDocIDs[normalized] {
			citedArticles++
		}
	}
	corroboration := 0.0
	if len(mapper.Obligations) > 0 {
		corroboration = float64(citedArticles) / float64(len(mapper.Obligations))
	}

	// Combine factors into 0–100 score
	confidence := avgAnnexScore*40 + corroboration*40
	if hasStrongMatch {
		confidence += 20
	}
	if confidence > 100 {
		confidence = 100
	}
	result.OverallConfidence = confidence

	// Ambiguity flags
	if classification.ExceptionCandidate {
		result.AmbiguityFlags = append(result.AmbiguityFlags,
			"Article 6(3) exception may apply — legal review recommended")
	}
	if !hasStrongMatch && len(classification.RiskTiers) > 0 && classification.RiskTiers[0] == "HIGH_RISK" {
		result.AmbiguityFlags = append(result.AmbiguityFlags,
			"No strong Annex III match found — classification may need review")
	}

	return result
}

// extractNumber pulls the first number from a string like "Article 9".
func extractNumber(s string) string {
	var num []byte
	started := false
	for _, c := range []byte(s) {
		if c >= '0' && c <= '9' {
			num = append(num, c)
			started = true
		} else if started {
			break
		}
	}
	return string(num)
}
