package crawler

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/coolbeans/regula/pkg/uscode"
)

// SourceResolver maps citations and URNs to fetchable URLs and document IDs.
// It knows about US law sources: USC (uscode.house.gov), CFR (ecfr.gov),
// state codes, and LII (law.cornell.edu) as a fallback.
type SourceResolver struct {
	// citationPatterns maps regex patterns to resolution functions.
	citationPatterns []*citationPattern
}

// citationPattern pairs a regex with a resolution function.
type citationPattern struct {
	name    string
	pattern *regexp.Regexp
	resolve func(matches []string) (*ResolvedSource, error)
}

// ResolvedSource is the result of resolving a citation to a fetchable URL.
type ResolvedSource struct {
	// URL is the fetchable HTTP URL.
	URL string

	// DocumentID is the derived document ID for the library.
	DocumentID string

	// Domain is the target domain.
	Domain string

	// SourceName is the human-readable source name.
	SourceName string

	// Citation is the original citation text.
	Citation string
}

// NewSourceResolver creates a SourceResolver with pre-registered US law patterns.
func NewSourceResolver() *SourceResolver {
	resolver := &SourceResolver{}
	resolver.registerUSPatterns()
	return resolver
}

// Resolve attempts to resolve a citation string to a fetchable URL and document ID.
// It tries all registered patterns in order and returns the first match.
func (resolver *SourceResolver) Resolve(citation string) (*ResolvedSource, error) {
	normalizedCitation := strings.TrimSpace(citation)
	if normalizedCitation == "" {
		return nil, fmt.Errorf("empty citation")
	}

	for _, citPattern := range resolver.citationPatterns {
		matches := citPattern.pattern.FindStringSubmatch(normalizedCitation)
		if matches != nil {
			resolved, err := citPattern.resolve(matches)
			if err != nil {
				continue
			}
			resolved.Citation = normalizedCitation
			return resolved, nil
		}
	}

	return nil, fmt.Errorf("unrecognized citation format: %s", normalizedCitation)
}

// ResolveURN resolves a URN-style identifier (from the triple store) to a fetchable URL.
func (resolver *SourceResolver) ResolveURN(urn string) (*ResolvedSource, error) {
	if urn == "" {
		return nil, fmt.Errorf("empty URN")
	}

	switch {
	case strings.HasPrefix(urn, "urn:us:usc:"):
		return resolver.resolveUSCURN(urn)
	case strings.HasPrefix(urn, "urn:us:cfr:"):
		return resolver.resolveCFRURN(urn)
	case strings.HasPrefix(urn, "urn:us:state:"):
		return resolver.resolveStateURN(urn)
	default:
		return nil, fmt.Errorf("unsupported URN scheme: %s", urn)
	}
}

// registerUSPatterns registers citation patterns for US law.
func (resolver *SourceResolver) registerUSPatterns() {
	// USC citation: "42 U.S.C. § 1320d" or "42 USC § 1320d" or "42 U.S.C. 1320d"
	resolver.citationPatterns = append(resolver.citationPatterns, &citationPattern{
		name:    "USC",
		pattern: regexp.MustCompile(`(?i)(\d+)\s+U\.?S\.?C\.?\s*§?\s*(\d+\w*(?:-\d+\w*)?)`),
		resolve: func(matches []string) (*ResolvedSource, error) {
			title := matches[1]
			section := matches[2]
			uscURI := uscode.USCURI{Title: title, Section: section}
			documentID := fmt.Sprintf("us-usc-%s-%s", title, section)
			return &ResolvedSource{
				URL:        uscURI.String(),
				DocumentID: strings.ToLower(documentID),
				Domain:     "uscode.house.gov",
				SourceName: "US Code",
			}, nil
		},
	})

	// CFR citation: "45 C.F.R. Part 164" or "45 CFR § 164.502"
	resolver.citationPatterns = append(resolver.citationPatterns, &citationPattern{
		name:    "CFR",
		pattern: regexp.MustCompile(`(?i)(\d+)\s+C\.?F\.?R\.?\s*(?:Part|§|Sec\.?)\s*(\d+)(?:\.(\d+\w*))?`),
		resolve: func(matches []string) (*ResolvedSource, error) {
			title := matches[1]
			part := matches[2]
			section := ""
			if len(matches) > 3 {
				section = matches[3]
			}
			cfrURI := uscode.CFRURI{Title: title, Part: part, Section: section}
			documentID := fmt.Sprintf("us-cfr-%s-%s", title, part)
			if section != "" {
				documentID = fmt.Sprintf("us-cfr-%s-%s-%s", title, part, section)
			}
			return &ResolvedSource{
				URL:        cfrURI.String(),
				DocumentID: strings.ToLower(documentID),
				Domain:     "www.ecfr.gov",
				SourceName: "eCFR",
			}, nil
		},
	})

	// Public Law citation: "Pub. L. 104-191" or "Public Law 104-191"
	resolver.citationPatterns = append(resolver.citationPatterns, &citationPattern{
		name:    "Public Law",
		pattern: regexp.MustCompile(`(?i)(?:Pub(?:lic)?\.?\s*L(?:aw)?\.?)\s*(\d+)-(\d+)`),
		resolve: func(matches []string) (*ResolvedSource, error) {
			congress := matches[1]
			lawNumber := matches[2]
			documentID := fmt.Sprintf("us-pl-%s-%s", congress, lawNumber)
			fetchURL := fmt.Sprintf("https://www.congress.gov/bill/%sth-congress/public-law/%s", congress, lawNumber)
			return &ResolvedSource{
				URL:        fetchURL,
				DocumentID: strings.ToLower(documentID),
				Domain:     "www.congress.gov",
				SourceName: "Congress.gov",
			}, nil
		},
	})

	// California Civil Code: "Cal. Civ. Code § 1798.100"
	resolver.citationPatterns = append(resolver.citationPatterns, &citationPattern{
		name:    "California Code",
		pattern: regexp.MustCompile(`(?i)(?:Cal(?:ifornia)?\.?\s*Civ(?:il)?\.?\s*Code)\s*(?:§|Sec(?:tion)?\.?)\s*(\d+(?:\.\d+)*)`),
		resolve: func(matches []string) (*ResolvedSource, error) {
			section := matches[1]
			documentID := fmt.Sprintf("us-ca-civ-%s", section)
			fetchURL := fmt.Sprintf("https://leginfo.legislature.ca.gov/faces/codes_displaySection.xhtml?sectionNum=%s&lawCode=CIV", section)
			return &ResolvedSource{
				URL:        fetchURL,
				DocumentID: strings.ToLower(documentID),
				Domain:     "leginfo.legislature.ca.gov",
				SourceName: "CA Legislature",
			}, nil
		},
	})

	// Virginia Code: "Va. Code Ann. § 59.1-575"
	resolver.citationPatterns = append(resolver.citationPatterns, &citationPattern{
		name:    "Virginia Code",
		pattern: regexp.MustCompile(`(?i)(?:Va(?:\.|\s+Virginia)?\.?\s*Code(?:\s+Ann\.?)?)\s*(?:§|Sec(?:tion)?\.?)\s*(\d+(?:\.\d+)?(?:-\d+)?)`),
		resolve: func(matches []string) (*ResolvedSource, error) {
			section := matches[1]
			documentID := fmt.Sprintf("us-va-code-%s", section)
			titlePart := strings.Split(section, ".")[0]
			fetchURL := fmt.Sprintf("https://law.lis.virginia.gov/vacode/title%s/", titlePart)
			return &ResolvedSource{
				URL:        fetchURL,
				DocumentID: strings.ToLower(documentID),
				Domain:     "law.lis.virginia.gov",
				SourceName: "Virginia LIS",
			}, nil
		},
	})

	// Colorado Revised Statutes: "C.R.S. § 6-1-1301"
	resolver.citationPatterns = append(resolver.citationPatterns, &citationPattern{
		name:    "Colorado Statutes",
		pattern: regexp.MustCompile(`(?i)(?:C\.?R\.?S\.?|Colorado\s+Revised\s+Statutes?)\s*(?:§|Sec(?:tion)?\.?)?\s*(\d+(?:-\d+)+)`),
		resolve: func(matches []string) (*ResolvedSource, error) {
			section := matches[1]
			documentID := fmt.Sprintf("us-co-crs-%s", section)
			return &ResolvedSource{
				URL:        fmt.Sprintf("https://www.law.cornell.edu/regulations/colorado/%s", section),
				DocumentID: strings.ToLower(documentID),
				Domain:     "www.law.cornell.edu",
				SourceName: "LII (Colorado)",
			}, nil
		},
	})

	// Connecticut General Statutes: "Conn. Gen. Stat. § 42-515"
	resolver.citationPatterns = append(resolver.citationPatterns, &citationPattern{
		name:    "Connecticut Statutes",
		pattern: regexp.MustCompile(`(?i)(?:Conn(?:ecticut)?\.?\s*Gen(?:eral)?\.?\s*Stat(?:utes)?\.?)\s*(?:§|Sec(?:tion)?\.?)\s*(\d+(?:[a-z])?-\d+\w*)`),
		resolve: func(matches []string) (*ResolvedSource, error) {
			section := matches[1]
			documentID := fmt.Sprintf("us-ct-stat-%s", section)
			return &ResolvedSource{
				URL:        fmt.Sprintf("https://www.law.cornell.edu/regulations/connecticut/%s", section),
				DocumentID: strings.ToLower(documentID),
				Domain:     "www.law.cornell.edu",
				SourceName: "LII (Connecticut)",
			}, nil
		},
	})

	// Texas Business & Commerce Code: "Tex. Bus. & Com. Code § 541.001"
	resolver.citationPatterns = append(resolver.citationPatterns, &citationPattern{
		name:    "Texas Code",
		pattern: regexp.MustCompile(`(?i)(?:Tex(?:as)?\.?\s*Bus(?:iness)?\.?\s*(?:&|and)\s*Com(?:merce)?\.?\s*Code)\s*(?:§|Sec(?:tion)?\.?)\s*(\d+\.\d+)`),
		resolve: func(matches []string) (*ResolvedSource, error) {
			section := matches[1]
			documentID := fmt.Sprintf("us-tx-buscom-%s", section)
			return &ResolvedSource{
				URL:        fmt.Sprintf("https://www.law.cornell.edu/regulations/texas/bus-com/%s", section),
				DocumentID: strings.ToLower(documentID),
				Domain:     "www.law.cornell.edu",
				SourceName: "LII (Texas)",
			}, nil
		},
	})

	// LII fallback for USC URL: "law.cornell.edu/uscode/text/{title}/{section}"
	resolver.citationPatterns = append(resolver.citationPatterns, &citationPattern{
		name:    "LII USC URL",
		pattern: regexp.MustCompile(`https?://(?:www\.)?law\.cornell\.edu/uscode/text/(\d+)/(\d+\w*)`),
		resolve: func(matches []string) (*ResolvedSource, error) {
			title := matches[1]
			section := matches[2]
			documentID := fmt.Sprintf("us-usc-%s-%s", title, section)
			return &ResolvedSource{
				URL:        matches[0],
				DocumentID: strings.ToLower(documentID),
				Domain:     "www.law.cornell.edu",
				SourceName: "LII",
			}, nil
		},
	})
}

// resolveUSCURN resolves a urn:us:usc:{title}/{section} URN.
func (resolver *SourceResolver) resolveUSCURN(urn string) (*ResolvedSource, error) {
	suffix := strings.TrimPrefix(urn, "urn:us:usc:")
	parts := strings.SplitN(suffix, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid USC URN format: %s", urn)
	}

	title := parts[0]
	section := parts[1]
	uscURI := uscode.USCURI{Title: title, Section: section}
	documentID := fmt.Sprintf("us-usc-%s-%s", title, section)

	return &ResolvedSource{
		URL:        uscURI.String(),
		DocumentID: strings.ToLower(documentID),
		Domain:     "uscode.house.gov",
		SourceName: "US Code",
		Citation:   urn,
	}, nil
}

// resolveCFRURN resolves a urn:us:cfr:{title}/{part}[/{section}] URN.
func (resolver *SourceResolver) resolveCFRURN(urn string) (*ResolvedSource, error) {
	suffix := strings.TrimPrefix(urn, "urn:us:cfr:")
	parts := strings.SplitN(suffix, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid CFR URN format: %s", urn)
	}

	title := parts[0]
	part := parts[1]
	section := ""
	if len(parts) == 3 {
		section = parts[2]
	}

	cfrURI := uscode.CFRURI{Title: title, Part: part, Section: section}
	documentID := fmt.Sprintf("us-cfr-%s-%s", title, part)
	if section != "" {
		documentID = fmt.Sprintf("us-cfr-%s-%s-%s", title, part, section)
	}

	return &ResolvedSource{
		URL:        cfrURI.String(),
		DocumentID: strings.ToLower(documentID),
		Domain:     "www.ecfr.gov",
		SourceName: "eCFR",
		Citation:   urn,
	}, nil
}

// resolveStateURN resolves a urn:us:state:{state}:{code}:{section} URN.
func (resolver *SourceResolver) resolveStateURN(urn string) (*ResolvedSource, error) {
	suffix := strings.TrimPrefix(urn, "urn:us:state:")
	parts := strings.SplitN(suffix, ":", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid state URN format: %s", urn)
	}

	state := strings.ToLower(parts[0])
	documentID := fmt.Sprintf("us-%s-%s", state, strings.Join(parts[1:], "-"))

	var fetchURL string
	var domain string
	var sourceName string

	switch state {
	case "ca":
		section := parts[len(parts)-1]
		fetchURL = fmt.Sprintf("https://leginfo.legislature.ca.gov/faces/codes_displaySection.xhtml?sectionNum=%s&lawCode=CIV", section)
		domain = "leginfo.legislature.ca.gov"
		sourceName = "CA Legislature"
	case "va":
		fetchURL = fmt.Sprintf("https://law.lis.virginia.gov/vacode/title%s/", parts[1])
		domain = "law.lis.virginia.gov"
		sourceName = "Virginia LIS"
	default:
		fetchURL = fmt.Sprintf("https://www.law.cornell.edu/regulations/%s/%s", state, strings.Join(parts[1:], "/"))
		domain = "www.law.cornell.edu"
		sourceName = fmt.Sprintf("LII (%s)", strings.ToUpper(state))
	}

	return &ResolvedSource{
		URL:        fetchURL,
		DocumentID: strings.ToLower(documentID),
		Domain:     domain,
		SourceName: sourceName,
		Citation:   urn,
	}, nil
}

// ExtractDomainFromURL extracts the hostname from a URL string.
func ExtractDomainFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsedURL.Host
}
