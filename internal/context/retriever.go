package context

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ContextRetriever provides intelligent context retrieval for AI assistance
type ContextRetriever struct {
	semanticIndex   *SemanticIndex
	keywordIndex    *KeywordIndex
	dependencyGraph *DependencyGraph
	config          *RetrieverConfig
}

// RetrieverConfig contains configuration for context retrieval
type RetrieverConfig struct {
	MaxTokens        int           `json:"max_tokens"`
	SemanticWeight   float64       `json:"semantic_weight"`
	KeywordWeight    float64       `json:"keyword_weight"`
	DependencyWeight float64       `json:"dependency_weight"`
	Timeout          time.Duration `json:"timeout"`
	CacheEnabled     bool          `json:"cache_enabled"`
	CacheTTL         time.Duration `json:"cache_ttl"`
}

// Context represents retrieved context for AI assistance
type Context struct {
	Semantic     []ContextItem `json:"semantic"`
	Keyword      []ContextItem `json:"keyword"`
	Dependencies []ContextItem `json:"dependencies"`
	Score        float64       `json:"score"`
	Tokens       int           `json:"tokens"`
	RetrievedAt  time.Time     `json:"retrieved_at"`
}

// ContextItem represents a single piece of context
type ContextItem struct {
	Content     string  `json:"content"`
	Type        string  `json:"type"` // function, struct, interface, etc.
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Score       float64 `json:"score"`
	Relevance   float64 `json:"relevance"`
	Tokens      int     `json:"tokens"`
	Description string  `json:"description"`
}

// SemanticIndex provides semantic search capabilities
type SemanticIndex struct {
	embeddings map[string][]float64
	items      map[string]ContextItem
	dimensions int
}

// KeywordIndex provides keyword-based search
type KeywordIndex struct {
	index map[string][]string // keyword -> item IDs
	items map[string]ContextItem
}

// DependencyGraph provides dependency-based context retrieval
type DependencyGraph struct {
	nodes map[string]*DependencyNode
	edges map[string][]string // node -> dependencies
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	File         string                 `json:"file"`
	Line         int                    `json:"line"`
	Content      string                 `json:"content"`
	Dependencies []string               `json:"dependencies"`
	Dependents   []string               `json:"dependents"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// NewContextRetriever creates a new context retriever
func NewContextRetriever(config *RetrieverConfig) *ContextRetriever {
	if config == nil {
		config = getDefaultConfig()
	}

	return &ContextRetriever{
		semanticIndex:   NewSemanticIndex(),
		keywordIndex:    NewKeywordIndex(),
		dependencyGraph: NewDependencyGraph(),
		config:          config,
	}
}

// RetrieveRelevantContext finds context for AI assistance
func (cr *ContextRetriever) RetrieveRelevantContext(ctx context.Context, query string) (*Context, error) {
	_ = time.Now()

	// Multi-source retrieval: semantic + keyword + dependency
	semanticResults := cr.semanticIndex.Search(query, cr.config.MaxTokens/3)
	keywordResults := cr.keywordIndex.Search(query, cr.config.MaxTokens/3)
	dependencyResults := cr.dependencyGraph.GetRelated(query, cr.config.MaxTokens/3)

	// Calculate relevance scores
	cr.calculateRelevanceScores(semanticResults, query)
	cr.calculateRelevanceScores(keywordResults, query)
	cr.calculateRelevanceScores(dependencyResults, query)

	// Rank and filter results
	semanticResults = cr.rankAndFilter(semanticResults, cr.config.MaxTokens/3)
	keywordResults = cr.rankAndFilter(keywordResults, cr.config.MaxTokens/3)
	dependencyResults = cr.rankAndFilter(dependencyResults, cr.config.MaxTokens/3)

	// Calculate total tokens
	totalTokens := cr.calculateTotalTokens(semanticResults, keywordResults, dependencyResults)

	// Calculate overall score
	score := cr.calculateOverallScore(semanticResults, keywordResults, dependencyResults)

	return &Context{
		Semantic:     semanticResults,
		Keyword:      keywordResults,
		Dependencies: dependencyResults,
		Score:        score,
		Tokens:       totalTokens,
		RetrievedAt:  time.Now(),
	}, nil
}

// IndexCode indexes code for context retrieval
func (cr *ContextRetriever) IndexCode(filePath string, code string) error {
	// Parse code and extract context items
	items := cr.extractContextItems(filePath, code)

	// Index in semantic index
	for _, item := range items {
		cr.semanticIndex.Add(item)
	}

	// Index in keyword index
	for _, item := range items {
		cr.keywordIndex.Add(item)
	}

	// Add to dependency graph
	for _, item := range items {
		cr.dependencyGraph.AddNode(item)
	}

	return nil
}

// Helper methods

func (cr *ContextRetriever) calculateRelevanceScores(items []ContextItem, query string) {
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	for i := range items {
		item := &items[i]
		contentLower := strings.ToLower(item.Content)

		// Calculate keyword relevance
		keywordScore := 0.0
		for _, word := range queryWords {
			if strings.Contains(contentLower, word) {
				keywordScore += 1.0
			}
		}
		keywordScore /= float64(len(queryWords))

		// Calculate semantic relevance (simplified)
		semanticScore := cr.calculateSemanticRelevance(item.Content, query)

		// Combine scores
		item.Relevance = (keywordScore * 0.3) + (semanticScore * 0.7)
		item.Score = item.Relevance
	}
}

func (cr *ContextRetriever) calculateSemanticRelevance(content, query string) float64 {
	// Simplified semantic relevance calculation
	// In practice, you'd use embeddings or more sophisticated NLP

	contentWords := strings.Fields(strings.ToLower(content))
	queryWords := strings.Fields(strings.ToLower(query))

	// Calculate word overlap
	overlap := 0
	for _, qWord := range queryWords {
		for _, cWord := range contentWords {
			if qWord == cWord {
				overlap++
				break
			}
		}
	}

	if len(queryWords) == 0 {
		return 0.0
	}

	return float64(overlap) / float64(len(queryWords))
}

func (cr *ContextRetriever) rankAndFilter(items []ContextItem, maxTokens int) []ContextItem {
	// Sort by relevance score
	sort.Slice(items, func(i, j int) bool {
		return items[i].Relevance > items[j].Relevance
	})

	// Filter by token limit
	var filtered []ContextItem
	tokenCount := 0

	for _, item := range items {
		if tokenCount+item.Tokens <= maxTokens {
			filtered = append(filtered, item)
			tokenCount += item.Tokens
		}
	}

	return filtered
}

func (cr *ContextRetriever) calculateTotalTokens(semantic, keyword, dependency []ContextItem) int {
	total := 0

	for _, item := range semantic {
		total += item.Tokens
	}
	for _, item := range keyword {
		total += item.Tokens
	}
	for _, item := range dependency {
		total += item.Tokens
	}

	return total
}

func (cr *ContextRetriever) calculateOverallScore(semantic, keyword, dependency []ContextItem) float64 {
	if len(semantic) == 0 && len(keyword) == 0 && len(dependency) == 0 {
		return 0.0
	}

	var totalScore float64
	var totalWeight float64

	// Weighted average of scores
	if len(semantic) > 0 {
		semanticScore := cr.averageScore(semantic)
		totalScore += semanticScore * cr.config.SemanticWeight
		totalWeight += cr.config.SemanticWeight
	}

	if len(keyword) > 0 {
		keywordScore := cr.averageScore(keyword)
		totalScore += keywordScore * cr.config.KeywordWeight
		totalWeight += cr.config.KeywordWeight
	}

	if len(dependency) > 0 {
		dependencyScore := cr.averageScore(dependency)
		totalScore += dependencyScore * cr.config.DependencyWeight
		totalWeight += cr.config.DependencyWeight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return totalScore / totalWeight
}

func (cr *ContextRetriever) averageScore(items []ContextItem) float64 {
	if len(items) == 0 {
		return 0.0
	}

	total := 0.0
	for _, item := range items {
		total += item.Score
	}

	return total / float64(len(items))
}

func (cr *ContextRetriever) extractContextItems(filePath, code string) []ContextItem {
	// Simplified implementation - in practice, you'd use AST parsing
	var items []ContextItem

	lines := strings.Split(code, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Extract functions
		if strings.HasPrefix(line, "func ") {
			item := ContextItem{
				Type:        "function",
				File:        filePath,
				Line:        i + 1,
				Content:     line,
				Description: "Function definition",
				Tokens:      cr.estimateTokens(line),
			}
			items = append(items, item)
		}

		// Extract structs
		if strings.HasPrefix(line, "type ") && strings.Contains(line, "struct") {
			item := ContextItem{
				Type:        "struct",
				File:        filePath,
				Line:        i + 1,
				Content:     line,
				Description: "Struct definition",
				Tokens:      cr.estimateTokens(line),
			}
			items = append(items, item)
		}

		// Extract interfaces
		if strings.HasPrefix(line, "type ") && strings.Contains(line, "interface") {
			item := ContextItem{
				Type:        "interface",
				File:        filePath,
				Line:        i + 1,
				Content:     line,
				Description: "Interface definition",
				Tokens:      cr.estimateTokens(line),
			}
			items = append(items, item)
		}
	}

	return items
}

func (cr *ContextRetriever) estimateTokens(text string) int {
	// Simplified token estimation (roughly 4 characters per token)
	return len(text) / 4
}

// SemanticIndex implementation

func NewSemanticIndex() *SemanticIndex {
	return &SemanticIndex{
		embeddings: make(map[string][]float64),
		items:      make(map[string]ContextItem),
		dimensions: 384, // Default embedding dimension
	}
}

func (si *SemanticIndex) Add(item ContextItem) {
	// Generate embedding for the item
	embedding := si.generateEmbedding(item.Content)

	// Store item and embedding
	itemID := fmt.Sprintf("%s:%d", item.File, item.Line)
	si.embeddings[itemID] = embedding
	si.items[itemID] = item
}

func (si *SemanticIndex) Search(query string, maxTokens int) []ContextItem {
	// Generate query embedding
	queryEmbedding := si.generateEmbedding(query)

	// Calculate similarities
	var results []ContextItem
	for itemID, embedding := range si.embeddings {
		similarity := si.calculateSimilarity(queryEmbedding, embedding)
		if similarity > 0.3 { // Threshold for relevance
			item := si.items[itemID]
			item.Score = similarity
			results = append(results, item)
		}
	}

	// Sort by similarity
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit by tokens
	var filtered []ContextItem
	tokenCount := 0
	for _, item := range results {
		if tokenCount+item.Tokens <= maxTokens {
			filtered = append(filtered, item)
			tokenCount += item.Tokens
		}
	}

	return filtered
}

func (si *SemanticIndex) generateEmbedding(text string) []float64 {
	// Simplified embedding generation
	// In practice, you'd use a proper embedding model

	words := strings.Fields(strings.ToLower(text))
	embedding := make([]float64, si.dimensions)

	for i, word := range words {
		if i >= si.dimensions {
			break
		}
		// Simple hash-based embedding
		hash := 0
		for _, char := range word {
			hash += int(char)
		}
		embedding[i] = float64(hash%100) / 100.0
	}

	return embedding
}

func (si *SemanticIndex) calculateSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	// Calculate cosine similarity
	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (normA * normB)
}

// KeywordIndex implementation

func NewKeywordIndex() *KeywordIndex {
	return &KeywordIndex{
		index: make(map[string][]string),
		items: make(map[string]ContextItem),
	}
}

func (ki *KeywordIndex) Add(item ContextItem) {
	itemID := fmt.Sprintf("%s:%d", item.File, item.Line)
	ki.items[itemID] = item

	// Extract keywords from content
	keywords := ki.extractKeywords(item.Content)

	for _, keyword := range keywords {
		ki.index[keyword] = append(ki.index[keyword], itemID)
	}
}

func (ki *KeywordIndex) Search(query string, maxTokens int) []ContextItem {
	queryWords := ki.extractKeywords(query)
	itemScores := make(map[string]float64)

	// Score items based on keyword matches
	for _, word := range queryWords {
		if itemIDs, exists := ki.index[word]; exists {
			for _, itemID := range itemIDs {
				itemScores[itemID] += 1.0
			}
		}
	}

	// Convert to results
	var results []ContextItem
	for itemID, score := range itemScores {
		item := ki.items[itemID]
		item.Score = score
		results = append(results, item)
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit by tokens
	var filtered []ContextItem
	tokenCount := 0
	for _, item := range results {
		if tokenCount+item.Tokens <= maxTokens {
			filtered = append(filtered, item)
			tokenCount += item.Tokens
		}
	}

	return filtered
}

func (ki *KeywordIndex) extractKeywords(text string) []string {
	// Simple keyword extraction
	words := strings.Fields(strings.ToLower(text))
	var keywords []string

	for _, word := range words {
		// Filter out common words and short words
		if len(word) > 2 && !isCommonWord(word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "and": true, "or": true, "but": true, "in": true,
		"on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true,
		"would": true, "could": true, "should": true, "may": true, "might": true,
	}

	return commonWords[word]
}

// DependencyGraph implementation

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*DependencyNode),
		edges: make(map[string][]string),
	}
}

func (dg *DependencyGraph) AddNode(item ContextItem) {
	nodeID := fmt.Sprintf("%s:%d", item.File, item.Line)

	node := &DependencyNode{
		ID:       nodeID,
		Type:     item.Type,
		File:     item.File,
		Line:     item.Line,
		Content:  item.Content,
		Metadata: make(map[string]interface{}),
	}

	dg.nodes[nodeID] = node
}

func (dg *DependencyGraph) AddEdge(from, to string) {
	dg.edges[from] = append(dg.edges[from], to)

	// Update node dependencies
	if fromNode, exists := dg.nodes[from]; exists {
		fromNode.Dependencies = append(fromNode.Dependencies, to)
	}

	if toNode, exists := dg.nodes[to]; exists {
		toNode.Dependents = append(toNode.Dependents, from)
	}
}

func (dg *DependencyGraph) GetRelated(query string, maxTokens int) []ContextItem {
	// Find nodes that match the query
	var matchingNodes []*DependencyNode
	for _, node := range dg.nodes {
		if strings.Contains(strings.ToLower(node.Content), strings.ToLower(query)) {
			matchingNodes = append(matchingNodes, node)
		}
	}

	// Get related nodes (dependencies and dependents)
	var relatedNodes []*DependencyNode
	for _, node := range matchingNodes {
		// Add dependencies
		for _, depID := range node.Dependencies {
			if depNode, exists := dg.nodes[depID]; exists {
				relatedNodes = append(relatedNodes, depNode)
			}
		}

		// Add dependents
		for _, depID := range node.Dependents {
			if depNode, exists := dg.nodes[depID]; exists {
				relatedNodes = append(relatedNodes, depNode)
			}
		}
	}

	// Convert to context items
	var results []ContextItem
	for _, node := range relatedNodes {
		item := ContextItem{
			Type:        node.Type,
			File:        node.File,
			Line:        node.Line,
			Content:     node.Content,
			Description: "Related dependency",
			Tokens:      len(node.Content) / 4, // Rough token estimate
		}
		results = append(results, item)
	}

	// Limit by tokens
	var filtered []ContextItem
	tokenCount := 0
	for _, item := range results {
		if tokenCount+item.Tokens <= maxTokens {
			filtered = append(filtered, item)
			tokenCount += item.Tokens
		}
	}

	return filtered
}

// Configuration helpers

func getDefaultConfig() *RetrieverConfig {
	return &RetrieverConfig{
		MaxTokens:        4000,
		SemanticWeight:   0.5,
		KeywordWeight:    0.3,
		DependencyWeight: 0.2,
		Timeout:          5 * time.Second,
		CacheEnabled:     true,
		CacheTTL:         1 * time.Hour,
	}
}
