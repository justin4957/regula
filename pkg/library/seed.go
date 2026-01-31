package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultCorpusEntries returns the hardcoded list of all known testdata documents.
func DefaultCorpusEntries() []CorpusEntry {
	return []CorpusEntry{
		{ID: "eu-gdpr", Jurisdiction: "EU", ShortName: "GDPR", FullName: "Regulation (EU) 2016/679 (General Data Protection Regulation)", Format: "eu", SourcePath: "gdpr.txt", SourceInfo: "Official Journal of the European Union, L 119, 4 May 2016"},
		{ID: "eu-eprivacy", Jurisdiction: "EU", ShortName: "ePrivacy Directive", FullName: "Directive 2002/58/EC (ePrivacy Directive)", Format: "eu", SourcePath: "corpus/eu-eprivacy/source.txt", SourceInfo: "Official Journal of the European Communities, L 201, 31 July 2002"},
		{ID: "eu-ai-act", Jurisdiction: "EU", ShortName: "EU AI Act", FullName: "Regulation (EU) 2024/1689 (Artificial Intelligence Act)", Format: "eu", SourcePath: "eu-ai-act.txt", SourceInfo: "Official Journal of the European Union, 2024"},
		{ID: "eu-dsa", Jurisdiction: "EU", ShortName: "EU DSA", FullName: "Regulation (EU) 2022/2065 (Digital Services Act)", Format: "eu", SourcePath: "eu-dsa.txt", SourceInfo: "Official Journal of the European Union, 2022"},
		{ID: "us-ca-ccpa", Jurisdiction: "US-CA", ShortName: "CCPA", FullName: "California Consumer Privacy Act (Cal. Civ. Code 1798.100 et seq.)", Format: "us", SourcePath: "ccpa.txt", SourceInfo: "California Civil Code, Title 1.81.5"},
		{ID: "us-va-vcdpa", Jurisdiction: "US-VA", ShortName: "VCDPA", FullName: "Virginia Consumer Data Protection Act (Va. Code Ann. 59.1-575 et seq.)", Format: "us", SourcePath: "vcdpa.txt", SourceInfo: "Code of Virginia, Title 59.1, Chapter 53"},
		{ID: "us-co-cpa", Jurisdiction: "US-CO", ShortName: "CPA", FullName: "Colorado Privacy Act (C.R.S. 6-1-1301 et seq.)", Format: "us", SourcePath: "cpa.txt", SourceInfo: "Colorado Revised Statutes, Title 6, Article 1, Part 13"},
		{ID: "us-ct-ctdpa", Jurisdiction: "US-CT", ShortName: "CTDPA", FullName: "Connecticut Data Privacy Act (Conn. Gen. Stat. 42-515 et seq.)", Format: "us", SourcePath: "ctdpa.txt", SourceInfo: "Connecticut General Statutes, Title 42, Chapter 743dd"},
		{ID: "us-ut-ucpa", Jurisdiction: "US-UT", ShortName: "UCPA", FullName: "Utah Consumer Privacy Act (U.C.A. 13-61-101 et seq.)", Format: "us", SourcePath: "ucpa.txt", SourceInfo: "Utah Code Annotated, Title 13, Chapter 61"},
		{ID: "us-tx-tdpsa", Jurisdiction: "US-TX", ShortName: "TDPSA", FullName: "Texas Data Privacy and Security Act (Tex. Bus. & Com. Code 541.001 et seq.)", Format: "us", SourcePath: "tdpsa.txt", SourceInfo: "Texas Business and Commerce Code, Title 11, Subtitle C"},
		{ID: "us-ia-icdpa", Jurisdiction: "US-IA", ShortName: "ICDPA", FullName: "Iowa Consumer Data Protection Act (Iowa Code 715D.1 et seq.)", Format: "us", SourcePath: "icdpa.txt", SourceInfo: "Iowa Code, Chapter 715D"},
		{ID: "us-coppa", Jurisdiction: "US-Federal", ShortName: "COPPA", FullName: "Children's Online Privacy Protection Act (15 U.S.C. 6501-6506)", Format: "us", SourcePath: "us-coppa.txt", SourceInfo: "United States Code, Title 15, Chapter 91"},
		{ID: "us-hipaa", Jurisdiction: "US-Federal", ShortName: "HIPAA", FullName: "Health Insurance Portability and Accountability Act excerpt (42 USC 1320d)", Format: "us", SourcePath: "corpus/us-hipaa/source.txt", SourceInfo: "United States Code, Title 42, Chapter 7, Subchapter XI, Part C"},
		{ID: "us-hipaa-cfr", Jurisdiction: "US-Federal", ShortName: "45 CFR 164", FullName: "Security and Privacy Standards (45 CFR Part 164 excerpt)", Format: "us", SourcePath: "corpus/us-hipaa-cfr/source.txt", SourceInfo: "Code of Federal Regulations, Title 45, Part 164"},
		{ID: "gb-dpa2018", Jurisdiction: "GB", ShortName: "DPA 2018", FullName: "Data Protection Act 2018 (2018 c. 12)", Format: "uk", SourcePath: "uk-dpa2018.txt", SourceInfo: "Acts of Parliament, 2018 c. 12"},
		{ID: "gb-si-example", Jurisdiction: "GB", ShortName: "GDPR SI 2019", FullName: "Data Protection, Privacy and Electronic Communications Regulations 2019 (S.I. 2019/419)", Format: "uk", SourcePath: "uk-si-example.txt", SourceInfo: "Statutory Instruments, S.I. 2019 No. 419"},
		{ID: "intl-uncitral", Jurisdiction: "INTL", ShortName: "UNCITRAL Model Law", FullName: "UNCITRAL Model Law on Electronic Commerce (1996)", Format: "generic", SourcePath: "corpus/intl-uncitral/source.txt", SourceInfo: "United Nations Commission on International Trade Law, 1996"},
		{ID: "au-privacy", Jurisdiction: "AU", ShortName: "Privacy Act 1988", FullName: "Privacy Act 1988 (Cth) excerpt", Format: "generic", SourcePath: "corpus/au-privacy/source.txt", SourceInfo: "Commonwealth of Australia, Act No. 119 of 1988"},
	}
}

// SeedFromCorpus ingests all entries from the provided corpus list, resolving
// source paths relative to testdataDir.
func SeedFromCorpus(lib *Library, testdataDir string, entries []CorpusEntry) (*SeedReport, error) {
	seedReport := &SeedReport{
		TotalAttempted: len(entries),
		Entries:        make([]SeedEntryState, 0, len(entries)),
	}

	for _, corpusEntry := range entries {
		// Check if already ingested
		if existing := lib.GetDocument(corpusEntry.ID); existing != nil && existing.Status == StatusReady {
			seedReport.Skipped++
			seedReport.Entries = append(seedReport.Entries, SeedEntryState{
				ID:     corpusEntry.ID,
				Status: "skipped",
			})
			continue
		}

		sourcePath := filepath.Join(testdataDir, corpusEntry.SourcePath)
		sourceText, err := os.ReadFile(sourcePath)
		if err != nil {
			seedReport.Failed++
			seedReport.Entries = append(seedReport.Entries, SeedEntryState{
				ID:     corpusEntry.ID,
				Status: "failed",
				Error:  fmt.Sprintf("failed to read source: %v", err),
			})
			continue
		}

		opts := AddOptions{
			Name:         corpusEntry.ShortName,
			ShortName:    corpusEntry.ShortName,
			FullName:     corpusEntry.FullName,
			Jurisdiction: corpusEntry.Jurisdiction,
			Format:       corpusEntry.Format,
			SourceInfo:   corpusEntry.SourceInfo,
			Force:        true,
		}

		_, err = lib.AddDocument(corpusEntry.ID, sourceText, opts)
		if err != nil {
			seedReport.Failed++
			seedReport.Entries = append(seedReport.Entries, SeedEntryState{
				ID:     corpusEntry.ID,
				Status: "failed",
				Error:  err.Error(),
			})
			continue
		}

		seedReport.Succeeded++
		seedReport.Entries = append(seedReport.Entries, SeedEntryState{
			ID:     corpusEntry.ID,
			Status: "ingested",
		})
	}

	return seedReport, nil
}

// SeedFromDirectory scans a directory for .txt files and ingests each one.
func SeedFromDirectory(lib *Library, dirPath string) (*SeedReport, error) {
	matches, err := filepath.Glob(filepath.Join(dirPath, "*.txt"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob directory: %w", err)
	}

	seedReport := &SeedReport{
		TotalAttempted: len(matches),
		Entries:        make([]SeedEntryState, 0, len(matches)),
	}

	for _, sourcePath := range matches {
		documentID := DeriveDocumentID(sourcePath)

		// Check if already ingested
		if existing := lib.GetDocument(documentID); existing != nil && existing.Status == StatusReady {
			seedReport.Skipped++
			seedReport.Entries = append(seedReport.Entries, SeedEntryState{
				ID:     documentID,
				Status: "skipped",
			})
			continue
		}

		sourceText, err := os.ReadFile(sourcePath)
		if err != nil {
			seedReport.Failed++
			seedReport.Entries = append(seedReport.Entries, SeedEntryState{
				ID:     documentID,
				Status: "failed",
				Error:  err.Error(),
			})
			continue
		}

		opts := AddOptions{
			Name:      documentID,
			ShortName: documentID,
			Force:     true,
		}

		_, err = lib.AddDocument(documentID, sourceText, opts)
		if err != nil {
			seedReport.Failed++
			seedReport.Entries = append(seedReport.Entries, SeedEntryState{
				ID:     documentID,
				Status: "failed",
				Error:  err.Error(),
			})
			continue
		}

		seedReport.Succeeded++
		seedReport.Entries = append(seedReport.Entries, SeedEntryState{
			ID:     documentID,
			Status: "ingested",
		})
	}

	return seedReport, nil
}

// LoadCorpusManifest reads the corpus manifest.json and returns entries.
func LoadCorpusManifest(manifestPath string) ([]CorpusEntry, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read corpus manifest: %w", err)
	}

	var corpusManifest struct {
		Entries []struct {
			ID           string `json:"id"`
			Jurisdiction string `json:"jurisdiction"`
			ShortName    string `json:"short_name"`
			FullName     string `json:"full_name"`
			Format       string `json:"format"`
			SourcePath   string `json:"source_path"`
			SourceInfo   string `json:"source_info"`
		} `json:"entries"`
	}

	if err := json.Unmarshal(data, &corpusManifest); err != nil {
		return nil, fmt.Errorf("failed to parse corpus manifest: %w", err)
	}

	entries := make([]CorpusEntry, len(corpusManifest.Entries))
	for i, manifestEntry := range corpusManifest.Entries {
		entries[i] = CorpusEntry{
			ID:           manifestEntry.ID,
			Jurisdiction: manifestEntry.Jurisdiction,
			ShortName:    manifestEntry.ShortName,
			FullName:     manifestEntry.FullName,
			Format:       manifestEntry.Format,
			SourcePath:   manifestEntry.SourcePath,
			SourceInfo:   manifestEntry.SourceInfo,
		}
	}

	return entries, nil
}
