package extract

import (
	"strings"
	"testing"
)

// FuzzParser tests the document parser with arbitrary input.
// Run with: go test -fuzz=FuzzParser -fuzztime=30s ./pkg/extract/...
func FuzzParser(f *testing.F) {
	// Add seed corpus from various regulatory document structures
	seeds := []string{
		// EU-style document structure
		`REGULATION (EU) 2016/679

CHAPTER I
GENERAL PROVISIONS

Article 1
Subject-matter and objectives

1. This Regulation lays down rules.
2. This Regulation protects fundamental rights.`,

		// US-style document structure
		`CALIFORNIA CONSUMER PRIVACY ACT

CHAPTER 1. GENERAL PROVISIONS

Section 1798.100. Consumer Rights

(a) A consumer shall have the right to request.
(b) A business shall disclose.`,

		// UK-style document structure
		`Data Protection Act 2018

PART 1
PRELIMINARY

1 Overview
(1) This Act makes provision about the processing of personal data.
(2) Most processing of personal data is subject to the GDPR.`,

		// Definition-heavy section
		`Article 4
Definitions

For the purposes of this Regulation:
(1) 'personal data' means any information;
(2) 'processing' means any operation;
(3) 'controller' means the natural or legal person;`,

		// Reference-heavy section
		`Article 17
Right to erasure ('right to be forgotten')

1. The data subject shall have the right to obtain from the controller the erasure of personal data concerning him or her without undue delay and the controller shall have the obligation to erase personal data without undue delay where one of the following grounds applies:
(a) the personal data are no longer necessary in relation to the purposes for which they were collected or otherwise processed;
(b) the data subject withdraws consent pursuant to point (a) of Article 6(1), or point (a) of Article 9(2), and where there is no other legal ground for the processing;
(c) the data subject objects to the processing pursuant to Article 21(1);`,

		// Recitals/preamble
		`(1) The protection of natural persons in relation to the processing of personal data is a fundamental right.
(2) The principles of, and rules on the protection of natural persons with regard to the processing of their personal data should respect their fundamental rights and freedoms.
(3) Directive 95/46/EC of the European Parliament and of the Council seeks to harmonise the protection.`,

		// Empty and minimal input
		"",
		"Article 1",
		"CHAPTER I",
		"Section 1",

		// Edge cases with special characters
		"Article 1(1)(a)(i)",
		"§ 1798.100",
		"45 C.F.R. Part 164",
		"Directive 95/46/EC",

		// Nested structure
		`CHAPTER I
GENERAL PROVISIONS

Section 1
Scope

Article 1
Subject-matter

1. This paragraph contains:
(a) point a;
(b) point b with:
(i) sub-point i;
(ii) sub-point ii.`,

		// Long line
		strings.Repeat("This is a very long paragraph that might cause buffer issues. ", 100),

		// Unicode characters
		"Article 1 — Subject-matter and objectives « test »",

		// Numbers and special patterns
		"123456789 Article 0 Article -1 Article 999999",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		parser := NewParser()

		// The parser should not panic on any input
		doc, err := parser.Parse(strings.NewReader(data))

		// We don't care about errors, just that it doesn't panic
		if err != nil {
			return
		}

		// Basic sanity checks on returned document
		if doc == nil {
			t.Error("Parser returned nil document without error")
			return
		}

		// Verify document structure is valid
		for _, chapter := range doc.Chapters {
			if chapter == nil {
				t.Error("Document contains nil chapter")
				continue
			}
			for _, article := range chapter.Articles {
				if article == nil {
					t.Error("Chapter contains nil article")
				}
			}
		}
	})
}

// FuzzReferenceExtractor tests the reference extractor with arbitrary text.
// Run with: go test -fuzz=FuzzReferenceExtractor -fuzztime=30s ./pkg/extract/...
func FuzzReferenceExtractor(f *testing.F) {
	// Add seed corpus with various reference patterns
	seeds := []string{
		// EU-style references
		"Article 6(1)(a)",
		"Articles 17 and 18",
		"Articles 12 to 22",
		"point (a) of Article 6(1)",
		"Chapter III",
		"Section 2",
		"paragraph 1",
		"Directive 95/46/EC",
		"Regulation (EU) 2016/679",
		"Regulation (EC) No 45/2001",
		"Decision 2010/87/EU",
		"Treaty on the Functioning of the European Union",
		"TFEU",

		// US-style references
		"Section 1798.100",
		"Section 1798.100(a)",
		"Section 1798.100(a)(1)",
		"Sections 1798.100 through 1798.199",
		"42 U.S.C. § 1983",
		"15 U.S.C. Section 1681",
		"45 C.F.R. Part 164",
		"Public Law 104-191",
		"Pub. L. 111-5",

		// Mixed and complex
		"See Article 17(1)(b) and Directive 95/46/EC",
		"pursuant to Section 1798.100(a)(1) of the CCPA",
		"as defined in 45 C.F.R. Parts 160 and 164",

		// Edge cases
		"Article",
		"Article 0",
		"Article -1",
		"Article 999999999999999999999999999999",
		"Section",
		"paragraph (a)(b)(c)(d)(e)(f)(g)(h)(i)(j)(k)(l)",
		strings.Repeat("Article 1 ", 1000),

		// Malformed references
		"Article (1)",
		"Article 1)",
		"(Article 1",
		"Article1",
		"Articleone",

		// Unicode
		"Article 1 — test",
		"Section § 100",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		extractor := NewReferenceExtractor()

		// Create a minimal article for testing
		article := &Article{
			Number: 1,
			Title:  "Test Article",
			Text:   data,
		}

		// The extractor should not panic on any input
		refs := extractor.ExtractFromArticle(article)

		// Basic sanity checks
		for _, ref := range refs {
			if ref == nil {
				t.Error("Extractor returned nil reference")
				continue
			}
			// Verify reference structure
			if ref.Type == "" {
				t.Error("Reference has empty type")
			}
		}
	})
}

// FuzzDefinitionExtractor tests definition extraction with arbitrary text.
// Run with: go test -fuzz=FuzzDefinitionExtractor -fuzztime=30s ./pkg/extract/...
func FuzzDefinitionExtractor(f *testing.F) {
	// Add seed corpus with definition patterns
	seeds := []string{
		// EU-style definitions
		`(1) 'personal data' means any information relating to an identified or identifiable natural person;`,
		`(2) 'processing' means any operation or set of operations which is performed on personal data;`,
		`(3) 'controller' means the natural or legal person, public authority, agency or other body which determines the purposes;`,

		// US-style definitions
		`"Business" means a sole proprietorship, partnership, corporation, or other legal entity.`,
		`"Consumer" means a natural person who is a California resident.`,
		`"Personal information" means information that identifies, relates to, or describes a consumer.`,

		// UK-style definitions
		`"personal data" means any information relating to an identified or identifiable living individual;`,
		`"the GDPR" means Regulation (EU) 2016/679;`,

		// Multiple definitions
		`For the purposes of this Regulation:
(1) 'personal data' means any information;
(2) 'processing' means any operation;
(3) 'controller' means the natural or legal person;`,

		// Edge cases
		"",
		"(1)",
		`"" means nothing`,
		`'' means empty`,
		"means",
		`(999) 'test' means test`,

		// Long definitions
		`(1) 'test term' means ` + strings.Repeat("a very long definition ", 100),

		// Special characters
		`(1) 'term—with—dashes' means something`,
		`(1) "term with quotes" means something`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		extractor := NewDefinitionExtractor()

		// Create a minimal document with an article containing the definition text
		doc := &Document{
			Title: "Test Document",
			Type:  DocumentTypeRegulation,
			Chapters: []*Chapter{
				{
					Number: "I",
					Title:  "Definitions",
					Articles: []*Article{
						{
							Number: 4,
							Title:  "Definitions",
							Text:   data,
						},
					},
				},
			},
		}

		// The extractor should not panic on any input
		defs := extractor.ExtractDefinitions(doc)

		// Basic sanity checks
		for _, def := range defs {
			if def == nil {
				t.Error("Extractor returned nil definition")
			}
		}
	})
}

// FuzzSemanticExtractor tests semantic annotation extraction with arbitrary text.
// Run with: go test -fuzz=FuzzSemanticExtractor -fuzztime=30s ./pkg/extract/...
func FuzzSemanticExtractor(f *testing.F) {
	// Add seed corpus with semantic patterns
	seeds := []string{
		// Rights
		"The data subject shall have the right to obtain",
		"Consumers have a right to request",
		"Every individual has the right to access",

		// Obligations
		"The controller shall ensure",
		"Businesses must provide",
		"The processor is obligated to",
		"shall be required to",

		// Prohibitions
		"shall not process",
		"is prohibited from",
		"must not disclose",

		// Permissions
		"may process personal data",
		"is permitted to",
		"allowed to transfer",

		// Conditions
		"where the data subject has given consent",
		"if the processing is necessary",
		"provided that adequate safeguards",
		"subject to appropriate safeguards",

		// Exceptions
		"except where",
		"unless the controller demonstrates",
		"notwithstanding paragraph 1",

		// Complex sentences
		"The data subject shall have the right to obtain from the controller the erasure of personal data concerning him or her without undue delay and the controller shall have the obligation to erase personal data without undue delay where one of the following grounds applies.",

		// Edge cases
		"",
		"shall",
		"right",
		"must",
		strings.Repeat("shall have the right ", 100),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		extractor := NewSemanticExtractor()

		// Create a minimal article for testing
		article := &Article{
			Number: 1,
			Title:  "Test Article",
			Text:   data,
		}

		// The extractor should not panic on any input
		annotations := extractor.ExtractFromArticle(article)

		// Basic sanity checks
		for _, ann := range annotations {
			if ann == nil {
				t.Error("Extractor returned nil annotation")
			}
		}
	})
}
