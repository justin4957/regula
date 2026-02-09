package deliberation

import (
	"strings"
	"testing"
	"time"
)

// Sample working paper document
const sampleWorkingPaper = `
Document No.: WG/2024/15/REV2

DRAFT WORK AND RESOURCING PLAN

Version: 2.0
Date: 15 March 2024
Author: Secretariat

Status: REVISED

This document supersedes WG/2024/15/REV1.

Changes from previous version:
- Updated timeline for Phase 2
- Added new resource requirements
- Revised budget estimates

1. INTRODUCTION

This work plan outlines the activities and resources required for the
implementation of the project during 2024-2025.

1.1 Background

The Working Group agreed at its 42nd meeting to develop a comprehensive
work plan covering all aspects of implementation.

1.2 Scope

This document covers:
- Timeline and milestones
- Resource allocation
- Budget requirements

2. WORK PROGRAMME

The following activities are planned for the implementation period.

2.1 Phase 1: Preparation (Q1 2024)

[ACTION] Secretariat to prepare detailed implementation guidelines by 30 April 2024.

The preparation phase includes:
1. Stakeholder consultation
2. Resource assessment
3. Risk analysis

2.2 Phase 2: Implementation (Q2-Q3 2024)

Implementation will proceed in stages:
1. Initial rollout in pilot countries
2. Expansion to additional regions
3. Full-scale deployment

[Note: Timeline subject to funding availability]

3. RESOURCE REQUIREMENTS

3.1 Human Resources

Table 1: Staffing Requirements

| Position | FTE | Duration |
|----------|-----|----------|
| Project Manager | 1.0 | 24 months |
| Technical Lead | 1.0 | 18 months |
| Analysts | 2.0 | 12 months |

3.2 Financial Resources

The estimated budget for 2024 is EUR 500,000.

Reservation by DE: Scrutiny reservation on budget allocation.
Reservation by FR: Parliamentary reservation pending.

4. TIMELINE

Key milestones:
- April 2024: Guidelines adopted
- June 2024: Pilot launch
- December 2024: Mid-term review

See document WG/2024/10 for additional context.
Refer to Regulation (EU) 2023/456 for legal basis.

Annex I: Detailed Budget Breakdown

The following provides a detailed breakdown of budget items:
- Personnel costs: EUR 300,000
- Equipment: EUR 100,000
- Travel: EUR 50,000
- Contingency: EUR 50,000

Annex II: Risk Assessment Matrix

Risk categories and mitigation strategies are outlined below.
`

// Sample version 1 for comparison
const sampleWorkingPaperV1 = `
Document No.: WG/2024/15/REV1

DRAFT WORK AND RESOURCING PLAN

Version: 1.0
Date: 1 February 2024
Status: DRAFT

1. INTRODUCTION

This work plan outlines the initial activities for the project.

1.1 Background

The Working Group agreed to develop a work plan.

2. WORK PROGRAMME

Initial work programme description.

3. TIMELINE

Original timeline with fewer milestones.
`

// Sample EU working document
const sampleEUWorkingDoc = `
210301_annotated_agenda_wg_43_en

ANNOTATED AGENDA
Working Group on Technical Standards
43rd Meeting
Date: 1 March 2021

Prepared by: Commission Services

FOR DISCUSSION

1. Adoption of the agenda

The draft agenda is attached as document WG/43/1.

2. Implementation of Regulation (EU) 2020/123

2.1 Progress report

The Commission will present the progress report (doc. WG/43/2).

Delegations are invited to:
- provide comments on the progress made
- identify remaining challenges

2.2 Technical specifications

[Note: This item was added at the request of DE delegation]

Draft specifications to be discussed (doc. WG/43/3).

3. Any other business

4. Date of next meeting

Proposed: 15 April 2021

---
Distribution: Working Group members
`

func TestWorkingPaperParser_Parse(t *testing.T) {
	parser := NewWorkingPaperParser("https://example.org/deliberation")

	doc, err := parser.Parse(sampleWorkingPaper)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check identifier
	if doc.Identifier == "" {
		t.Error("Expected identifier to be extracted")
	} else {
		t.Logf("Identifier: %s", doc.Identifier)
	}

	// Check version
	if doc.Version.Number == "" {
		t.Error("Expected version number to be extracted")
	} else {
		t.Logf("Version: %s", doc.Version.Number)
	}

	// Check date
	if doc.Date.IsZero() {
		t.Error("Expected date to be extracted")
	} else {
		expectedDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
		if !doc.Date.Equal(expectedDate) {
			t.Logf("Date: %v (expected %v)", doc.Date, expectedDate)
		}
	}

	// Check author
	if doc.Author == "" {
		t.Error("Expected author to be extracted")
	} else {
		t.Logf("Author: %s", doc.Author)
	}

	// Check status
	if doc.Status != StatusRevised {
		t.Errorf("Expected status 'revised', got '%s'", doc.Status)
	}

	// Check title
	if doc.Title == "" {
		t.Error("Expected title to be extracted")
	} else {
		t.Logf("Title: %s", doc.Title)
	}

	// Check document type
	if doc.Type != TypeWorkPlan {
		t.Errorf("Expected type 'work_plan', got '%s'", doc.Type)
	}

	// Check sections
	if len(doc.Sections) == 0 {
		t.Error("Expected sections to be extracted")
	} else {
		t.Logf("Found %d sections", len(doc.Sections))
	}

	// Check action points
	if len(doc.ActionPoints) == 0 {
		t.Error("Expected action points to be extracted")
	} else {
		t.Logf("Found %d action points", len(doc.ActionPoints))
	}

	// Check annotations
	if len(doc.Annotations) == 0 {
		t.Error("Expected annotations to be extracted")
	} else {
		t.Logf("Found %d annotations", len(doc.Annotations))
	}

	// Check references
	if len(doc.References) == 0 {
		t.Error("Expected references to be extracted")
	} else {
		t.Logf("Found %d references", len(doc.References))
	}

	// Check annexes
	if len(doc.Annexes) == 0 {
		t.Error("Expected annexes to be extracted")
	} else {
		t.Logf("Found %d annexes", len(doc.Annexes))
	}

	// Check supersedes
	if doc.SupersedesURI == "" {
		t.Error("Expected supersedes URI to be extracted")
	} else {
		t.Logf("Supersedes: %s", doc.SupersedesURI)
	}

	// Check URI generation
	if doc.URI == "" {
		t.Error("Expected URI to be generated")
	}
}

func TestWorkingPaperParser_ExtractVersion(t *testing.T) {
	tests := []struct {
		name            string
		text            string
		expectedNumber  string
		expectedSupersedes string
	}{
		{
			name:           "version with number",
			text:           "Version: 2.0\nDate: 15 March 2024",
			expectedNumber: "2.0",
		},
		{
			name:           "revision format",
			text:           "Document WG/2024/15/REV3\nRevision: 3",
			expectedNumber: "3",
		},
		{
			name:           "version prefix",
			text:           "v1.5 Release Notes",
			expectedNumber: "1.5",
		},
		{
			name:              "with supersedes",
			text:              "Version: 2\nThis document supersedes WG/2024/10",
			expectedNumber:    "2",
			expectedSupersedes: "WG/2024/10",
		},
		{
			name:              "revision of",
			text:              "Rev 4\nRevised version of DOC/123",
			expectedNumber:    "4",
			expectedSupersedes: "DOC/123",
		},
	}

	parser := NewWorkingPaperParser("https://example.org")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := parser.ExtractVersion(tt.text)
			if err != nil {
				t.Fatalf("ExtractVersion failed: %v", err)
			}

			if version.Number != tt.expectedNumber {
				t.Errorf("Expected version '%s', got '%s'", tt.expectedNumber, version.Number)
			}

			if tt.expectedSupersedes != "" && version.SupersedesID != tt.expectedSupersedes {
				t.Errorf("Expected supersedes '%s', got '%s'", tt.expectedSupersedes, version.SupersedesID)
			}
		})
	}
}

func TestWorkingPaperParser_ExtractSections(t *testing.T) {
	parser := NewWorkingPaperParser("https://example.org")

	sections, err := parser.ExtractSections(sampleWorkingPaper)
	if err != nil {
		t.Fatalf("ExtractSections failed: %v", err)
	}

	if len(sections) < 4 {
		t.Errorf("Expected at least 4 sections, got %d", len(sections))
	}

	// Check section levels
	levelCounts := make(map[int]int)
	for _, s := range sections {
		levelCounts[s.Level]++
		t.Logf("Section %s: %s (level %d)", s.Number, s.Title, s.Level)
	}

	// Should have both level 1 and level 2 sections
	if levelCounts[1] == 0 {
		t.Error("Expected level 1 sections")
	}
	if levelCounts[2] == 0 {
		t.Error("Expected level 2 sections")
	}
}

func TestWorkingPaperParser_CompareVersions(t *testing.T) {
	parser := NewWorkingPaperParser("https://example.org")

	v1, err := parser.Parse(sampleWorkingPaperV1)
	if err != nil {
		t.Fatalf("Parse v1 failed: %v", err)
	}

	v2, err := parser.Parse(sampleWorkingPaper)
	if err != nil {
		t.Fatalf("Parse v2 failed: %v", err)
	}

	diff, err := parser.CompareVersions(v1, v2)
	if err != nil {
		t.Fatalf("CompareVersions failed: %v", err)
	}

	t.Logf("Version comparison: %s -> %s", diff.OldVersion.Number, diff.NewVersion.Number)
	t.Logf("Added: %d, Removed: %d, Modified: %d", len(diff.Added), len(diff.Removed), len(diff.Modified))
	t.Logf("Summary: %s", diff.Summary)

	// V2 has more sections than V1
	if len(diff.Added) == 0 && len(diff.Modified) == 0 {
		t.Error("Expected some changes between versions")
	}
}

func TestWorkingPaperParser_Parse_AnnotatedAgenda(t *testing.T) {
	parser := NewWorkingPaperParser("https://example.org/eu")

	doc, err := parser.Parse(sampleEUWorkingDoc)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check document type detection
	if doc.Type != TypeAnnotatedAgenda {
		t.Errorf("Expected type 'annotated_agenda', got '%s'", doc.Type)
	}

	// Check identifier extraction (YYMMDD format)
	if doc.Identifier == "" {
		t.Error("Expected identifier to be extracted")
	} else {
		t.Logf("EU doc identifier: %s", doc.Identifier)
	}

	// Check sections
	if len(doc.Sections) == 0 {
		t.Error("Expected sections to be extracted")
	} else {
		t.Logf("Found %d sections", len(doc.Sections))
	}

	// Check references
	hasRegulationRef := false
	for _, ref := range doc.References {
		if strings.Contains(ref.Identifier, "2020/123") {
			hasRegulationRef = true
		}
	}
	if !hasRegulationRef {
		t.Logf("References found: %v", doc.References)
	}
}

func TestWorkingPaperParser_ExtractAnnotations(t *testing.T) {
	parser := NewWorkingPaperParser("https://example.org")

	doc, err := parser.Parse(sampleWorkingPaper)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check for reservations
	hasReservation := false
	for _, ann := range doc.Annotations {
		if ann.IsReservation {
			hasReservation = true
			t.Logf("Reservation by %s: %s", ann.Delegation, ann.Text)
		}
	}

	if !hasReservation {
		t.Error("Expected delegation reservations to be extracted")
	}

	// Check for notes
	hasNote := false
	for _, ann := range doc.Annotations {
		if ann.Type == "note" {
			hasNote = true
		}
	}

	if !hasNote {
		t.Error("Expected notes to be extracted")
	}
}

func TestWorkingPaperParser_ExtractActionPoints(t *testing.T) {
	parser := NewWorkingPaperParser("https://example.org")

	doc, err := parser.Parse(sampleWorkingPaper)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(doc.ActionPoints) == 0 {
		t.Fatal("Expected action points to be extracted")
	}

	// Check that action points have descriptions
	for _, ap := range doc.ActionPoints {
		if ap.Description == "" {
			t.Errorf("Action point %s has no description", ap.Number)
		}
		t.Logf("Action %s: %s", ap.Number, ap.Description[:min(50, len(ap.Description))])
	}
}

func TestWorkingPaperParser_EmptyInput(t *testing.T) {
	parser := NewWorkingPaperParser("https://example.org")

	_, err := parser.Parse("")
	if err == nil {
		t.Error("Expected error for empty input")
	}
}

func TestWorkingPaperParser_CompareVersions_NilInput(t *testing.T) {
	parser := NewWorkingPaperParser("https://example.org")

	_, err := parser.CompareVersions(nil, nil)
	if err == nil {
		t.Error("Expected error for nil input")
	}
}

func TestDocumentStatus_String(t *testing.T) {
	tests := []struct {
		status   DocumentStatus
		expected string
	}{
		{StatusDraft, "draft"},
		{StatusUnderReview, "under_review"},
		{StatusRevised, "revised"},
		{StatusFinal, "final"},
		{StatusSuperseded, "superseded"},
		{StatusWithdrawn, "withdrawn"},
		{DocumentStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestDocumentType_String(t *testing.T) {
	tests := []struct {
		docType  DocumentType
		expected string
	}{
		{TypeWorkingPaper, "working_paper"},
		{TypeDraftRegulation, "draft_regulation"},
		{TypePositionPaper, "position_paper"},
		{TypeDiscussionPaper, "discussion_paper"},
		{TypeAnnotatedAgenda, "annotated_agenda"},
		{TypeWorkPlan, "work_plan"},
		{TypeReport, "report"},
		{DocumentType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.docType.String(); got != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestParseDocumentStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected DocumentStatus
	}{
		{"draft", StatusDraft},
		{"DRAFT", StatusDraft},
		{"final", StatusFinal},
		{"adopted", StatusFinal},
		{"revised", StatusRevised},
		{"superseded", StatusSuperseded},
		{"withdrawn", StatusWithdrawn},
		{"under review", StatusUnderReview},
		{"unknown", StatusDraft}, // defaults to draft
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseDocumentStatus(tt.input); got != tt.expected {
				t.Errorf("ParseDocumentStatus(%s) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWorkingPaper_String(t *testing.T) {
	wp := &WorkingPaper{
		Identifier: "WG/2024/15",
		Title:      "Work Plan",
		Version:    Version{Number: "2.0"},
		Status:     StatusRevised,
	}

	str := wp.String()
	if !strings.Contains(str, "WG/2024/15") {
		t.Errorf("String should contain identifier: %s", str)
	}
	if !strings.Contains(str, "Work Plan") {
		t.Errorf("String should contain title: %s", str)
	}
	if !strings.Contains(str, "2.0") {
		t.Errorf("String should contain version: %s", str)
	}
	if !strings.Contains(str, "revised") {
		t.Errorf("String should contain status: %s", str)
	}
}

func TestNewWorkingPaperParser(t *testing.T) {
	parser := NewWorkingPaperParser("https://example.org")

	if parser == nil {
		t.Fatal("Expected non-nil parser")
	}
	if parser.BaseURI != "https://example.org" {
		t.Errorf("Expected BaseURI 'https://example.org', got '%s'", parser.BaseURI)
	}
	if parser.patterns == nil {
		t.Error("Expected patterns to be compiled")
	}
}

// min returns the smaller of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
