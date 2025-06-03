package intelligence

import (
	"regexp"
	"strings"
)

// BasicConceptExtractor implements the ConceptExtractor interface
type BasicConceptExtractor struct {
	technicalTerms *regexp.Regexp
	codePatterns   *regexp.Regexp
	actionWords    *regexp.Regexp
	conceptWords   *regexp.Regexp
}

// NewBasicConceptExtractor creates a new basic concept extractor
func NewBasicConceptExtractor() *BasicConceptExtractor {
	return &BasicConceptExtractor{
		technicalTerms: regexp.MustCompile(`(?i)\b(api|database|server|client|framework|library|algorithm|function|class|method|variable|array|object|string|integer|boolean|json|xml|http|rest|graphql|sql|nosql|cache|redis|postgresql|mysql|mongodb|docker|kubernetes|git|github|aws|azure|gcp|typescript|javascript|python|go|java|rust|cpp|html|css|react|vue|angular|node|express|flask|django|rails|spring|laravel)\b`),
		codePatterns:   regexp.MustCompile(`(?i)\b(pattern|design|architecture|structure|component|module|package|import|export|dependency|interface|implementation|abstraction|inheritance|polymorphism|encapsulation|composition|aggregation)\b`),
		actionWords:    regexp.MustCompile(`(?i)\b(create|build|implement|develop|design|refactor|optimize|fix|debug|test|deploy|configure|setup|install|update|upgrade|migrate|scale|monitor|log|validate|authenticate|authorize|encrypt|decrypt|parse|serialize|deserialize|compress|decompress|backup|restore)\b`),
		conceptWords:   regexp.MustCompile(`(?i)\b(concept|idea|approach|strategy|methodology|principle|paradigm|philosophy|theory|practice|technique|mechanism|process|workflow|pipeline|lifecycle|pattern|model|schema|format|protocol|standard|specification|convention|guideline|best_practice|anti_pattern)\b`),
	}
}

// ExtractConcepts extracts concepts from text
func (bce *BasicConceptExtractor) ExtractConcepts(text string) ([]Concept, error) {
	// Pre-allocate with estimated capacity based on typical matches
	concepts := make([]Concept, 0, 50)

	// Extract technical terms
	technicalMatches := bce.technicalTerms.FindAllString(text, -1)
	for _, match := range technicalMatches {
		concept := Concept{
			Name:        match,
			Type:        "technical_term",
			Description: "Technical term: " + match,
			Confidence:  0.8,
			Context: map[string]any{
				"category": "technology",
				"source":   "technical_terms_regex",
			},
		}
		concepts = append(concepts, concept)
	}

	// Extract architectural patterns
	codeMatches := bce.codePatterns.FindAllString(text, -1)
	for _, match := range codeMatches {
		concept := Concept{
			Name:        match,
			Type:        "architectural_concept",
			Description: "Architectural concept: " + match,
			Confidence:  0.7,
			Context: map[string]any{
				"category": "architecture",
				"source":   "code_patterns_regex",
			},
		}
		concepts = append(concepts, concept)
	}

	// Extract action concepts
	actionMatches := bce.actionWords.FindAllString(text, -1)
	for _, match := range actionMatches {
		concept := Concept{
			Name:        match,
			Type:        "action",
			Description: "Action concept: " + match,
			Confidence:  0.6,
			Context: map[string]any{
				"category": "action",
				"source":   "action_words_regex",
			},
		}
		concepts = append(concepts, concept)
	}

	// Extract abstract concepts
	conceptMatches := bce.conceptWords.FindAllString(text, -1)
	for _, match := range conceptMatches {
		concept := Concept{
			Name:        match,
			Type:        "abstract_concept",
			Description: "Abstract concept: " + match,
			Confidence:  0.5,
			Context: map[string]any{
				"category": "abstract",
				"source":   "concept_words_regex",
			},
		}
		concepts = append(concepts, concept)
	}

	// Extract key phrases (noun phrases)
	phrases := bce.ExtractKeyPhrases(text)
	for _, phrase := range phrases {
		if len(phrase) > 5 && !bce.isCommonPhrase(phrase) {
			concept := Concept{
				Name:        phrase,
				Type:        "key_phrase",
				Description: "Key phrase: " + phrase,
				Confidence:  0.4,
				Context: map[string]any{
					"category": "phrase",
					"source":   "key_phrase_extraction",
				},
			}
			concepts = append(concepts, concept)
		}
	}

	// Remove duplicates and sort by confidence
	uniqueConcepts := bce.removeDuplicateConcepts(concepts)

	return uniqueConcepts, nil
}

// IdentifyTechnicalTerms identifies technical terms in text
func (bce *BasicConceptExtractor) IdentifyTechnicalTerms(text string) []string {
	matches := bce.technicalTerms.FindAllString(text, -1)
	return bce.unique(matches)
}

// ExtractKeyPhrases extracts key phrases from text
func (bce *BasicConceptExtractor) ExtractKeyPhrases(text string) []string {
	// Simple noun phrase extraction using patterns
	nounPhraseRegex := regexp.MustCompile(`(?i)\b([A-Z][a-z]+ (?:[A-Z][a-z]+ )*[A-Z][a-z]+)\b`)
	camelCaseRegex := regexp.MustCompile(`\b[a-z]+[A-Z][a-zA-Z]*\b`)

	var phrases []string

	// Extract capitalized noun phrases
	nounMatches := nounPhraseRegex.FindAllString(text, -1)
	phrases = append(phrases, nounMatches...)

	// Extract camelCase identifiers
	camelMatches := camelCaseRegex.FindAllString(text, -1)
	phrases = append(phrases, camelMatches...)

	return bce.unique(phrases)
}

// BasicEntityExtractor implements the EntityExtractor interface
type BasicEntityExtractor struct {
	fileRegex     *regexp.Regexp
	functionRegex *regexp.Regexp
	variableRegex *regexp.Regexp
	commandRegex  *regexp.Regexp
}

// NewBasicEntityExtractor creates a new basic entity extractor
func NewBasicEntityExtractor() *BasicEntityExtractor {
	return &BasicEntityExtractor{
		fileRegex:     regexp.MustCompile(`\b[\w\-_]+\.(go|js|ts|py|java|cpp|h|hpp|c|rs|rb|php|cs|kt|swift|scala|clj|elm|hs|ml|fs|dart|r|m|pl|sh|bat|ps1|sql|json|yaml|yml|xml|html|css|scss|less|md|txt|csv|log|config|conf|ini|env|dockerfile|makefile)\b`),
		functionRegex: regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\s*\(`),
		variableRegex: regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`),
		commandRegex:  regexp.MustCompile(`(?m)^\s*[$#>]\s*(\w+(?:\s+[\w\-\.]+)*)`),
	}
}

// ExtractFiles extracts file references from text
func (bee *BasicEntityExtractor) ExtractFiles(text string) []string {
	matches := bee.fileRegex.FindAllString(text, -1)
	return bee.unique(matches)
}

// ExtractFunctions extracts function references from text
func (bee *BasicEntityExtractor) ExtractFunctions(text string) []string {
	matches := bee.functionRegex.FindAllString(text, -1)
	var functions []string

	for _, match := range matches {
		// Remove the opening parenthesis
		funcName := strings.TrimSuffix(strings.TrimSpace(match), "(")
		if len(funcName) > 2 && !bee.isCommonWord(funcName) {
			functions = append(functions, funcName)
		}
	}

	return bee.unique(functions)
}

// ExtractVariables extracts variable references from text
func (bee *BasicEntityExtractor) ExtractVariables(text string) []string {
	// This is a simplified implementation
	// In practice, you'd want more sophisticated parsing
	codeBlockRegex := regexp.MustCompile("```[\\s\\S]*?```")
	codeBlocks := codeBlockRegex.FindAllString(text, -1)

	var variables []string
	for _, block := range codeBlocks {
		matches := bee.variableRegex.FindAllString(block, -1)
		for _, match := range matches {
			if len(match) > 2 && !bee.isCommonWord(match) && !bee.isKeyword(match) {
				variables = append(variables, match)
			}
		}
	}

	return bee.unique(variables)
}

// ExtractCommands extracts command references from text
func (bee *BasicEntityExtractor) ExtractCommands(text string) []string {
	matches := bee.commandRegex.FindAllStringSubmatch(text, -1)
	var commands []string

	for _, match := range matches {
		if len(match) > 1 {
			commands = append(commands, match[1])
		}
	}

	return bee.unique(commands)
}

// Helper methods for BasicConceptExtractor

func (bce *BasicConceptExtractor) isCommonPhrase(phrase string) bool {
	commonPhrases := map[string]bool{
		"the system":      true,
		"the process":     true,
		"the method":      true,
		"the function":    true,
		"the component":   true,
		"the service":     true,
		"the application": true,
		"the user":        true,
		"the client":      true,
		"the server":      true,
	}

	return commonPhrases[strings.ToLower(phrase)]
}

func (bce *BasicConceptExtractor) removeDuplicateConcepts(concepts []Concept) []Concept {
	seen := make(map[string]bool)
	var result []Concept

	for _, concept := range concepts {
		key := strings.ToLower(concept.Name)
		if !seen[key] {
			seen[key] = true
			result = append(result, concept)
		}
	}

	return result
}

func (bce *BasicConceptExtractor) unique(items []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// Helper methods for BasicEntityExtractor

func (bee *BasicEntityExtractor) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "had": true,
		"her": true, "was": true, "one": true, "our": true, "out": true,
		"day": true, "get": true, "has": true, "him": true, "how": true,
		"man": true, "new": true, "now": true, "old": true, "see": true,
		"two": true, "way": true, "who": true, "boy": true, "did": true,
		"its": true, "let": true, "put": true, "say": true, "she": true,
		"too": true, "use": true, "may": true, "end": true, "why": true,
		"try": true, "ask": true, "men": true, "run": true, "own": true,
	}

	return commonWords[strings.ToLower(word)]
}

func (bee *BasicEntityExtractor) isKeyword(word string) bool {
	keywords := map[string]bool{
		"if": true, "else": true, "for": true, "while": true, "do": true,
		"switch": true, "case": true, "break": true, "continue": true,
		"return": true, "function": true, "var": true, "let": true, "const": true,
		"class": true, "interface": true, "struct": true, "enum": true,
		"public": true, "private": true, "protected": true, "static": true,
		"final": true, "abstract": true, "override": true, "virtual": true,
		"import": true, "export": true, "from": true, "as": true,
		"try": true, "catch": true, "finally": true, "throw": true,
		"new": true, "delete": true, "this": true, "super": true,
		"true": true, "false": true, "null": true, "undefined": true,
		"package": true, "namespace": true, "using": true, "include": true,
	}

	return keywords[strings.ToLower(word)]
}

func (bee *BasicEntityExtractor) unique(items []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
