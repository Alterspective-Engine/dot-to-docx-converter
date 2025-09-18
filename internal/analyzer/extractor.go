// Package analyzer provides document format detection and text extraction
package analyzer

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// DocumentFormat represents the detected document format
type DocumentFormat int

const (
	FormatUnknown DocumentFormat = iota
	FormatPlainText
	FormatZipBased  // Modern Word 2007+ format (DOCX/DOTX)
	FormatOLEBased  // Legacy Word format (DOC/DOT)
	FormatRTF       // Rich Text Format
)

// DocumentExtractor handles document format detection and text extraction
type DocumentExtractor struct {
	// Patterns for extracting text from XML
	xmlTextPattern *regexp.Regexp
	rtfTextPattern *regexp.Regexp
}

// NewDocumentExtractor creates a new document extractor
func NewDocumentExtractor() *DocumentExtractor {
	return &DocumentExtractor{
		xmlTextPattern: regexp.MustCompile(`<w:t[^>]*>([^<]+)</w:t>`),
		rtfTextPattern: regexp.MustCompile(`\\par|\\tab|\\line`),
	}
}

// DetectFormat detects the document format based on magic numbers
func (e *DocumentExtractor) DetectFormat(content []byte) DocumentFormat {
	if len(content) < 8 {
		return FormatUnknown
	}

	// Check for ZIP-based format (PK header)
	if bytes.HasPrefix(content, []byte{0x50, 0x4B, 0x03, 0x04}) {
		return FormatZipBased
	}

	// Check for OLE compound document format
	if bytes.HasPrefix(content, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
		return FormatOLEBased
	}

	// Check for RTF format
	if bytes.HasPrefix(content, []byte("{\\rtf")) {
		return FormatRTF
	}

	// Check if it looks like plain text
	if e.isProbablyText(content[:min(1000, len(content))]) {
		return FormatPlainText
	}

	return FormatUnknown
}

// ExtractText extracts readable text from the document
func (e *DocumentExtractor) ExtractText(content []byte) (string, error) {
	format := e.DetectFormat(content)

	switch format {
	case FormatZipBased:
		return e.extractFromZip(content)
	case FormatOLEBased:
		return e.extractFromOLE(content)
	case FormatRTF:
		return e.extractFromRTF(content)
	case FormatPlainText:
		return string(content), nil
	default:
		// Try to extract any readable text as fallback
		return e.extractReadableText(content), nil
	}
}

// extractFromZip extracts text from ZIP-based Word documents (DOCX/DOTX)
func (e *DocumentExtractor) extractFromZip(content []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("failed to read ZIP archive: %w", err)
	}

	var textBuilder strings.Builder
	documentsProcessed := 0

	// Primary document content files to extract
	targetFiles := []string{
		"word/document.xml",
		"word/header1.xml",
		"word/header2.xml",
		"word/header3.xml",
		"word/footer1.xml",
		"word/footer2.xml",
		"word/footer3.xml",
		"word/footnotes.xml",
		"word/endnotes.xml",
		"word/comments.xml",
	}

	for _, file := range reader.File {
		// Check if this is one of our target files
		isTarget := false
		for _, target := range targetFiles {
			if file.Name == target {
				isTarget = true
				break
			}
		}

		if !isTarget {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			continue
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		// Extract text from XML content
		text := e.extractTextFromXML(string(data))
		if len(text) > 0 {
			if textBuilder.Len() > 0 {
				textBuilder.WriteString("\n")
			}
			textBuilder.WriteString(text)
			documentsProcessed++
		}
	}

	if documentsProcessed == 0 {
		// If no standard Word documents found, try to extract from any XML files
		for _, file := range reader.File {
			if strings.HasSuffix(file.Name, ".xml") && strings.Contains(file.Name, "word/") {
				rc, err := file.Open()
				if err != nil {
					continue
				}

				data, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					continue
				}

				text := e.extractTextFromXML(string(data))
				if len(text) > 0 {
					if textBuilder.Len() > 0 {
						textBuilder.WriteString("\n")
					}
					textBuilder.WriteString(text)
				}
			}
		}
	}

	result := textBuilder.String()
	if len(result) == 0 {
		return "", fmt.Errorf("no text content found in ZIP-based document")
	}

	return result, nil
}

// extractTextFromXML extracts text from Word XML content
func (e *DocumentExtractor) extractTextFromXML(xmlContent string) string {
	var textBuilder strings.Builder

	// Extract text from w:t elements
	matches := e.xmlTextPattern.FindAllStringSubmatch(xmlContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			text := strings.TrimSpace(match[1])
			if len(text) > 0 {
				if textBuilder.Len() > 0 {
					textBuilder.WriteString(" ")
				}
				textBuilder.WriteString(text)
			}
		}
	}

	// Also look for field codes and merge fields
	fieldPatterns := []*regexp.Regexp{
		regexp.MustCompile(`<w:instrText[^>]*>([^<]+)</w:instrText>`),
		regexp.MustCompile(`MERGEFIELD\s+([^\s]+)`),
		regexp.MustCompile(`IF\s+([^}]+)`),
	}

	for _, pattern := range fieldPatterns {
		matches := pattern.FindAllStringSubmatch(xmlContent, -1)
		for _, match := range matches {
			if len(match) > 0 {
				if textBuilder.Len() > 0 {
					textBuilder.WriteString(" ")
				}
				// Add field codes with braces to maintain pattern recognition
				textBuilder.WriteString("{" + match[0] + "}")
			}
		}
	}

	return textBuilder.String()
}

// extractFromOLE extracts text from OLE compound documents
func (e *DocumentExtractor) extractFromOLE(content []byte) (string, error) {
	// OLE compound document structure is complex
	// We'll extract readable strings and look for Word-specific patterns

	var textBuilder strings.Builder

	// Skip OLE header (512 bytes)
	if len(content) > 512 {
		// Look for Word document streams
		textContent := e.extractReadableText(content[512:])

		// Try to find Word-specific text patterns
		// Word documents often have readable text interspersed with binary data
		textBuilder.WriteString(textContent)
	}

	// Extract any field codes that might be visible
	fieldPatterns := []string{
		"MERGEFIELD",
		"IF ",
		"DOCPROPERTY",
		"DOCVARIABLE",
		"«", "»", // Merge field delimiters
	}

	contentStr := string(content)
	for _, pattern := range fieldPatterns {
		if idx := strings.Index(contentStr, pattern); idx >= 0 {
			// Extract surrounding context
			start := max(0, idx-50)
			end := min(len(contentStr), idx+200)
			context := e.extractReadableText([]byte(contentStr[start:end]))
			if len(context) > 0 {
				if textBuilder.Len() > 0 {
					textBuilder.WriteString("\n")
				}
				textBuilder.WriteString(context)
			}
		}
	}

	result := textBuilder.String()
	if len(result) == 0 {
		// Fallback: extract any readable text
		result = e.extractReadableText(content)
	}

	return result, nil
}

// extractFromRTF extracts text from RTF documents
func (e *DocumentExtractor) extractFromRTF(content []byte) (string, error) {
	rtfStr := string(content)

	// Remove RTF control words and groups
	// This is a simplified extraction - a full RTF parser would be more accurate
	text := rtfStr

	// Remove RTF groups
	text = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(text, " ")

	// Remove RTF control words
	text = regexp.MustCompile(`\\[a-z]+[0-9]*\s?`).ReplaceAllString(text, " ")

	// Convert RTF line breaks to spaces
	text = e.rtfTextPattern.ReplaceAllString(text, " ")

	// Clean up
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return text, nil
}

// extractReadableText extracts readable ASCII/UTF-8 text from binary content
func (e *DocumentExtractor) extractReadableText(content []byte) string {
	var textBuilder strings.Builder
	var currentWord strings.Builder
	minWordLength := 3

	for i := 0; i < len(content); i++ {
		ch := content[i]

		// Check if character is printable ASCII or common UTF-8
		if (ch >= 32 && ch <= 126) || ch == '\n' || ch == '\r' || ch == '\t' {
			currentWord.WriteByte(ch)
		} else {
			// End of readable sequence
			if currentWord.Len() >= minWordLength {
				word := currentWord.String()
				// Filter out hex strings and other non-text patterns
				if !e.isBinaryPattern(word) {
					if textBuilder.Len() > 0 {
						textBuilder.WriteString(" ")
					}
					textBuilder.WriteString(word)
				}
			}
			currentWord.Reset()
		}
	}

	// Don't forget the last word
	if currentWord.Len() >= minWordLength {
		word := currentWord.String()
		if !e.isBinaryPattern(word) {
			if textBuilder.Len() > 0 {
				textBuilder.WriteString(" ")
			}
			textBuilder.WriteString(word)
		}
	}

	return textBuilder.String()
}

// isProbablyText checks if content is likely plain text
func (e *DocumentExtractor) isProbablyText(sample []byte) bool {
	if len(sample) == 0 {
		return false
	}

	printableCount := 0
	for _, b := range sample {
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
			printableCount++
		}
	}

	// If more than 90% printable characters, probably text
	return float64(printableCount)/float64(len(sample)) > 0.9
}

// isBinaryPattern checks if a string looks like binary data rather than text
func (e *DocumentExtractor) isBinaryPattern(s string) bool {
	// Check for common binary patterns

	// All uppercase hex
	if matched, _ := regexp.MatchString(`^[0-9A-F]+$`, s); matched && len(s) > 8 {
		return true
	}

	// Looks like a memory address or pointer
	if matched, _ := regexp.MatchString(`^0x[0-9a-fA-F]+$`, s); matched {
		return true
	}

	// Too many non-alphabetic characters
	nonAlpha := 0
	for _, ch := range s {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == ' ') {
			nonAlpha++
		}
	}

	if float64(nonAlpha)/float64(len(s)) > 0.5 {
		return true
	}

	return false
}

// ExtractMetadata attempts to extract document metadata
func (e *DocumentExtractor) ExtractMetadata(content []byte) map[string]string {
	metadata := make(map[string]string)
	format := e.DetectFormat(content)

	switch format {
	case FormatZipBased:
		// Extract from docProps/core.xml and docProps/app.xml
		reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
		if err != nil {
			return metadata
		}

		for _, file := range reader.File {
			if file.Name == "docProps/core.xml" || file.Name == "docProps/app.xml" {
				rc, err := file.Open()
				if err != nil {
					continue
				}

				data, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					continue
				}

				// Simple extraction of common properties
				e.extractXMLMetadata(string(data), metadata)
			}
		}

	case FormatOLEBased:
		// OLE properties are in specific streams
		// This would require full OLE parsing
		metadata["format"] = "OLE Compound Document"

	case FormatRTF:
		metadata["format"] = "Rich Text Format"

	case FormatPlainText:
		metadata["format"] = "Plain Text"
	}

	return metadata
}

// extractXMLMetadata extracts metadata from XML content
func (e *DocumentExtractor) extractXMLMetadata(xmlContent string, metadata map[string]string) {
	// Extract common metadata fields
	patterns := map[string]*regexp.Regexp{
		"title":       regexp.MustCompile(`<dc:title>([^<]+)</dc:title>`),
		"creator":     regexp.MustCompile(`<dc:creator>([^<]+)</dc:creator>`),
		"description": regexp.MustCompile(`<dc:description>([^<]+)</dc:description>`),
		"subject":     regexp.MustCompile(`<dc:subject>([^<]+)</dc:subject>`),
		"keywords":    regexp.MustCompile(`<cp:keywords>([^<]+)</cp:keywords>`),
		"lastModifiedBy": regexp.MustCompile(`<cp:lastModifiedBy>([^<]+)</cp:lastModifiedBy>`),
		"revision":    regexp.MustCompile(`<cp:revision>([^<]+)</cp:revision>`),
		"application": regexp.MustCompile(`<Application>([^<]+)</Application>`),
		"appVersion":  regexp.MustCompile(`<AppVersion>([^<]+)</AppVersion>`),
	}

	for key, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(xmlContent); len(matches) > 1 {
			metadata[key] = matches[1]
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Helper type for reading binary data
type binaryReader struct {
	data []byte
	pos  int
}

func newBinaryReader(data []byte) *binaryReader {
	return &binaryReader{data: data, pos: 0}
}

func (r *binaryReader) readUint16() (uint16, error) {
	if r.pos+2 > len(r.data) {
		return 0, io.EOF
	}
	val := binary.LittleEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return val, nil
}

func (r *binaryReader) readUint32() (uint32, error) {
	if r.pos+4 > len(r.data) {
		return 0, io.EOF
	}
	val := binary.LittleEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return val, nil
}

// WordDocumentStream represents the main document stream in OLE files
type WordDocumentStream struct {
	Text      string
	FieldCodes []string
	Tables     int
}

// parseWordDocumentStream attempts to parse the Word document stream from OLE
func (e *DocumentExtractor) parseWordDocumentStream(content []byte) (*WordDocumentStream, error) {
	doc := &WordDocumentStream{
		FieldCodes: make([]string, 0),
	}

	// This is a simplified parser - full implementation would need complete OLE/DOC spec
	// For now, we extract readable text and look for patterns

	doc.Text = e.extractReadableText(content)

	// Look for field code patterns
	fieldPatterns := []string{
		"MERGEFIELD",
		"IF ",
		"DOCPROPERTY",
		"DOCVARIABLE",
		"INCLUDETEXT",
		"REF ",
		"=",
	}

	for _, pattern := range fieldPatterns {
		if strings.Contains(doc.Text, pattern) {
			// Extract the field code context
			idx := strings.Index(doc.Text, pattern)
			if idx >= 0 {
				end := idx + 100
				if end > len(doc.Text) {
					end = len(doc.Text)
				}
				fieldCode := doc.Text[idx:end]
				// Clean it up
				fieldCode = strings.Split(fieldCode, "\x00")[0]
				if len(fieldCode) > 0 {
					doc.FieldCodes = append(doc.FieldCodes, fieldCode)
				}
			}
		}
	}

	return doc, nil
}

// DocumentInfo contains extracted document information
type DocumentInfo struct {
	Format     DocumentFormat
	Text       string
	FieldCodes []string
	Metadata   map[string]string
	HasMacros  bool
	TableCount int
}

// AnalyzeDocument performs complete document analysis with text extraction
func (e *DocumentExtractor) AnalyzeDocument(content []byte) (*DocumentInfo, error) {
	info := &DocumentInfo{
		Format:     e.DetectFormat(content),
		FieldCodes: make([]string, 0),
		Metadata:   make(map[string]string),
	}

	// Extract text based on format
	text, err := e.ExtractText(content)
	if err != nil {
		// Even if extraction fails, try to get something
		text = e.extractReadableText(content)
	}
	info.Text = text

	// Extract metadata
	info.Metadata = e.ExtractMetadata(content)

	// Check for macros
	macroIndicators := []string{
		"VBAProject",
		"Macros",
		"Sub ",
		"Function ",
		"End Sub",
		"End Function",
	}

	contentStr := string(content)
	for _, indicator := range macroIndicators {
		if strings.Contains(contentStr, indicator) {
			info.HasMacros = true
			break
		}
	}

	// Count tables (simplified)
	info.TableCount = strings.Count(text, "<table") + strings.Count(text, "\\trowd")

	return info, nil
}