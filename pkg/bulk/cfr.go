package bulk

import (
	"fmt"
	"path/filepath"
	"time"
)

// CFRSource downloads Code of Federal Regulations XML from govinfo.gov.
// Bulk XML is available per-title as ZIP archives.
type CFRSource struct {
	config DownloadConfig
	year   string
}

// NewCFRSource creates a CFRSource targeting the configured year.
func NewCFRSource(config DownloadConfig) *CFRSource {
	year := config.CFRYear
	if year == "" {
		year = "2024"
	}
	return &CFRSource{config: config, year: year}
}

func (source *CFRSource) Name() string { return "cfr" }

func (source *CFRSource) Description() string {
	return fmt.Sprintf("Code of Federal Regulations XML from govinfo.gov (%s edition, 50 titles)", source.year)
}

// ListDatasets returns all 50 CFR titles as downloadable datasets.
func (source *CFRSource) ListDatasets() ([]Dataset, error) {
	var datasets []Dataset

	for _, titleEntry := range cfrTitles {
		titleNumber := titleEntry.Number
		titleName := titleEntry.Name

		downloadURL := fmt.Sprintf("https://www.govinfo.gov/bulkdata/CFR/%s/title-%s/CFR-%s-title-%s.zip",
			source.year, titleNumber, source.year, titleNumber)

		datasets = append(datasets, Dataset{
			SourceName:  "cfr",
			Identifier:  fmt.Sprintf("cfr-%s-title-%s", source.year, titleNumber),
			DisplayName: fmt.Sprintf("Title %s - %s (%s)", titleNumber, titleName, source.year),
			URL:         downloadURL,
			Format:      "zip",
			Jurisdiction: "US",
		})
	}

	return datasets, nil
}

// DownloadDataset downloads a CFR title ZIP to the downloads directory.
func (source *CFRSource) DownloadDataset(dataset Dataset, downloader *Downloader) (*DownloadResult, error) {
	sourceDir := downloader.SourceDirectory("cfr")
	localPath := filepath.Join(sourceDir, filepath.Base(dataset.URL))

	bytesWritten, skipped, err := downloader.DownloadFile(
		dataset.URL, localPath, PrintDownloadProgress)
	if err != nil {
		return &DownloadResult{
			Dataset: dataset,
			Error:   err.Error(),
		}, err
	}

	if !skipped {
		downloader.Manifest().RecordDownload(&DownloadRecord{
			Identifier:   dataset.Identifier,
			SourceName:   "cfr",
			URL:          dataset.URL,
			LocalPath:    localPath,
			SizeBytes:    bytesWritten,
			DownloadedAt: time.Now(),
		})
		downloader.SaveManifest()
		fmt.Println()
	}

	return &DownloadResult{
		Dataset:      dataset,
		LocalPath:    localPath,
		BytesWritten: bytesWritten,
		Skipped:      skipped,
		DownloadedAt: time.Now(),
	}, nil
}

// cfrTitles contains the 50 CFR titles with display names.
var cfrTitles = []struct {
	Number string
	Name   string
}{
	{"1", "General Provisions"},
	{"2", "Grants and Agreements"},
	{"3", "The President"},
	{"4", "Accounts"},
	{"5", "Administrative Personnel"},
	{"6", "Domestic Security"},
	{"7", "Agriculture"},
	{"8", "Aliens and Nationality"},
	{"9", "Animals and Animal Products"},
	{"10", "Energy"},
	{"11", "Federal Elections"},
	{"12", "Banks and Banking"},
	{"13", "Business Credit and Assistance"},
	{"14", "Aeronautics and Space"},
	{"15", "Commerce and Foreign Trade"},
	{"16", "Commercial Practices"},
	{"17", "Commodity and Securities Exchanges"},
	{"18", "Conservation of Power and Water Resources"},
	{"19", "Customs Duties"},
	{"20", "Employees' Benefits"},
	{"21", "Food and Drugs"},
	{"22", "Foreign Relations"},
	{"23", "Highways"},
	{"24", "Housing and Urban Development"},
	{"25", "Indians"},
	{"26", "Internal Revenue"},
	{"27", "Alcohol, Tobacco Products and Firearms"},
	{"28", "Judicial Administration"},
	{"29", "Labor"},
	{"30", "Mineral Resources"},
	{"31", "Money and Finance: Treasury"},
	{"32", "National Defense"},
	{"33", "Navigation and Navigable Waters"},
	{"34", "Education"},
	{"35", "Panama Canal [Reserved]"},
	{"36", "Parks, Forests, and Public Property"},
	{"37", "Patents, Trademarks, and Copyrights"},
	{"38", "Pensions, Bonuses, and Veterans' Relief"},
	{"39", "Postal Service"},
	{"40", "Protection of Environment"},
	{"41", "Public Contracts and Property Management"},
	{"42", "Public Health"},
	{"43", "Public Lands: Interior"},
	{"44", "Emergency Management and Assistance"},
	{"45", "Public Welfare"},
	{"46", "Shipping"},
	{"47", "Telecommunication"},
	{"48", "Federal Acquisition Regulations System"},
	{"49", "Transportation"},
	{"50", "Wildlife and Fisheries"},
}
