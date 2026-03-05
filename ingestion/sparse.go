package ingestion

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"strings"
	"unicode"
)

// SparseVector holds indices and values for a Qdrant sparse vector.
type SparseVector struct {
	Indices []uint32  `json:"indices"`
	Values  []float32 `json:"values"`
}

// SparseEncoder builds BM25-style sparse vectors from text.
// It must be trained on a corpus first to compute IDF weights.
type SparseEncoder struct {
	docFreq  map[uint32]int // term hash → number of docs containing it
	numDocs  int
	vocabMax uint32 // hash space size
}

// NewSparseEncoder creates a SparseEncoder.
func NewSparseEncoder() *SparseEncoder {
	return &SparseEncoder{
		docFreq:  make(map[uint32]int),
		vocabMax: 1 << 20, // ~1M hash buckets
	}
}

// Fit computes document frequencies from a corpus of texts.
func (e *SparseEncoder) Fit(texts []string) {
	e.numDocs = len(texts)
	for _, text := range texts {
		seen := map[uint32]bool{}
		for _, token := range tokenize(text) {
			h := hashToken(token, e.vocabMax)
			if !seen[h] {
				e.docFreq[h]++
				seen[h] = true
			}
		}
	}
}

// Encode produces a sparse vector for a single text using BM25 TF-IDF weights.
func (e *SparseEncoder) Encode(text string) SparseVector {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return SparseVector{}
	}

	// Compute term frequencies
	tf := map[uint32]int{}
	for _, token := range tokens {
		h := hashToken(token, e.vocabMax)
		tf[h]++
	}

	// BM25 parameters
	const k1 = 1.2
	const b = 0.75
	avgDL := 200.0 // approximate average document length for legal text
	dl := float64(len(tokens))

	indices := make([]uint32, 0, len(tf))
	values := make([]float32, 0, len(tf))

	for h, count := range tf {
		df := e.docFreq[h]
		if df == 0 {
			df = 1 // unseen terms get minimal IDF
		}

		// IDF: log((N - df + 0.5) / (df + 0.5) + 1)
		idf := math.Log(float64(e.numDocs-df)+0.5)/float64(df)+0.5 + 1.0

		// BM25 TF: (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * dl/avgdl))
		tfNorm := (float64(count) * (k1 + 1)) / (float64(count) + k1*(1-b+b*dl/avgDL))

		weight := idf * tfNorm
		if weight > 0 {
			indices = append(indices, h)
			values = append(values, float32(weight))
		}
	}

	return SparseVector{Indices: indices, Values: values}
}

// tokenize splits text into lowercase tokens, removing punctuation and stop words.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	tokens := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) < 2 || stopWords[w] {
			continue
		}
		tokens = append(tokens, w)
	}
	return tokens
}

// hashToken maps a token string to a uint32 index within the hash space.
func hashToken(token string, max uint32) uint32 {
	h := fnv.New32a()
	h.Write([]byte(token))
	return h.Sum32() % max
}

// sparseEncoderData is the serializable form of SparseEncoder.
type sparseEncoderData struct {
	DocFreq  map[uint32]int `json:"doc_freq"`
	NumDocs  int            `json:"num_docs"`
	VocabMax uint32         `json:"vocab_max"`
}

// Save writes the fitted encoder to a JSON file.
func (e *SparseEncoder) Save(path string) error {
	data := sparseEncoderData{
		DocFreq:  e.docFreq,
		NumDocs:  e.numDocs,
		VocabMax: e.vocabMax,
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(data)
}

// LoadSparseEncoder loads a fitted encoder from a JSON file.
func LoadSparseEncoder(path string) (*SparseEncoder, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var data sparseEncoderData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return &SparseEncoder{
		docFreq:  data.DocFreq,
		numDocs:  data.NumDocs,
		vocabMax: data.VocabMax,
	}, nil
}

// stopWords are common English words excluded from sparse vectors.
var stopWords = map[string]bool{
	"the": true, "be": true, "to": true, "of": true, "and": true,
	"in": true, "that": true, "have": true, "it": true, "for": true,
	"not": true, "on": true, "with": true, "he": true, "as": true,
	"you": true, "do": true, "at": true, "this": true, "but": true,
	"his": true, "by": true, "from": true, "they": true, "we": true,
	"her": true, "she": true, "or": true, "an": true, "will": true,
	"my": true, "one": true, "all": true, "would": true, "there": true,
	"their": true, "what": true, "so": true, "if": true, "about": true,
	"which": true, "when": true, "who": true, "no": true, "is": true,
	"are": true, "was": true, "were": true, "been": true, "has": true,
	"had": true, "did": true, "does": true, "a": true, "its": true,
	"than": true, "into": true, "can": true, "may": true, "shall": true,
}
