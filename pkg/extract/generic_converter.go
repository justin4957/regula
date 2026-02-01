package extract

import (
	"strconv"
	"strings"

	"github.com/coolbeans/regula/pkg/pattern"
)

// convertGenericDocument transforms a GenericDocument produced by the
// GenericParser into the standard Document model used by the extraction
// pipeline. It maps the flat-but-leveled GenericSection list into a
// Chapter > Section > Article tree using level-based assignment:
//
//   - Level 0 → Chapter
//   - Level 1 → Section (within current Chapter)
//   - Level 2+ → Article (within current Section or Chapter)
//
// Implicit containers are created when a section arrives without a
// parent at the expected level (e.g., a Level 1 section with no
// preceding Level 0 section).
func convertGenericDocument(genericDocument *pattern.GenericDocument) *Document {
	if genericDocument == nil {
		return &Document{
			Chapters: make([]*Chapter, 0),
		}
	}

	doc := &Document{
		Title:       genericDocument.Title,
		Type:        DocumentTypeUnknown,
		Chapters:    make([]*Chapter, 0),
		Definitions: convertGenericDefinitions(genericDocument.Definitions),
	}

	if len(genericDocument.Sections) == 0 {
		return doc
	}

	var currentChapter *Chapter
	var currentSection *Section
	chapterIndex := 0
	sectionIndex := 0
	articleIndex := 0

	for _, genericSection := range genericDocument.Sections {
		switch {
		case genericSection.Level == 0:
			// Top-level: create a new Chapter
			chapterIndex++
			sectionIndex = 0
			articleIndex = 0
			currentChapter = buildChapterFromSection(genericSection, chapterIndex)
			currentSection = nil
			doc.Chapters = append(doc.Chapters, currentChapter)

		case genericSection.Level == 1:
			// Mid-level: create a Section within current Chapter
			if currentChapter == nil {
				chapterIndex++
				currentChapter = &Chapter{
					Number:   strconv.Itoa(chapterIndex),
					Title:    "",
					Sections: make([]*Section, 0),
					Articles: make([]*Article, 0),
				}
				doc.Chapters = append(doc.Chapters, currentChapter)
			}
			sectionIndex++
			articleIndex = 0
			currentSection = &Section{
				Number:   sectionNumberToInt(genericSection.Number, sectionIndex),
				Title:    genericSection.Title,
				Articles: make([]*Article, 0),
			}
			currentChapter.Sections = append(currentChapter.Sections, currentSection)

		default:
			// Deeper level: create an Article
			articleIndex++
			article := buildArticleFromSection(genericSection, articleIndex)

			if currentSection != nil {
				currentSection.Articles = append(currentSection.Articles, article)
			} else if currentChapter != nil {
				currentChapter.Articles = append(currentChapter.Articles, article)
			} else {
				// No container — create an implicit chapter
				chapterIndex++
				currentChapter = &Chapter{
					Number:   strconv.Itoa(chapterIndex),
					Title:    "",
					Sections: make([]*Section, 0),
					Articles: []*Article{article},
				}
				doc.Chapters = append(doc.Chapters, currentChapter)
			}
		}
	}

	// If no chapters were created but sections exist, wrap everything
	// in a single implicit chapter with each section becoming an Article.
	if len(doc.Chapters) == 0 {
		implicitChapter := &Chapter{
			Number:   "1",
			Title:    "",
			Sections: make([]*Section, 0),
			Articles: make([]*Article, 0),
		}
		for i, genericSection := range genericDocument.Sections {
			implicitChapter.Articles = append(implicitChapter.Articles,
				buildArticleFromSection(genericSection, i+1))
		}
		doc.Chapters = append(doc.Chapters, implicitChapter)
	}

	return doc
}

// buildChapterFromSection creates a Chapter from a top-level GenericSection.
func buildChapterFromSection(genericSection *pattern.GenericSection, chapterIndex int) *Chapter {
	chapterNumber := genericSection.Number
	if chapterNumber == "" {
		chapterNumber = strconv.Itoa(chapterIndex)
	}

	chapter := &Chapter{
		Number:   chapterNumber,
		Title:    genericSection.Title,
		Sections: make([]*Section, 0),
		Articles: make([]*Article, 0),
	}

	// If the section has content (not just a header), create an article for it
	if genericSection.Content != "" {
		chapter.Articles = append(chapter.Articles, &Article{
			Number: 1,
			Title:  genericSection.Title,
			Text:   genericSection.Content,
		})
	}

	return chapter
}

// buildArticleFromSection creates an Article from a GenericSection.
func buildArticleFromSection(genericSection *pattern.GenericSection, articleIndex int) *Article {
	articleNumber := sectionNumberToInt(genericSection.Number, articleIndex)
	return &Article{
		Number:    articleNumber,
		SectionID: genericSection.Number,
		Title:     genericSection.Title,
		Text:      genericSection.Content,
	}
}

// convertGenericDefinitions transforms a slice of GenericDefinitions into
// the standard Definition model with sequential numbering.
func convertGenericDefinitions(genericDefinitions []*pattern.GenericDefinition) []*Definition {
	if len(genericDefinitions) == 0 {
		return nil
	}

	definitions := make([]*Definition, 0, len(genericDefinitions))
	for i, genericDefinition := range genericDefinitions {
		definitions = append(definitions, &Definition{
			Number: i + 1,
			Term:   genericDefinition.Term,
			Text:   genericDefinition.Definition,
		})
	}
	return definitions
}

// sectionNumberToInt converts a section number string (arabic, roman
// numeral, or single letter) into an integer. Falls back to
// fallbackIndex when the string cannot be parsed.
func sectionNumberToInt(numberString string, fallbackIndex int) int {
	if numberString == "" {
		return fallbackIndex
	}

	// Try parsing as integer
	if intValue, err := strconv.Atoi(numberString); err == nil {
		return intValue
	}

	// Try parsing as single lowercase letter (a=1, b=2, ...) before roman
	// numerals. Lowercase single characters like 'c', 'i', 'v' are almost
	// always alphabetic list items (e.g., "(a)", "(b)", "(c)"), not roman
	// numerals. Uppercase single characters like 'I', 'V', 'C' are ambiguous
	// but treated as roman numerals below.
	if len(numberString) == 1 {
		ch := numberString[0]
		if ch >= 'a' && ch <= 'z' {
			return int(ch-'a') + 1
		}
	}

	// Try parsing as roman numeral (uppercase single chars + multi-char)
	if romanValue := romanToArabic(numberString); romanValue > 0 {
		return romanValue
	}

	// Try parsing as single uppercase letter (A=1, B=2, ...) for non-roman
	// characters like 'B', 'E', 'F', etc.
	if len(numberString) == 1 {
		ch := numberString[0]
		if ch >= 'A' && ch <= 'Z' {
			return int(ch-'A') + 1
		}
	}

	return fallbackIndex
}

// romanToArabic converts a roman numeral string to its integer value.
// Returns 0 for invalid input.
func romanToArabic(roman string) int {
	if roman == "" {
		return 0
	}

	roman = strings.ToUpper(roman)
	romanValues := map[byte]int{
		'I': 1, 'V': 5, 'X': 10, 'L': 50,
		'C': 100, 'D': 500, 'M': 1000,
	}

	total := 0
	for i := 0; i < len(roman); i++ {
		currentValue, ok := romanValues[roman[i]]
		if !ok {
			return 0 // Invalid character
		}
		if i+1 < len(roman) {
			nextValue, nextOk := romanValues[roman[i+1]]
			if nextOk && currentValue < nextValue {
				total -= currentValue
				continue
			}
		}
		total += currentValue
	}
	return total
}
