package citation

import (
	"strings"
	"testing"
)

// FuzzEUCitationParser tests the EU citation parser with arbitrary input.
// Run with: go test -fuzz=FuzzEUCitationParser -fuzztime=30s ./pkg/citation/...
func FuzzEUCitationParser(f *testing.F) {
	// Add seed corpus with EU citation patterns
	seeds := []string{
		// Regulations
		"Regulation (EU) 2016/679",
		"Regulation (EC) No 45/2001",
		"Regulation (EU) 2018/1725",
		"Regulation (EC) No 1049/2001",

		// Directives
		"Directive 95/46/EC",
		"Directive (EU) 2016/680",
		"Directive 2002/58/EC",
		"Directive (EU) 2015/1535",

		// Decisions
		"Decision 2010/87/EU",
		"Decision 2016/2102/EU",

		// Treaties
		"TFEU",
		"TEU",
		"Treaty on the Functioning of the European Union",
		"Treaty on European Union",

		// Article references
		"Article 1",
		"Article 6(1)(a)",
		"Article 17(1)(b)",
		"Article 9(2)(a)",
		"Articles 12 and 14",
		"Articles 12 to 22",

		// Chapter and Section references
		"Chapter I",
		"Chapter III",
		"Chapter XIV",
		"Section 1",
		"Section 5",

		// Complex references
		"Article 6(1), point (a) of Article 9(2)",
		"pursuant to Regulation (EU) 2016/679, Article 17",
		"Directive 95/46/EC, as amended by Regulation (EU) 2016/679",

		// Edge cases
		"",
		"Regulation",
		"Article",
		"Directive",
		"Chapter",
		"Regulation (XX) 1234/567",
		"Article 0",
		"Article -1",
		"Article 999999999999999999999999999",
		"Chapter 0",
		"Chapter Z",

		// Multiple citations in text
		"See Regulation (EU) 2016/679, Directive 95/46/EC, and Article 17(1)(a).",
		strings.Repeat("Article 1 ", 1000),

		// Malformed patterns
		"Regulation () 2016/679",
		"Regulation (EU)",
		"Regulation 2016/679",
		"Article (1)(2)",
		"Directive //",

		// Unicode and special characters
		"Article 1 — Subject-matter",
		"Regulation (EU) 2016/679 «GDPR»",
		"Article 17 'Right to erasure'",

		// Partial matches
		"This regulation",
		"article of clothing",
		"directive from management",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		parser := NewEUCitationParser()

		// The parser should not panic on any input
		citations, err := parser.Parse(data)

		// We don't care about errors for malformed input
		if err != nil {
			return
		}

		// Basic sanity checks
		for _, cit := range citations {
			if cit == nil {
				t.Error("Parser returned nil citation")
				continue
			}
			if cit.Type == "" {
				t.Error("Citation has empty type")
			}
			if cit.RawText == "" {
				t.Error("Citation has empty raw text")
			}
		}

		// Test normalization doesn't panic
		for _, cit := range citations {
			_ = parser.Normalize(cit)
		}
	})
}

// FuzzBluebookParser tests the Bluebook (US) citation parser with arbitrary input.
// Run with: go test -fuzz=FuzzBluebookParser -fuzztime=30s ./pkg/citation/...
func FuzzBluebookParser(f *testing.F) {
	// Add seed corpus with US citation patterns
	seeds := []string{
		// U.S. Code citations
		"42 U.S.C. § 1983",
		"15 U.S.C. § 1681",
		"15 U.S.C. §§ 1681-1681x",
		"42 U.S.C. Section 1320d",
		"15 U.S.C. Sec. 1681",
		"15 U.S.C. Section 1681 et seq.",

		// C.F.R. citations
		"45 C.F.R. Part 164",
		"45 C.F.R. § 164.502",
		"21 C.F.R. Part 50",
		"45 C.F.R. Parts 160 and 164",

		// Public Law citations
		"Public Law 104-191",
		"Pub. L. 111-5",
		"P.L. 104-191",

		// Case citations
		"Brown v. Board of Education, 347 U.S. 483 (1954)",
		"Roe v. Wade, 410 U.S. 113 (1973)",
		"Marbury v. Madison, 5 U.S. 137 (1803)",

		// State code sections (California style)
		"Section 1798.100",
		"Section 1798.100(a)",
		"Section 1798.100(a)(1)",
		"Sections 1798.100 through 1798.199",

		// Complex citations
		"42 U.S.C. § 1983, as amended by Pub. L. 111-5",
		"45 C.F.R. Parts 160 and 164, implementing Public Law 104-191",

		// Edge cases
		"",
		"U.S.C.",
		"C.F.R.",
		"Section",
		"42 U.S.C.",
		"U.S.C. § 1983",
		"45 C.F.R.",
		"Public Law",
		"Pub. L.",

		// Invalid numbers
		"0 U.S.C. § 0",
		"-1 U.S.C. § -1",
		"999999999 U.S.C. § 999999999999999999",

		// Multiple citations
		"See 42 U.S.C. § 1983, 15 U.S.C. § 1681, and 45 C.F.R. Part 164.",
		strings.Repeat("42 U.S.C. § 1983 ", 1000),

		// Malformed patterns
		"U.S.C. §",
		"C.F.R. Part",
		"v. Board",
		"(1954)",

		// Unicode and special characters
		"42 U.S.C. § 1983 — civil rights",
		"45 C.F.R. Part 164 HIPAA",

		// Partial matches
		"user's code",
		"configuration",
		"public law enforcement",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		parser := NewBluebookParser()

		// The parser should not panic on any input
		citations, err := parser.Parse(data)

		// We don't care about errors for malformed input
		if err != nil {
			return
		}

		// Basic sanity checks
		for _, cit := range citations {
			if cit == nil {
				t.Error("Parser returned nil citation")
				continue
			}
			if cit.Type == "" {
				t.Error("Citation has empty type")
			}
			if cit.RawText == "" {
				t.Error("Citation has empty raw text")
			}
		}

		// Test normalization doesn't panic
		for _, cit := range citations {
			_ = parser.Normalize(cit)
		}
	})
}

// FuzzOSCOLAParser tests the OSCOLA (UK) citation parser with arbitrary input.
// Run with: go test -fuzz=FuzzOSCOLAParser -fuzztime=30s ./pkg/citation/...
func FuzzOSCOLAParser(f *testing.F) {
	// Add seed corpus with UK citation patterns
	seeds := []string{
		// UK Acts
		"Data Protection Act 2018",
		"Human Rights Act 1998",
		"the European Union (Withdrawal) Act 2018",
		"Equality Act 2010",
		"Freedom of Information Act 2000",

		// Act chapter references
		"[2018 c. 12]",
		"[1998 c. 42]",
		"[2018 c. 16]",

		// Statutory Instruments
		"SI 2019/419",
		"SI 2018/1400",
		"Statutory Instruments 2019 No. 419",
		"Statutory Instrument 2018 No. 1400",

		// Section references
		"s 6",
		"s 6(1)",
		"s 6(1)(a)",
		"ss 12",
		"section 114",
		"section 21(1)",
		"sections 12",

		// Neutral citations
		"[2019] UKSC 5",
		"[2021] EWCA Civ 1234",
		"[2020] EWCA Crim 456",
		"[2022] EWHC 789",
		"[2004] UKHL 56",
		"[2023] UKPC 10",
		"[2021] UKUT 100",
		"[2022] UKFTT 50",

		// Law report citations
		"[1994] 1 AC 212",
		"[2003] QB 195",
		"[2020] 2 WLR 456",

		// ECHR references
		"ECHR art 8",
		"ECHR article 6",
		"ECHR art 6(1)",
		"ECHR art 10",

		// Structural references
		"Schedule 7",
		"Schedule 12",
		"Part 3",
		"Part 7",

		// Complex mixed text
		"Data Protection Act 2018 [2018 c. 12] implements Regulation (EU) 2016/679",
		"under s 6(1) of the Act and ECHR art 8",
		"see Schedule 7 to the Data Protection Act 2018",

		// Edge cases
		"",
		"Act",
		"section",
		"Part",
		"Schedule",
		"SI",
		"ECHR",

		// Malformed patterns
		"Act 2018",
		"s",
		"section (1)",
		"[] UKSC 5",
		"[abcd] UKSC 5",
		"SI /",
		"ECHR art",
		"Schedule",
		"Part",

		// Large input
		strings.Repeat("s 6(1) ", 1000),
		strings.Repeat("Data Protection Act 2018 ", 100),

		// Unicode and special characters
		"Data Protection Act 2018 -- amended",
		"s 6(1) of the Act",

		// Partial matches
		"acting in accordance",
		"partial section of text",
		"this is part of the whole",
		"scheduled for later",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		parser := NewOSCOLAParser()

		// The parser should not panic on any input
		citations, err := parser.Parse(data)

		// We don't care about errors for malformed input
		if err != nil {
			return
		}

		// Basic sanity checks
		for _, cit := range citations {
			if cit == nil {
				t.Error("Parser returned nil citation")
				continue
			}
			if cit.Type == "" {
				t.Error("Citation has empty type")
			}
			if cit.RawText == "" {
				t.Error("Citation has empty raw text")
			}
		}

		// Test normalization doesn't panic
		for _, cit := range citations {
			_ = parser.Normalize(cit)
		}

		// Test ToURI doesn't panic
		for _, cit := range citations {
			_, _ = parser.ToURI(cit)
		}
	})
}

// FuzzCitationRegistry tests the citation registry with arbitrary input.
// Run with: go test -fuzz=FuzzCitationRegistry -fuzztime=30s ./pkg/citation/...
func FuzzCitationRegistry(f *testing.F) {
	// Add seed corpus with mixed citations
	seeds := []string{
		// Mixed EU and US
		"Regulation (EU) 2016/679 and 42 U.S.C. § 1983",
		"Article 17 and Section 1798.100",
		"Directive 95/46/EC compared to 45 C.F.R. Part 164",

		// Real document excerpts
		"pursuant to Article 6(1)(a) of Regulation (EU) 2016/679",
		"as defined in 45 C.F.R. Parts 160 and 164, implementing Public Law 104-191",
		"See Brown v. Board of Education, 347 U.S. 483 (1954)",

		// Mixed EU/US/UK
		"Data Protection Act 2018 implements Regulation (EU) 2016/679",
		"s 6(1) and Article 17 compared to 42 U.S.C. § 1983",
		"[2019] UKSC 5 under ECHR art 8 and Directive 95/46/EC",
		"SI 2019/419 amends the Data Protection Act 2018",

		// Edge cases
		"",
		"no citations here",
		strings.Repeat("text without citations ", 100),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		registry := NewCitationRegistry()
		registry.Register(NewEUCitationParser())
		registry.Register(NewBluebookParser())
		registry.Register(NewOSCOLAParser())

		// The registry should not panic on any input
		// Try EU, US, and UK jurisdictions
		citations := registry.ParseAll(data, "EU")
		citationsUS := registry.ParseAll(data, "US")
		citationsUK := registry.ParseAll(data, "UK")
		citations = append(citations, citationsUS...)
		citations = append(citations, citationsUK...)

		// Basic sanity checks
		for _, cit := range citations {
			if cit == nil {
				t.Error("Registry returned nil citation")
			}
		}
	})
}
