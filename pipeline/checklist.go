package pipeline

import (
	"fmt"
	"strings"
	"time"
)

// GenerateChecklist runs Stage 5: formats obligations into a markdown checklist.
func GenerateChecklist(classification *ClassifyResult, mapper *MapperResult, scorer *ScorerResult, description string) string {
	var b strings.Builder

	// Header
	b.WriteString("## EU AI Act Compliance Checklist\n\n")
	fmt.Fprintf(&b, "**System:** %s\n", description)
	fmt.Fprintf(&b, "**Risk Tier:** %s\n", formatRiskTier(mapper.RiskTier))
	if len(mapper.ClassificationBasis) > 0 {
		fmt.Fprintf(&b, "**Basis:** %s\n", strings.Join(mapper.ClassificationBasis, ", "))
	}
	fmt.Fprintf(&b, "**Confidence:** %.0f%%\n", scorer.OverallConfidence)
	fmt.Fprintf(&b, "**Generated:** %s\n\n", time.Now().Format("2006-01-02"))

	// Exception info
	if mapper.ExceptionApplicable {
		fmt.Fprintf(&b, "> **Note:** Art. 6(3) exception may apply: %s\n\n", mapper.ExceptionReasoning)
	}

	// Mandatory obligations
	mandatory := filterByPriority(mapper.Obligations, "MANDATORY")
	if len(mandatory) > 0 {
		b.WriteString("### Mandatory Obligations\n\n")
		for _, ob := range mandatory {
			fmt.Fprintf(&b, "#### %s: %s\n", ob.Article, ob.Title)
			fmt.Fprintf(&b, "- [ ] %s\n", ob.Summary)
			if ob.Deadline != "" {
				fmt.Fprintf(&b, "  - Deadline: %s\n", ob.Deadline)
			}
			b.WriteString("\n")
		}
	}

	// Recommended obligations
	recommended := filterByPriority(mapper.Obligations, "RECOMMENDED")
	if len(recommended) > 0 {
		b.WriteString("### Recommended Actions\n\n")
		for _, ob := range recommended {
			fmt.Fprintf(&b, "#### %s: %s\n", ob.Article, ob.Title)
			fmt.Fprintf(&b, "- [ ] %s\n\n", ob.Summary)
		}
	}

	// Ambiguity warnings
	if len(scorer.AmbiguityFlags) > 0 {
		b.WriteString("### Areas Requiring Legal Review\n\n")
		for _, flag := range scorer.AmbiguityFlags {
			fmt.Fprintf(&b, "- %s\n", flag)
		}
		b.WriteString("\n")
	}

	// Key articles
	articles := collectArticles(mapper.Obligations)
	if len(articles) > 0 {
		fmt.Fprintf(&b, "### Key Articles\n\n%s\n", strings.Join(articles, " · "))
	}

	return b.String()
}

func formatRiskTier(tier string) string {
	switch tier {
	case "HIGH_RISK":
		return "HIGH-RISK"
	case "LIMITED_RISK":
		return "LIMITED-RISK"
	case "MINIMAL_RISK":
		return "MINIMAL-RISK"
	case "PROHIBITED":
		return "PROHIBITED"
	default:
		return tier
	}
}

func filterByPriority(obligations []Obligation, priority string) []Obligation {
	var filtered []Obligation
	for _, o := range obligations {
		if o.Priority == priority {
			filtered = append(filtered, o)
		}
	}
	return filtered
}

func collectArticles(obligations []Obligation) []string {
	seen := map[string]bool{}
	var articles []string
	for _, o := range obligations {
		if !seen[o.Article] {
			seen[o.Article] = true
			articles = append(articles, o.Article)
		}
	}
	return articles
}
