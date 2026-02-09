package deliberation

import "regexp"

// compileDefaultPatterns creates the default generic pattern set.
func compileDefaultPatterns() *minutesPatterns {
	return &minutesPatterns{
		// Date patterns - various formats
		datePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})`),
			regexp.MustCompile(`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2}),?\s+(\d{4})`),
			regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`),
			regexp.MustCompile(`(\d{1,2})/(\d{1,2})/(\d{4})`),
			regexp.MustCompile(`(?i)Date[:\s]+(.+?)(?:\n|$)`),
		},

		// Time patterns
		timePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:at|time)[:\s]+(\d{1,2}[:.]\d{2}(?:\s*[AP]M)?)`),
			regexp.MustCompile(`(\d{1,2}[:.]\d{2})\s*(?:hours|hrs|h)`),
			regexp.MustCompile(`(?i)(?:convened|opened|began)[^.]*?(\d{1,2}[:.]\d{2})`),
		},

		// Location pattern
		locationPattern: regexp.MustCompile(`(?i)(?:Location|Venue|Place|Room)[:\s]+(.+?)(?:\n|$)`),

		// Meeting number/sequence
		meetingNumPattern: regexp.MustCompile(`(?i)(?:Meeting|Session)\s+(?:No\.?\s*)?(\d+)`),

		// Session pattern
		sessionPattern: regexp.MustCompile(`(?i)(\d+)(?:st|nd|rd|th)\s+(?:Session|Meeting)`),

		// Series pattern
		seriesPattern: regexp.MustCompile(`(?i)(?:Working\s+Group|Committee|Board|Council)\s+([A-Z0-9]+(?:\s+[A-Z0-9]+)*)`),

		// Chair pattern - handles "Chair: Ms. Name (Country)" format
		chairPattern: regexp.MustCompile(`(?i)(?:Chair(?:man|woman|person)?|Presiding)[:\s]+(?:(?:Mr|Ms|Mrs|Dr|Prof)\.?\s+)?([A-Z][a-zA-Z\s.'-]+?)(?:\s*\([A-Z]+\))?(?:\n|,|$)`),

		// Attendees pattern
		attendeesPattern: regexp.MustCompile(`(?i)(?:Present|Attendees|Members\s+present|Participants)[:\s]+(.+?)(?:\n\n|\nApolog|Chair)`),

		// Apologies pattern
		apologyPattern: regexp.MustCompile(`(?i)(?:Apologies|Absent|Regrets)[:\s]+(.+?)(?:\n\n|\n[A-Z])`),

		// Agenda item patterns - various numbering styles
		agendaItemPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?m)^(\d+)\.\s+(.+?)$`),                                  // "1. Item title"
			regexp.MustCompile(`(?m)^Item\s+(\d+)[.:\s]+(.+?)$`),                         // "Item 1: Title"
			regexp.MustCompile(`(?m)^(\d+)\)\s+(.+?)$`),                                  // "1) Item title"
			regexp.MustCompile(`(?m)^([IVXLCDM]+)\.\s+(.+?)$`),                           // "I. Item title"
			regexp.MustCompile(`(?m)^([a-z])\)\s+(.+?)$`),                                // "a) Item title"
			regexp.MustCompile(`(?i)(?:Agenda\s+)?Item\s+(\d+(?:\.\d+)?)[:\s]+([^\n]+)`), // "Agenda Item 3.1: Title"
		},

		// Sub-item pattern
		subItemPattern: regexp.MustCompile(`(?m)^\s+(\d+\.\d+|\([a-z]\))\s+(.+?)$`),

		// Speaker patterns - various attribution styles
		speakerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:The\s+)?(?:representative|delegate|member)\s+(?:of|from)\s+([A-Z][a-zA-Z\s]+?)\s+(?:stated|said|noted|expressed|proposed|suggested|asked|requested|observed)`),
			regexp.MustCompile(`(?i)([A-Z][a-zA-Z\s.'-]+?)\s+(?:stated|said|noted|expressed|proposed|suggested|asked|requested|observed)`),
			regexp.MustCompile(`(?i)(?:Mr|Ms|Mrs|Dr|Prof)\.?\s+([A-Z][a-zA-Z\s.'-]+?)\s+(?:stated|said|noted|expressed|proposed|suggested|asked|requested|observed)`),
			regexp.MustCompile(`(?i)The\s+(Chair(?:man|woman|person)?|Secretary|Rapporteur)\s+(?:stated|said|noted|explained|clarified|announced)`),
			regexp.MustCompile(`([A-Z][A-Z\s]+?):\s+`), // "MEMBER STATE X: " (EU style)
		},

		// Motion pattern
		motionPattern: regexp.MustCompile(`(?i)(?:moved|proposed|put\s+forward)\s+(?:that\s+)?(.+?)(?:\.|$)`),

		// Amendment pattern
		amendmentPattern: regexp.MustCompile(`(?i)(?:amendment|amend)\s+(?:to\s+)?(.+?)(?:\.|$)`),

		// Seconded pattern
		secondedPattern: regexp.MustCompile(`(?i)seconded\s+by\s+([A-Z][a-zA-Z\s.'-]+)`),

		// Decision patterns
		decisionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:The\s+(?:committee|group|council|board)\s+)?(?:decided|agreed|resolved)\s+(?:that\s+)?(.+?)(?:\.|$)`),
			regexp.MustCompile(`(?i)(?:Decision|Resolution)[:\s]+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)(?:It\s+was\s+)?(?:decided|agreed|resolved)\s+(?:to\s+)?(.+?)(?:\.|$)`),
			regexp.MustCompile(`(?i)(?:DECISION|AGREED|RESOLVED)[:\s]+(.+?)(?:\n|$)`),
		},

		// Vote pattern - captures for/against/abstain
		votePattern: regexp.MustCompile(`(?i)(?:vote|voting)[:\s]+(\d+)\s*(?:for|in\s+favour)[,\s]+(\d+)\s*against[,\s]+(\d+)\s*abstain`),

		// Adopted pattern
		adoptedPattern: regexp.MustCompile(`(?i)(?:was\s+)?adopted(?:\s+(?:by|with|unanimously))?[:\s]*(.+?)(?:\.|$)`),

		// Rejected pattern
		rejectedPattern: regexp.MustCompile(`(?i)(?:was\s+)?rejected[:\s]*(.+?)(?:\.|$)`),

		// Action patterns
		actionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\[?ACTION\]?[:\s]+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)(?:action|task)[:\s]+([A-Z][a-zA-Z\s.'-]+?)\s+(?:to|will)\s+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)(?:The\s+)?([A-Z][a-zA-Z\s]+?)\s+(?:was\s+)?(?:asked|requested|invited)\s+to\s+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)(?:The\s+)?(?:Secretariat|Secretary)\s+(?:was\s+)?(?:asked|requested|instructed)\s+to\s+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)(?:agreed|decided)\s+(?:that\s+)?([A-Z][a-zA-Z\s]+?)\s+(?:would|should|will)\s+(.+?)(?:\n|$)`),
		},

		// Document reference pattern
		documentRefPattern: regexp.MustCompile(`(?i)(?:document|doc\.?|paper)\s+(?:No\.?\s*)?([A-Z0-9/\-]+)`),

		// Article reference pattern
		articleRefPattern: regexp.MustCompile(`(?i)Article\s+(\d+(?:\(\d+\))?)`),

		// Meeting reference pattern
		meetingRefPattern: regexp.MustCompile(`(?i)(?:previous|last|next)\s+meeting|meeting\s+(?:of|on)\s+(\d{1,2}\s+\w+)`),
	}
}

// compileEUPatterns creates patterns optimized for EU-style meeting minutes.
func compileEUPatterns() *minutesPatterns {
	p := compileDefaultPatterns()

	// Override with EU-specific patterns
	p.speakerPatterns = append([]*regexp.Regexp{
		// EU delegation style: "MEMBER STATE:" or "The MEMBER STATE representative"
		regexp.MustCompile(`(?i)(?:The\s+)?([A-Z]{2,}(?:\s+[A-Z]+)*)\s+(?:delegation|representative)?\s*(?:stated|said|noted|expressed|proposed|suggested|asked|requested|observed|:)`),
		// Explicit delegation mention
		regexp.MustCompile(`(?i)(?:The\s+)?delegation\s+(?:of|from)\s+([A-Z][a-zA-Z\s]+?)\s+(?:stated|said|noted|expressed|proposed|suggested)`),
		// Commission/Council representatives
		regexp.MustCompile(`(?i)(?:The\s+)?(Commission|Council|Presidency)\s+(?:representative\s+)?(?:stated|said|noted|explained|clarified)`),
	}, p.speakerPatterns...)

	// EU document references
	p.documentRefPattern = regexp.MustCompile(`(?i)(?:document|doc\.?)\s+(?:No\.?\s*)?(\d+/\d+(?:/\d+)?(?:\s+REV\s*\d*)?|ST\s*\d+/\d+|[A-Z]+\(\d+\)\d+)`)

	// Council/EP style agenda
	p.agendaItemPatterns = append([]*regexp.Regexp{
		regexp.MustCompile(`(?m)^(\d+)\.\s+([A-Z][^\n]+)`),
		regexp.MustCompile(`(?i)(?:Agenda\s+)?(?:point|item)\s+(\d+(?:\.\d+)?)[:\s]+([^\n]+)`),
	}, p.agendaItemPatterns...)

	// EU decision patterns
	p.decisionPatterns = append([]*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:The\s+)?(?:Council|Committee|Working\s+Group)\s+(?:decided|agreed|approved|endorsed)\s+(?:that\s+)?(.+?)(?:\.|$)`),
		regexp.MustCompile(`(?i)(?:OUTCOME|CONCLUSIONS?)[:\s]+(.+?)(?:\n\n|$)`),
	}, p.decisionPatterns...)

	return p
}

// compileUNPatterns creates patterns optimized for UN-style meeting minutes.
func compileUNPatterns() *minutesPatterns {
	p := compileDefaultPatterns()

	// UN-specific speaker patterns
	p.speakerPatterns = append([]*regexp.Regexp{
		// UN delegation style
		regexp.MustCompile(`(?i)(?:The\s+)?representative\s+of\s+([A-Z][a-zA-Z\s]+?)\s+(?:said|stated|noted|expressed|proposed|emphasized)`),
		// UN officials
		regexp.MustCompile(`(?i)(?:The\s+)?(Secretary-General|President|Vice-President|Rapporteur)\s+(?:said|stated|noted|explained)`),
		// Member state direct
		regexp.MustCompile(`([A-Z][a-zA-Z\s]+?)(?:,\s+speaking\s+(?:on\s+behalf\s+of|for))?,?\s+(?:said|stated|noted|expressed)`),
	}, p.speakerPatterns...)

	// UN document references
	p.documentRefPattern = regexp.MustCompile(`(?i)(?:document|resolution)\s+(?:No\.?\s*)?([A-Z]/(?:RES/)?[\d/]+|[A-Z]+/C\.\d+/[\d/]+)`)

	// UN session pattern
	p.sessionPattern = regexp.MustCompile(`(?i)(\d+)(?:st|nd|rd|th)\s+(?:session|plenary\s+meeting)`)

	// UN decision patterns
	p.decisionPatterns = append([]*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:The\s+)?(?:General\s+Assembly|Security\s+Council|Committee)\s+(?:decided|resolved|adopted)\s+(?:that\s+)?(.+?)(?:\.|$)`),
		regexp.MustCompile(`(?i)(?:Draft\s+)?resolution\s+(.+?)\s+was\s+adopted`),
	}, p.decisionPatterns...)

	// UN vote pattern with regional groups
	p.votePattern = regexp.MustCompile(`(?i)(?:vote|voting)[:\s]+(\d+)\s*(?:for|in\s+favour)[,\s]+(\d+)\s*against[,\s]+(\d+)\s*abstentions?`)

	return p
}
