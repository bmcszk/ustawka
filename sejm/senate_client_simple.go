//revive:disable:max-public-structs
package sejm

import (
	"context"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// SenateClient interface for Senate data access
type SenateClient interface {
	GetManifest(ctx context.Context) (*SenateManifest, error)
	GetLatestVotingFiles(manifest *SenateManifest) (*SenateDataFileInfo, *SenateDataFileInfo)
	GetIndividualVotingData(ctx context.Context, fileURL string) ([]SenateVotingRecord, error)
	GetClubVotingData(ctx context.Context, fileURL string) ([]SenateVotingRecord, error)
}

// SimpleSenateClient provides basic access to Senate data
type SimpleSenateClient struct {
	httpClient *http.Client
	baseURL    string
}

// SenateManifest represents Senate data manifest
type SenateManifest struct {
	XMLName xml.Name              `xml:"manifest"`
	Files   []SenateDataFileInfo `xml:"file"`
}

// SenateDataFileInfo represents Senate file info
type SenateDataFileInfo struct {
	Name         string `xml:"name,attr"`
	URL          string `xml:"url,attr"`
	LastModified string `xml:"lastModified,attr"`
}

// SenateVotingRecord represents a Senate voting record
type SenateVotingRecord struct {
	ActID       string
	VotingDate  time.Time
	Subject     string
	Result      string
	VotesFor    int
	VotesAgainst int
	VotesAbstain int
}

// BasicSenateManifest represents a simplified manifest
type BasicSenateManifest struct {
	XMLName xml.Name                   `xml:"manifest"`
	Files   []BasicSenateDataFileInfo `xml:"file"`
}

// BasicSenateDataFileInfo represents basic file info
type BasicSenateDataFileInfo struct {
	Name         string `xml:"name,attr"`
	URL          string `xml:"url,attr"`
	LastModified string `xml:"lastModified,attr"`
}

// NewSimpleSenateClient creates a simplified Senate client
func NewSimpleSenateClient() *SimpleSenateClient {
	return &SimpleSenateClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.dane.gov.pl/1.4/datasets/4648,glosowania-senatu",
	}
}

// GetBasicManifest retrieves a basic manifest
func (c *SimpleSenateClient) GetBasicManifest(ctx context.Context) (*BasicSenateManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching manifest: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Error closing response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var manifest BasicSenateManifest
	if err := xml.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest XML: %w", err)
	}

	return &manifest, nil
}

// GetBasicCSVData fetches basic CSV data
func (c *SimpleSenateClient) GetBasicCSVData(ctx context.Context, fileURL string) ([][]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching data: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Error closing response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	return c.parseBasicCSV(resp.Body), nil
}

// parseBasicCSV parses CSV with basic error handling
func (*SimpleSenateClient) parseBasicCSV(reader io.Reader) [][]string {
	csvReader := csv.NewReader(reader)
	csvReader.LazyQuotes = true

	var records [][]string
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip invalid records
		}
		records = append(records, record)
	}

	return records
}

// Implement SenateClient interface

// GetManifest retrieves the Senate data manifest
func (c *SimpleSenateClient) GetManifest(ctx context.Context) (*SenateManifest, error) {
	basic, err := c.GetBasicManifest(ctx)
	if err != nil {
		return nil, err
	}
	
	// Convert BasicSenateManifest to SenateManifest
	manifest := &SenateManifest{
		XMLName: basic.XMLName,
		Files:   make([]SenateDataFileInfo, len(basic.Files)),
	}
	
	for i, file := range basic.Files {
		manifest.Files[i] = SenateDataFileInfo(file)
	}
	
	return manifest, nil
}

// GetLatestVotingFiles finds the latest individual and club voting files
func (*SimpleSenateClient) GetLatestVotingFiles(manifest *SenateManifest) (
	individualFile, clubFile *SenateDataFileInfo) {
	for i := range manifest.Files {
		file := &manifest.Files[i]
		switch file.Name {
		case "individual_votes.csv":
			individualFile = file
		case "club_votes.csv":
			clubFile = file
		}
	}
	
	return individualFile, clubFile
}

// GetIndividualVotingData retrieves individual voting data
func (c *SimpleSenateClient) GetIndividualVotingData(ctx context.Context,
	fileURL string) ([]SenateVotingRecord, error) {
	records, err := c.GetBasicCSVData(ctx, fileURL)
	if err != nil {
		return nil, err
	}
	
	// Convert CSV records to SenateVotingRecord structs
	var votingRecords []SenateVotingRecord
	for _, record := range records {
		if len(record) >= 3 {
			votingRecord := SenateVotingRecord{
				ActID:   record[0],
				Subject: record[1],
				Result:  record[2],
			}
			votingRecords = append(votingRecords, votingRecord)
		}
	}
	
	return votingRecords, nil
}

// GetClubVotingData retrieves club voting data  
func (c *SimpleSenateClient) GetClubVotingData(ctx context.Context, fileURL string) ([]SenateVotingRecord, error) {
	records, err := c.GetBasicCSVData(ctx, fileURL)
	if err != nil {
		return nil, err
	}
	
	// Convert CSV records to SenateVotingRecord structs
	var votingRecords []SenateVotingRecord
	for _, record := range records {
		if len(record) >= 3 {
			votingRecord := SenateVotingRecord{
				ActID:   record[0],
				Subject: record[1],
				Result:  record[2],
			}
			votingRecords = append(votingRecords, votingRecord)
		}
	}
	
	return votingRecords, nil
}