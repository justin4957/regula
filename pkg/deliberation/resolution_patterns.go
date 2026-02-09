package deliberation

import "regexp"

// compileGenericResolutionPatterns creates default patterns for resolution parsing.
func compileGenericResolutionPatterns() *resolutionPatterns {
	return &resolutionPatterns{
		// Identifier patterns
		identifierPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)Resolution\s+(\d+[/-]\d+(?:[/-]\d+)?)`),
			regexp.MustCompile(`(?i)Decision\s+(\d+[/-]\d+(?:[/-]\d+)?)`),
			regexp.MustCompile(`[A-Z]/RES/\d+/\d+`),
			regexp.MustCompile(`\d{4}/\d+/[A-Z]+`),
		},

		// Date patterns
		datePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})`),
			regexp.MustCompile(`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2}),?\s+(\d{4})`),
			regexp.MustCompile(`(?i)(?:adopted|dated?)[:\s]+(\d{1,2})\s+(\w+)\s+(\d{4})`),
		},

		// Body pattern
		bodyPattern: regexp.MustCompile(`(?i)(?:The\s+)?(General\s+Assembly|Security\s+Council|Board\s+of\s+Directors|Committee|Council|Commission)(?:\s+of\s+[^,\n]+)?`),

		// Session pattern
		sessionPattern: regexp.MustCompile(`(?i)(\d+)(?:st|nd|rd|th)\s+(?:session|meeting|plenary)`),

		// Title pattern
		titlePattern: regexp.MustCompile(`(?i)(?:Resolution|Decision)\s+(?:\d+[/-]\d+[/-]?\d*\s+)?(?:on\s+)?([^\n]+?)(?:\n|$)`),

		// Preamble patterns
		preambleStart: regexp.MustCompile(`(?i)(?:The\s+(?:General\s+Assembly|Council|Committee|Board)|Preamble)`),
		preambleEnd:   regexp.MustCompile(`(?i)(?:^\s*1\.|Decides|Resolves|Adopts|Hereby)`),

		// Recital patterns - match paragraphs starting with recital phrases
		recitalPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?im)^([A-Z][a-z]+ing[^,\n]*(?:,\s*[^\n]+)?)`),
			regexp.MustCompile(`(?m)^\((\d+)\)\s+([^\n]+)`),
		},

		// Intro phrases for recitals
		introPhrasesRe: regexp.MustCompile(`(?i)^(Recalling|Reaffirming|Noting|Noting with (?:concern|satisfaction|appreciation)|Recognizing|Acknowledging|Aware|Being aware|Bearing in mind|Considering|Convinced|Deeply (?:concerned|convinced)|Deploring|Determined|Emphasizing|Encouraged|Expressing|Guided by|Having (?:considered|examined|heard|received|regard)|Mindful|Observing|Reiterating|Stressing|Taking (?:into account|note)|Underlining|Welcoming|Whereas)\b`),

		// Operative patterns
		operativeStart: regexp.MustCompile(`(?im)(?:^\s*1\.\s+|Decides|Resolves|Hereby)`),
		clausePattern:  regexp.MustCompile(`(?m)^\s*(\d+)\.\s+`),
		subClausePattern: regexp.MustCompile(`(?m)^\s*\(([a-z]|[ivx]+|\d+)\)\s+(.+?)(?:\n|$)`),

		// Action verbs in operative clauses
		actionVerbPattern: regexp.MustCompile(`(?i)^(Decides|Resolves|Requests|Calls upon|Invites|Urges|Encourages|Recommends|Endorses|Approves|Adopts|Authorizes|Affirms|Confirms|Declares|Determines|Directs|Notes|Welcomes|Expresses|Condemns|Deplores|Demands|Insists|Reiterates|Stresses|Emphasizes)\b`),

		// Vote patterns
		votePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\d+)\s*(?:votes?\s+)?(?:in\s+favour|for)[,\s]+(\d+)\s*(?:votes?\s+)?against[,\s]+(\d+)\s*abstentions?`),
			regexp.MustCompile(`(?i)(?:adopted\s+by\s+)?(\d+)\s+to\s+(\d+)(?:,?\s+with\s+(\d+)\s+abstentions?)?`),
			regexp.MustCompile(`(?i)vote[:\s]+(\d+)-(\d+)-(\d+)`),
		},

		// Adoption pattern
		adoptionPattern: regexp.MustCompile(`(?i)(?:was\s+)?adopted(?:\s+(?:unanimously|without\s+(?:a\s+)?vote))?`),

		// Consensus pattern
		consensusPattern: regexp.MustCompile(`(?i)(?:adopted\s+)?(?:by\s+)?consensus|without\s+(?:a\s+)?vote|unanimously`),

		// Reference patterns
		resolutionRefPattern: regexp.MustCompile(`(?i)(?:resolution|decision)\s+(\d+[/-]\d+(?:[/-]\d+)?|[A-Z]/RES/\d+/\d+)`),
		treatyRefPattern:     regexp.MustCompile(`(?i)(?:Charter|Convention|Treaty|Covenant)\s+(?:of|on)\s+[A-Z][a-zA-Z\s]+`),
		regulationRefPattern: regexp.MustCompile(`(?i)(?:Regulation|Directive)\s+\(?(?:EU|EC)\)?\s*(?:No\.?\s*)?\d+/\d+`),
		reportRefPattern:     regexp.MustCompile(`(?i)(?:report|document)\s+(?:No\.?\s*)?([A-Z0-9/\-]+)`),
	}
}

// compileUNResolutionPatterns creates patterns optimized for UN resolutions.
func compileUNResolutionPatterns() *resolutionPatterns {
	p := compileGenericResolutionPatterns()

	// UN-specific identifier patterns
	p.identifierPatterns = []*regexp.Regexp{
		regexp.MustCompile(`[A-Z]/RES/(\d+)/(\d+)`),                       // A/RES/79/100
		regexp.MustCompile(`[A-Z]/(\d+)/L\.(\d+)`),                        // A/79/L.45 (draft)
		regexp.MustCompile(`S/RES/(\d+)\s*\((\d{4})\)`),                   // S/RES/2728 (2024)
		regexp.MustCompile(`(?i)Resolution\s+(\d+)/(\d+)`),                // Resolution 79/100
		regexp.MustCompile(`E/RES/(\d{4})/(\d+)`),                         // ECOSOC
	}

	// UN body patterns
	p.bodyPattern = regexp.MustCompile(`(?i)(?:The\s+)?(General\s+Assembly|Security\s+Council|Economic\s+and\s+Social\s+Council|Human\s+Rights\s+Council|Trusteeship\s+Council)`)

	// UN session pattern
	p.sessionPattern = regexp.MustCompile(`(?i)(\d+)(?:st|nd|rd|th)\s+(?:session|plenary\s+meeting|meeting)`)

	// UN-specific preamble detection
	p.preambleStart = regexp.MustCompile(`(?i)(?:The\s+General\s+Assembly|The\s+Security\s+Council)`)
	p.preambleEnd = regexp.MustCompile(`(?im)(?:^\s*1\.\s+[A-Z]|Decides|Resolves|Adopts)`)

	// UN vote patterns
	p.votePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:A\s+)?recorded\s+vote[:\s]+(\d+)\s+(?:in\s+favour|to)[,\s]+(\d+)\s+against[,\s]+(\d+)\s+abstentions?`),
		regexp.MustCompile(`(?i)adopted\s+by\s+(?:a\s+vote\s+of\s+)?(\d+)\s+to\s+(\d+),?\s+with\s+(\d+)\s+abstentions?`),
		regexp.MustCompile(`(?i)Vote:\s*(\d+)-(\d+)-(\d+)`),
	}

	// UN document references
	p.resolutionRefPattern = regexp.MustCompile(`(?i)(?:resolution\s+)?([A-Z]/RES/\d+/\d+|\d+/\d+)`)
	p.reportRefPattern = regexp.MustCompile(`(?i)(?:report|document)\s+([A-Z]/\d+/\d+(?:/\w+)?)`)

	return p
}

// compileEUResolutionPatterns creates patterns optimized for EU decisions/resolutions.
func compileEUResolutionPatterns() *resolutionPatterns {
	p := compileGenericResolutionPatterns()

	// EU-specific identifier patterns
	p.identifierPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)Decision\s+\(EU\)\s*(\d{4}/\d+)`),         // Decision (EU) 2024/123
		regexp.MustCompile(`(?i)Council\s+Decision\s+(\d{4}/\d+/\w+)`),    // Council Decision 2024/123/EU
		regexp.MustCompile(`(\d{4}/\d+/EU)`),                               // 2024/123/EU
		regexp.MustCompile(`(?i)Regulation\s+\(EU\)\s*(\d{4}/\d+)`),       // Regulation (EU) 2024/123
	}

	// EU body patterns
	p.bodyPattern = regexp.MustCompile(`(?i)(?:The\s+)?(European\s+Parliament|Council(?:\s+of\s+the\s+European\s+Union)?|European\s+Council|Commission)`)

	// EU-specific preamble (uses "whereas" and "having regard")
	p.preambleStart = regexp.MustCompile(`(?i)(?:THE\s+(?:EUROPEAN\s+PARLIAMENT|COUNCIL)|Having\s+regard)`)
	p.preambleEnd = regexp.MustCompile(`(?i)(?:HAS\s+ADOPTED|HAVE\s+ADOPTED|DECIDES|Article\s+1)`)

	// EU recital patterns - numbered recitals in parentheses
	p.recitalPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\((\d+)\)\s+([^\n]+)`),
		regexp.MustCompile(`(?im)^(Whereas[^.]+\.)`),
		regexp.MustCompile(`(?im)^(Having\s+regard[^.]+\.)`),
	}

	// EU intro phrases
	p.introPhrasesRe = regexp.MustCompile(`(?i)^(Having\s+regard|Whereas|Acting\s+in\s+accordance|On\s+a\s+proposal|After\s+consulting|After\s+obtaining|Whereas)\b`)

	// EU operative patterns - uses Articles instead of numbered clauses
	p.operativeStart = regexp.MustCompile(`(?im)(?:HAS\s+ADOPTED|HAVE\s+ADOPTED|Article\s+1)`)
	p.clausePattern = regexp.MustCompile(`(?im)^Article\s+(\d+)\b`)

	// EU action verbs
	p.actionVerbPattern = regexp.MustCompile(`(?i)^(?:Article\s+\d+\s+)?(?:shall|is\s+hereby|The\s+\w+\s+shall)\b`)

	// EU document references
	p.resolutionRefPattern = regexp.MustCompile(`(?i)(?:Decision|Regulation|Directive)\s+(?:\(EU\)\s*)?(\d{4}/\d+(?:/\w+)?)`)
	p.regulationRefPattern = regexp.MustCompile(`(?i)(?:Regulation|Directive)\s+\(EU\)\s*(?:No\.?\s*)?(\d+/\d+)`)

	return p
}
