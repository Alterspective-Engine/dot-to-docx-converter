package metadata

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/analyzer"
)

type FieldType string

const (
	FieldTypeMergeField  FieldType = "MERGEFIELD"
	FieldTypeFormula     FieldType = "FORMULA"
	FieldTypeIF          FieldType = "IF"
	FieldTypeDocProperty FieldType = "DOCPROPERTY"
	FieldTypeDocVariable FieldType = "DOCVARIABLE"
	FieldTypeASK         FieldType = "ASK"
	FieldTypeFILLIN      FieldType = "FILLIN"
	FieldTypeREF         FieldType = "REF"
)

type Field struct {
	Original         string    `json:"original"`
	Type             FieldType `json:"type"`
	Position         int       `json:"position"`
	Context          string    `json:"context,omitempty"`
	SuggestedMapping string    `json:"suggestedMapping"`
	ExtractedValue   string    `json:"extractedValue,omitempty"`
}

type TemplateMetadata struct {
	SourceFile      string               `json:"sourceFile"`
	ConversionDate  time.Time            `json:"conversionDate"`
	Version         string               `json:"version"`
	Format          string               `json:"format"`
	FileSize        int64                `json:"fileSize"`
	Complexity      ComplexityInfo       `json:"complexity"`
	Fields          FieldCollection      `json:"fields"`
	ConversionNotes []string             `json:"conversionNotes"`
	Statistics      ConversionStatistics `json:"statistics"`
	Mappings        map[string]string    `json:"mappings,omitempty"`
}

type ComplexityInfo struct {
	Score int    `json:"score"`
	Level string `json:"level"`
}

type FieldCollection struct {
	MergeFields   []Field `json:"mergeFields"`
	Formulas      []Field `json:"formulas"`
	IFStatements  []Field `json:"ifStatements"`
	DocProperties []Field `json:"docProperties"`
	DocVariables  []Field `json:"docVariables"`
	OtherFields   []Field `json:"otherFields"`
}

type ConversionStatistics struct {
	TotalFields      int `json:"totalFields"`
	MappableFields   int `json:"mappableFields"`
	ComplexFields    int `json:"complexFields"`
	ExtractedText    int `json:"extractedTextLength"`
	TableCount       int `json:"tableCount"`
	ImageCount       int `json:"imageCount"`
	EstimatedSuccess int `json:"estimatedSuccessRate"`
}

type MetadataExtractor struct {
	patterns map[string]*regexp.Regexp
}

func NewMetadataExtractor() *MetadataExtractor {
	return &MetadataExtractor{
		patterns: map[string]*regexp.Regexp{
			"mergefield":  regexp.MustCompile(`(?i)«([^»]+)»|{{\s*([^}]+)\s*}}|\bMERGEFIELD\s+(\w+)`),
			"formula":     regexp.MustCompile(`(?i)=\s*(?:SUM|AVG|COUNT|MAX|MIN|IF)\s*\([^)]+\)`),
			"ifstatement": regexp.MustCompile(`(?i)(?:IF\s+[^{]+\s*{|{{\s*#if\s+[^}]+}})`),
			"docproperty": regexp.MustCompile(`(?i)\bDOCPROPERTY\s+"?([^"\s]+)"?`),
			"docvariable": regexp.MustCompile(`(?i)\bDOCVARIABLE\s+"?([^"\s]+)"?`),
			"ask":         regexp.MustCompile(`(?i)\bASK\s+(\w+)\s+"([^"]+)"`),
			"fillin":      regexp.MustCompile(`(?i)\bFILLIN\s+"([^"]+)"`),
			"ref":         regexp.MustCompile(`(?i)\bREF\s+(\w+)`),
		},
	}
}

func (m *MetadataExtractor) ExtractMetadata(content []byte, filename string, docInfo *analyzer.DocumentInfo, report *analyzer.ComplexityReport) (*TemplateMetadata, error) {
	metadata := &TemplateMetadata{
		SourceFile:     filename,
		ConversionDate: time.Now(),
		Version:        "2.1.0",
		Format:         m.detectFormat(content),
		FileSize:       int64(len(content)),
		Complexity: ComplexityInfo{
			Score: report.Score,
			Level: report.Level,
		},
		Fields:          FieldCollection{},
		ConversionNotes: []string{},
		Mappings:        make(map[string]string),
	}

	text := docInfo.Text
	if text == "" && len(content) > 0 {
		text = string(content)
	}

	m.extractMergeFields(text, &metadata.Fields)
	m.extractFormulas(text, &metadata.Fields)
	m.extractIFStatements(text, &metadata.Fields)
	m.extractDocProperties(text, &metadata.Fields)
	m.extractOtherFields(text, &metadata.Fields)

	m.generateSuggestedMappings(metadata)
	m.calculateStatistics(metadata, docInfo)
	m.generateConversionNotes(metadata, report)

	return metadata, nil
}

func (m *MetadataExtractor) detectFormat(content []byte) string {
	if len(content) > 4 {
		if content[0] == 0x50 && content[1] == 0x4B {
			return "Modern (ZIP-based)"
		}
		if content[0] == 0xD0 && content[1] == 0xCF {
			return "Legacy (OLE-based)"
		}
	}
	return "Unknown"
}

func (m *MetadataExtractor) extractMergeFields(text string, fields *FieldCollection) {
	matches := m.patterns["mergefield"].FindAllStringSubmatchIndex(text, -1)
	seen := make(map[string]bool)

	for _, match := range matches {
		if match[0] < 0 || match[1] > len(text) {
			continue
		}

		original := text[match[0]:match[1]]
		if seen[original] {
			continue
		}
		seen[original] = true

		fieldName := m.extractFieldName(original)
		context := m.extractContext(text, match[0], 50)

		field := Field{
			Original:         original,
			Type:             FieldTypeMergeField,
			Position:         match[0],
			Context:          context,
			ExtractedValue:   fieldName,
			SuggestedMapping: m.suggestMapping(fieldName, FieldTypeMergeField),
		}
		fields.MergeFields = append(fields.MergeFields, field)
	}
}

func (m *MetadataExtractor) extractFormulas(text string, fields *FieldCollection) {
	matches := m.patterns["formula"].FindAllStringSubmatchIndex(text, -1)

	for _, match := range matches {
		if match[0] < 0 || match[1] > len(text) {
			continue
		}

		original := text[match[0]:match[1]]
		context := m.extractContext(text, match[0], 50)

		field := Field{
			Original:         original,
			Type:             FieldTypeFormula,
			Position:         match[0],
			Context:          context,
			SuggestedMapping: m.suggestFormulaMapping(original),
		}
		fields.Formulas = append(fields.Formulas, field)
	}
}

func (m *MetadataExtractor) extractIFStatements(text string, fields *FieldCollection) {
	matches := m.patterns["ifstatement"].FindAllStringSubmatchIndex(text, -1)

	for _, match := range matches {
		if match[0] < 0 || match[1] > len(text) {
			continue
		}

		original := text[match[0]:match[1]]
		context := m.extractContext(text, match[0], 100)

		field := Field{
			Original:         original,
			Type:             FieldTypeIF,
			Position:         match[0],
			Context:          context,
			SuggestedMapping: m.suggestIFMapping(original),
		}
		fields.IFStatements = append(fields.IFStatements, field)
	}
}

func (m *MetadataExtractor) extractDocProperties(text string, fields *FieldCollection) {
	matches := m.patterns["docproperty"].FindAllStringSubmatchIndex(text, -1)

	for _, match := range matches {
		if match[0] < 0 || match[1] > len(text) {
			continue
		}

		original := text[match[0]:match[1]]
		propName := m.extractPropertyName(original)

		field := Field{
			Original:         original,
			Type:             FieldTypeDocProperty,
			Position:         match[0],
			ExtractedValue:   propName,
			SuggestedMapping: fmt.Sprintf("{{document.%s}}", strings.ToLower(propName)),
		}
		fields.DocProperties = append(fields.DocProperties, field)
	}
}

func (m *MetadataExtractor) extractOtherFields(text string, fields *FieldCollection) {
	for fieldType, pattern := range m.patterns {
		if fieldType == "mergefield" || fieldType == "formula" || fieldType == "ifstatement" || fieldType == "docproperty" {
			continue
		}

		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if match[0] < 0 || match[1] > len(text) {
				continue
			}

			original := text[match[0]:match[1]]
			var fType FieldType
			switch fieldType {
			case "docvariable":
				fType = FieldTypeDocVariable
			case "ask":
				fType = FieldTypeASK
			case "fillin":
				fType = FieldTypeFILLIN
			case "ref":
				fType = FieldTypeREF
			default:
				continue
			}

			field := Field{
				Original:         original,
				Type:             fType,
				Position:         match[0],
				SuggestedMapping: m.suggestGenericMapping(original, fType),
			}
			fields.OtherFields = append(fields.OtherFields, field)
		}
	}
}

func (m *MetadataExtractor) extractContext(text string, position int, contextSize int) string {
	start := position - contextSize
	if start < 0 {
		start = 0
	}
	end := position + contextSize
	if end > len(text) {
		end = len(text)
	}

	context := text[start:end]
	context = strings.ReplaceAll(context, "\n", " ")
	context = strings.ReplaceAll(context, "\r", " ")
	context = strings.TrimSpace(context)

	if len(context) > 100 {
		context = context[:97] + "..."
	}

	return context
}

func (m *MetadataExtractor) extractFieldName(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "«»{}")
	text = strings.TrimSpace(text)

	if strings.Contains(text, "MERGEFIELD") {
		parts := strings.Fields(text)
		if len(parts) > 1 {
			return parts[1]
		}
	}

	return text
}

func (m *MetadataExtractor) extractPropertyName(text string) string {
	parts := strings.Fields(text)
	if len(parts) > 1 {
		return strings.Trim(parts[1], `"`)
	}
	return ""
}

func (m *MetadataExtractor) suggestMapping(fieldName string, fieldType FieldType) string {
	fieldName = strings.ToLower(fieldName)
	fieldName = strings.ReplaceAll(fieldName, " ", "_")
	fieldName = strings.ReplaceAll(fieldName, "-", "_")

	commonMappings := map[string]string{
		"clientname":  "{{client.name}}",
		"client_name": "{{client.name}}",
		"firstname":   "{{client.firstName}}",
		"first_name":  "{{client.firstName}}",
		"lastname":    "{{client.lastName}}",
		"last_name":   "{{client.lastName}}",
		"date":        "{{document.date}}",
		"today":       "{{system.today}}",
		"company":     "{{company.name}}",
		"address":     "{{client.address}}",
		"email":       "{{client.email}}",
		"phone":       "{{client.phone}}",
		"matter":      "{{matter.reference}}",
		"matterref":   "{{matter.reference}}",
		"matter_ref":  "{{matter.reference}}",
	}

	if mapping, exists := commonMappings[fieldName]; exists {
		return mapping
	}

	if strings.Contains(fieldName, "client") {
		return fmt.Sprintf("{{client.%s}}", strings.ReplaceAll(fieldName, "client", ""))
	}
	if strings.Contains(fieldName, "matter") {
		return fmt.Sprintf("{{matter.%s}}", strings.ReplaceAll(fieldName, "matter", ""))
	}

	return fmt.Sprintf("{{fields.%s}}", fieldName)
}

func (m *MetadataExtractor) suggestFormulaMapping(formula string) string {
	formula = strings.ToUpper(formula)
	if strings.Contains(formula, "SUM") {
		return "{{calculation.sum}}"
	}
	if strings.Contains(formula, "AVG") {
		return "{{calculation.average}}"
	}
	if strings.Contains(formula, "COUNT") {
		return "{{calculation.count}}"
	}
	return "{{calculation.custom}}"
}

func (m *MetadataExtractor) suggestIFMapping(ifStatement string) string {
	if strings.Contains(strings.ToLower(ifStatement), "client") {
		if strings.Contains(strings.ToLower(ifStatement), "type") {
			return "{{#if client.type}}"
		}
		return "{{#if client}}"
	}
	return "{{#if condition}}"
}

func (m *MetadataExtractor) suggestGenericMapping(text string, fieldType FieldType) string {
	switch fieldType {
	case FieldTypeDocVariable:
		return "{{variables.custom}}"
	case FieldTypeASK:
		return "{{input.prompt}}"
	case FieldTypeFILLIN:
		return "{{input.fillin}}"
	case FieldTypeREF:
		return "{{reference.field}}"
	default:
		return "{{field.unknown}}"
	}
}

func (m *MetadataExtractor) generateSuggestedMappings(metadata *TemplateMetadata) {
	for _, field := range metadata.Fields.MergeFields {
		if field.ExtractedValue != "" {
			metadata.Mappings[field.ExtractedValue] = field.SuggestedMapping
		}
	}
}

func (m *MetadataExtractor) calculateStatistics(metadata *TemplateMetadata, docInfo *analyzer.DocumentInfo) {
	totalFields := len(metadata.Fields.MergeFields) +
		len(metadata.Fields.Formulas) +
		len(metadata.Fields.IFStatements) +
		len(metadata.Fields.DocProperties) +
		len(metadata.Fields.DocVariables) +
		len(metadata.Fields.OtherFields)

	mappableFields := len(metadata.Fields.MergeFields) + len(metadata.Fields.DocProperties)
	complexFields := len(metadata.Fields.Formulas) + len(metadata.Fields.IFStatements)

	successRate := 100
	if metadata.Complexity.Level == "high" {
		successRate = 70
	} else if metadata.Complexity.Level == "critical" {
		successRate = 50
	} else if complexFields > 5 {
		successRate = 85
	}

	metadata.Statistics = ConversionStatistics{
		TotalFields:      totalFields,
		MappableFields:   mappableFields,
		ComplexFields:    complexFields,
		ExtractedText:    len(docInfo.Text),
		TableCount:       docInfo.TableCount,
		ImageCount:       0, // ImageCount field not yet available in DocumentInfo
		EstimatedSuccess: successRate,
	}
}

func (m *MetadataExtractor) generateConversionNotes(metadata *TemplateMetadata, report *analyzer.ComplexityReport) {
	if len(metadata.Fields.Formulas) > 0 {
		metadata.ConversionNotes = append(metadata.ConversionNotes,
			fmt.Sprintf("%d formulas detected - manual validation required after conversion",
				len(metadata.Fields.Formulas)))
	}

	if len(metadata.Fields.IFStatements) > 3 {
		metadata.ConversionNotes = append(metadata.ConversionNotes,
			"Complex conditional logic detected - review branching after conversion")
	}

	if metadata.Format == "Legacy (OLE-based)" {
		metadata.ConversionNotes = append(metadata.ConversionNotes,
			"Legacy format - extra validation recommended")
	}

	if metadata.Statistics.ExtractedText < 100 {
		metadata.ConversionNotes = append(metadata.ConversionNotes,
			"Limited text extracted - document may contain embedded objects")
	}

	for _, issue := range report.Issues {
		if issue.Severity == "high" || issue.Severity == "critical" {
			metadata.ConversionNotes = append(metadata.ConversionNotes, issue.Description)
		}
	}
}

func (m *MetadataExtractor) SaveMetadata(metadata *TemplateMetadata, outputPath string) error {
	_, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return nil
}

func (m *MetadataExtractor) GenerateFieldMarkers(text string, metadata *TemplateMetadata) string {
	result := text

	for _, field := range metadata.Fields.MergeFields {
		marker := fmt.Sprintf("[[FIELD:%s:MERGEFIELD]]", field.ExtractedValue)
		result = strings.ReplaceAll(result, field.Original, marker)
	}

	for _, field := range metadata.Fields.Formulas {
		marker := fmt.Sprintf("[[FORMULA:%s:CALCULATE]]", field.Original)
		result = strings.ReplaceAll(result, field.Original, marker)
	}

	for _, field := range metadata.Fields.IFStatements {
		marker := fmt.Sprintf("[[IF:%s:START]]", field.Original)
		result = strings.ReplaceAll(result, field.Original, marker)
	}

	return result
}
