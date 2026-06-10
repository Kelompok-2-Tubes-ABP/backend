package services

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
)

// KnowledgeEntry represents a Q&A pair in the knowledge base
type KnowledgeEntry struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// KnowledgeBase represents the entire knowledge base
type KnowledgeBase struct {
	Entries []KnowledgeEntry `json:"knowledge_base"`
}

// RAGService handles retrieval augmented generation
type RAGService struct {
	kb   KnowledgeBase
	topK int
}

// NewRAGService creates a new RAG service
func NewRAGService(kbPath string, topK int) (*RAGService, error) {
	if topK <= 0 {
		topK = 3
	}

	rag := &RAGService{
		topK: topK,
	}

	if err := rag.loadKnowledgeBase(kbPath); err != nil {
		return nil, fmt.Errorf("failed to load knowledge base: %w", err)
	}

	return rag, nil
}

// loadKnowledgeBase loads the knowledge base from JSON file
func (r *RAGService) loadKnowledgeBase(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var kb KnowledgeBase
	if err := json.Unmarshal(data, &kb); err != nil {
		return err
	}

	r.kb = kb
	return nil
}

// SearchResult represents a search result with score
type SearchResult struct {
	Entry KnowledgeEntry
	Score float64
}

// Search finds the most relevant Q&A entries for a query
func (r *RAGService) Search(query string) []SearchResult {
	cleanQuery := r.cleanText(query)
	queryTokens := r.tokenize(cleanQuery)

	if len(queryTokens) == 0 {
		return nil
	}

	results := make([]SearchResult, 0)

	for _, entry := range r.kb.Entries {
		entryQuestion := r.cleanText(entry.Question)
		entryContent := r.cleanText(entry.Question + " " + entry.Answer)
		entryTokens := r.tokenize(entryContent)
		questionTokens := r.tokenize(entryQuestion)

		// Multiple scoring methods
		keywordScore := r.keywordMatchScore(queryTokens, entryTokens)
		questionScore := r.jaccardSimilarity(queryTokens, questionTokens)
		exactMatchScore := r.exactMatchScore(cleanQuery, entryQuestion)
		fuzzyScore := r.fuzzyMatchScore(queryTokens, entryTokens)

		// Combined score with weights
		totalScore := (keywordScore * 0.35) + (questionScore * 0.30) + (exactMatchScore * 0.25) + (fuzzyScore * 0.10)

		if totalScore > 0.05 {
			results = append(results, SearchResult{
				Entry: entry,
				Score: totalScore,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > r.topK {
		results = results[:r.topK]
	}

	return results
}

// BuildContext builds a context string from search results
func (r *RAGService) BuildContext(query string) string {
	results := r.Search(query)

	if len(results) == 0 {
		return ""
	}

	context := "📚 **Referensi Pengetahuan Keuangan:**\n\n"

	for i, result := range results {
		context += fmt.Sprintf("**%d.** %s\n%s\n\n",
			i+1, result.Entry.Question, result.Entry.Answer)
	}

	return context
}

// Helper functions

func (r *RAGService) cleanText(text string) string {
	text = strings.ToLower(text)
	text = regexp.MustCompile(`[^a-z0-9\s]`).ReplaceAllString(text, " ")
	text = strings.Join(strings.Fields(text), " ")
	return text
}

func (r *RAGService) tokenize(text string) []string {
	return strings.Fields(text)
}

func (r *RAGService) keywordMatchScore(queryTokens, contentTokens []string) float64 {
	if len(queryTokens) == 0 {
		return 0
	}

	matches := 0
	for _, qt := range queryTokens {
		for _, ct := range contentTokens {
			if len(qt) >= 3 && len(ct) >= 3 {
				// Check substring match
				if strings.Contains(ct, qt) || strings.Contains(qt, ct) {
					matches++
					break
				}
				// Check Levenshtein-like partial match
				if r.partialMatch(qt, ct) > 0.7 {
					matches++
					break
				}
			} else if qt == ct {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(queryTokens))
}

func (r *RAGService) jaccardSimilarity(tokens1, tokens2 []string) float64 {
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0
	}

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, t := range tokens1 {
		set1[t] = true
	}
	for _, t := range tokens2 {
		set2[t] = true
	}

	intersection := 0
	for t := range set1 {
		if set2[t] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func (r *RAGService) exactMatchScore(query, target string) float64 {
	if query == "" || target == "" {
		return 0
	}

	// Check if query words appear in target
	queryWords := r.tokenize(query)
	targetWords := r.tokenize(target)

	if len(queryWords) == 0 {
		return 0
	}

	matches := 0
	for _, qw := range queryWords {
		for _, tw := range targetWords {
			if strings.Contains(tw, qw) || strings.Contains(qw, tw) {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(queryWords))
}

func (r *RAGService) fuzzyMatchScore(queryTokens, contentTokens []string) float64 {
	if len(queryTokens) == 0 {
		return 0
	}

	matches := 0
	for _, qt := range queryTokens {
		for _, ct := range contentTokens {
			if r.partialMatch(qt, ct) > 0.6 {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(queryTokens))
}

// partialMatch calculates similarity between two strings using simple algorithm
func (r *RAGService) partialMatch(s1, s2 string) float64 {
	if len(s1) < 2 || len(s2) < 2 {
		return 0
	}

	// Simple prefix/suffix matching
	common := 0
	minLen := int(math.Min(float64(len(s1)), float64(len(s2))))

	for i := 0; i < minLen; i++ {
		if s1[i] == s2[i] {
			common++
		} else {
			break
		}
	}

	prefixScore := float64(common) / float64(minLen)

	// Suffix matching
	common = 0
	for i := 0; i < minLen; i++ {
		l1 := len(s1) - 1 - i
		l2 := len(s2) - 1 - i
		if l1 >= 0 && l2 >= 0 && s1[l1] == s2[l2] {
			common++
		} else {
			break
		}
	}

	suffixScore := float64(common) / float64(minLen)

	return (prefixScore + suffixScore) / 2
}
