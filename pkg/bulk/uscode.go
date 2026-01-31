package bulk

import (
	"fmt"
	"path/filepath"
	"time"
)

// USCodeSource downloads US Code XML from uscode.house.gov.
// Individual title ZIPs contain USLM XML for each title.
type USCodeSource struct {
	config DownloadConfig
}

// NewUSCodeSource creates a USCodeSource with the given config.
func NewUSCodeSource(config DownloadConfig) *USCodeSource {
	return &USCodeSource{config: config}
}

func (source *USCodeSource) Name() string { return "uscode" }

func (source *USCodeSource) Description() string {
	return "US Code XML from uscode.house.gov (54 titles, USLM format)"
}

// ListDatasets returns all 54 USC titles as downloadable datasets.
func (source *USCodeSource) ListDatasets() ([]Dataset, error) {
	var datasets []Dataset

	for _, titleEntry := range uscTitles {
		titleNumber := titleEntry.Number
		titleName := titleEntry.Name

		downloadURL := fmt.Sprintf("%s/xml_usc%s@%s.zip",
			uscBaseURL, titleNumber, uscReleasePoint)

		datasets = append(datasets, Dataset{
			SourceName:  "uscode",
			Identifier:  fmt.Sprintf("usc-title-%s", titleNumber),
			DisplayName: fmt.Sprintf("Title %s - %s", titleNumber, titleName),
			URL:         downloadURL,
			Format:      "zip",
			Jurisdiction: "US",
		})
	}

	return datasets, nil
}

// DownloadDataset downloads a USC title ZIP to the downloads directory.
func (source *USCodeSource) DownloadDataset(dataset Dataset, downloader *Downloader) (*DownloadResult, error) {
	sourceDir := downloader.SourceDirectory("uscode")
	localPath := filepath.Join(sourceDir, filepath.Base(dataset.URL))

	bytesWritten, skipped, err := downloader.DownloadFile(
		dataset.URL, localPath, PrintDownloadProgress)
	if err != nil {
		return &DownloadResult{
			Dataset: dataset,
			Error:   err.Error(),
		}, err
	}

	// Record in manifest
	if !skipped {
		downloader.Manifest().RecordDownload(&DownloadRecord{
			Identifier:   dataset.Identifier,
			SourceName:   "uscode",
			URL:          dataset.URL,
			LocalPath:    localPath,
			SizeBytes:    bytesWritten,
			DownloadedAt: time.Now(),
		})
		downloader.SaveManifest()
		fmt.Println() // newline after progress bar
	}

	return &DownloadResult{
		Dataset:      dataset,
		LocalPath:    localPath,
		BytesWritten: bytesWritten,
		Skipped:      skipped,
		DownloadedAt: time.Now(),
	}, nil
}

// uscBaseURL is the base URL for US Code release point downloads.
const uscBaseURL = "https://uscode.house.gov/download/releasepoints/us/pl/" + uscReleasePoint

// uscReleasePoint is the latest known release point.
const uscReleasePoint = "119/73not60"

// uscTitles contains the 54 US Code titles with display names.
var uscTitles = []struct {
	Number string
	Name   string
}{
	{"01", "General Provisions"},
	{"02", "The Congress"},
	{"03", "The President"},
	{"04", "Flag and Seal, Seat of Government, and the States"},
	{"05", "Government Organization and Employees"},
	{"06", "Domestic Security"},
	{"07", "Agriculture"},
	{"08", "Aliens and Nationality"},
	{"09", "Arbitration"},
	{"10", "Armed Forces"},
	{"11", "Bankruptcy"},
	{"12", "Banks and Banking"},
	{"13", "Census"},
	{"14", "Coast Guard"},
	{"15", "Commerce and Trade"},
	{"16", "Conservation"},
	{"17", "Copyrights"},
	{"18", "Crimes and Criminal Procedure"},
	{"19", "Customs Duties"},
	{"20", "Education"},
	{"21", "Food and Drugs"},
	{"22", "Foreign Relations and Intercourse"},
	{"23", "Highways"},
	{"24", "Hospitals and Asylums"},
	{"25", "Indians"},
	{"26", "Internal Revenue Code"},
	{"27", "Intoxicating Liquors"},
	{"28", "Judiciary and Judicial Procedure"},
	{"29", "Labor"},
	{"30", "Mineral Lands and Mining"},
	{"31", "Money and Finance"},
	{"32", "National Guard"},
	{"33", "Navigation and Navigable Waters"},
	{"34", "Crime Control and Law Enforcement"},
	{"35", "Patents"},
	{"36", "Patriotic and National Observances"},
	{"37", "Pay and Allowances of the Uniformed Services"},
	{"38", "Veterans' Benefits"},
	{"39", "Postal Service"},
	{"40", "Public Buildings, Property, and Works"},
	{"41", "Public Contracts"},
	{"42", "The Public Health and Welfare"},
	{"43", "Public Lands"},
	{"44", "Public Printing and Documents"},
	{"45", "Railroads"},
	{"46", "Shipping"},
	{"47", "Telecommunications"},
	{"48", "Territories and Insular Possessions"},
	{"49", "Transportation"},
	{"50", "War and National Defense"},
	{"51", "National and Commercial Space Programs"},
	{"52", "Voting and Elections"},
	{"53", "Reserved"},
	{"54", "National Park Service and Related Programs"},
}
