package cataloger

import (
	"strings"
	"unicode"
)

// FieldNormalizer handles field name standardization and similarity detection
type FieldNormalizer struct {
	commonAbbreviations map[string]string
	synonymGroups       map[string][]string
}

// NewFieldNormalizer creates a new field normalizer
func NewFieldNormalizer() *FieldNormalizer {
	return &FieldNormalizer{
		commonAbbreviations: initAbbreviations(),
		synonymGroups:       initSynonyms(),
	}
}

// Standardize converts field name to standard form
func (n *FieldNormalizer) Standardize(fieldName string) string {
	// Remove special characters and normalize case
	cleaned := n.cleanFieldName(fieldName)

	// Expand abbreviations
	expanded := n.expandAbbreviations(cleaned)

	// Convert to standard format (camelCase)
	standardized := n.toCamelCase(expanded)

	// Apply synonym mapping
	if primary := n.mapToSynonym(standardized); primary != "" {
		return primary
	}

	return standardized
}

// FindSimilarGroups groups similar field names together
func (n *FieldNormalizer) FindSimilarGroups(fields []string) map[string][]string {
	groups := make(map[string][]string)
	processed := make(map[string]bool)

	for i, field1 := range fields {
		if processed[field1] {
			continue
		}

		variants := []string{}
		for j, field2 := range fields {
			if i == j || processed[field2] {
				continue
			}

			similarity := n.CalculateSimilarity(field1, field2)
			if similarity > 0.8 { // 80% similarity threshold
				variants = append(variants, field2)
				processed[field2] = true
			}
		}

		if len(variants) > 0 {
			groups[field1] = variants
		}
		processed[field1] = true
	}

	return groups
}

// CalculateSimilarity returns similarity score between two field names (0-1)
func (n *FieldNormalizer) CalculateSimilarity(field1, field2 string) float64 {
	// Standardize both fields
	std1 := n.Standardize(field1)
	std2 := n.Standardize(field2)

	if std1 == std2 {
		return 1.0
	}

	// Calculate Levenshtein distance
	distance := n.levenshteinDistance(std1, std2)
	maxLen := max(len(std1), len(std2))

	if maxLen == 0 {
		return 0.0
	}

	// Convert distance to similarity score
	similarity := 1.0 - float64(distance)/float64(maxLen)

	// Boost score if fields share common tokens
	tokens1 := n.tokenize(std1)
	tokens2 := n.tokenize(std2)
	commonTokens := n.countCommonTokens(tokens1, tokens2)

	if commonTokens > 0 {
		tokenBoost := float64(commonTokens) / float64(max(len(tokens1), len(tokens2))) * 0.3
		similarity = min(1.0, similarity+tokenBoost)
	}

	return similarity
}

// cleanFieldName removes special characters and normalizes spaces
func (n *FieldNormalizer) cleanFieldName(name string) string {
	var result strings.Builder

	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteRune(' ')
		}
	}

	// Normalize multiple spaces
	cleaned := strings.Join(strings.Fields(result.String()), " ")
	return strings.TrimSpace(cleaned)
}

// expandAbbreviations expands common abbreviations
func (n *FieldNormalizer) expandAbbreviations(text string) string {
	words := strings.Fields(strings.ToLower(text))
	expanded := make([]string, 0, len(words))

	for _, word := range words {
		if expansion, exists := n.commonAbbreviations[word]; exists {
			expanded = append(expanded, expansion)
		} else {
			expanded = append(expanded, word)
		}
	}

	return strings.Join(expanded, " ")
}

// toCamelCase converts text to camelCase
func (n *FieldNormalizer) toCamelCase(text string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	// First word lowercase
	result := strings.ToLower(words[0])

	// Remaining words title case
	for i := 1; i < len(words); i++ {
		if len(words[i]) > 0 {
			result += strings.ToUpper(string(words[i][0])) + strings.ToLower(words[i][1:])
		}
	}

	return result
}

// mapToSynonym maps field to primary synonym if exists
func (n *FieldNormalizer) mapToSynonym(field string) string {
	fieldLower := strings.ToLower(field)

	for primary, synonyms := range n.synonymGroups {
		if strings.ToLower(primary) == fieldLower {
			return primary // Already primary
		}
		for _, syn := range synonyms {
			if strings.ToLower(syn) == fieldLower {
				return primary
			}
		}
	}

	return ""
}

// tokenize splits field name into tokens
func (n *FieldNormalizer) tokenize(field string) []string {
	// Split on camelCase boundaries
	var tokens []string
	var current strings.Builder

	for i, r := range field {
		if i > 0 && unicode.IsUpper(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
		current.WriteRune(unicode.ToLower(r))
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// countCommonTokens counts tokens that appear in both lists
func (n *FieldNormalizer) countCommonTokens(tokens1, tokens2 []string) int {
	set1 := make(map[string]bool)
	for _, t := range tokens1 {
		set1[t] = true
	}

	count := 0
	for _, t := range tokens2 {
		if set1[t] {
			count++
		}
	}

	return count
}

// levenshteinDistance calculates edit distance between two strings
func (n *FieldNormalizer) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first column and row
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Calculate distances
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = minInt(
				matrix[i-1][j]+1, // deletion
				minInt(
					matrix[i][j-1]+1,      // insertion
					matrix[i-1][j-1]+cost, // substitution
				),
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// Helper functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// initAbbreviations creates common abbreviation mappings
func initAbbreviations() map[string]string {
	return map[string]string{
		"fname": "firstname",
		"lname": "lastname",
		"dob":   "dateofbirth",
		"addr":  "address",
		"st":    "street",
		"num":   "number",
		"no":    "number",
		"tel":   "telephone",
		"mob":   "mobile",
		"ref":   "reference",
		"amt":   "amount",
		"qty":   "quantity",
		"desc":  "description",
		"dept":  "department",
		"mgr":   "manager",
		"emp":   "employee",
		"cust":  "customer",
		"acct":  "account",
		"inv":   "invoice",
		"po":    "purchaseorder",
		"cc":    "creditcard",
		"ssn":   "socialsecuritynumber",
		"ein":   "employeridentificationnumber",
		"vin":   "vehicleidentificationnumber",
		"id":    "identifier",
		"max":   "maximum",
		"min":   "minimum",
		"avg":   "average",
		"temp":  "temporary",
		"perm":  "permanent",
		"org":   "organization",
		"div":   "division",
		"co":    "company",
		"corp":  "corporation",
		"ltd":   "limited",
		"llc":   "limitedliabilitycompany",
		"inc":   "incorporated",
	}
}

// initSynonyms creates synonym groups for field mapping
func initSynonyms() map[string][]string {
	return map[string][]string{
		"clientName": {
			"customerName",
			"clientFullName",
			"partyName",
			"applicantName",
			"claimantName",
			"plaintiffName",
			"defendantName",
		},
		"firstName": {
			"givenName",
			"forename",
			"christianName",
		},
		"lastName": {
			"surname",
			"familyName",
		},
		"dateOfBirth": {
			"birthDate",
			"dob",
			"birthday",
		},
		"address": {
			"streetAddress",
			"postalAddress",
			"mailingAddress",
			"residentialAddress",
		},
		"phoneNumber": {
			"telephone",
			"contactNumber",
			"phone",
			"tel",
		},
		"email": {
			"emailAddress",
			"electronicMail",
			"emailId",
		},
		"matterNumber": {
			"caseNumber",
			"fileNumber",
			"matterReference",
			"caseReference",
			"claimNumber",
		},
		"amount": {
			"total",
			"sum",
			"value",
			"price",
			"cost",
		},
		"date": {
			"currentDate",
			"todaysDate",
			"today",
			"now",
		},
		"jurisdiction": {
			"state",
			"territory",
			"region",
			"court",
		},
		"matterType": {
			"caseType",
			"claimType",
			"practiceArea",
		},
	}
}
