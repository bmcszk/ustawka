package sejm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// baseURL is the base URL for the ELI API
var baseURL = "https://api.sejm.gov.pl/eli"

// sejmAPIBaseURL is the base URL for the main Sejm API
var sejmAPIBaseURL = "https://api.sejm.gov.pl/sejm"

// Client provides access to the Sejm API
type Client struct {
	httpClient      *http.Client
	baseURL         string
	sejmAPIBaseURL  string
}

// Act represents basic information about a legislative act
type Act struct {
	ID        string `json:"ELI"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Published string `json:"promulgation"`
	Position  int    `json:"pos"`
	Year      int    `json:"year"`
	Type      string `json:"type"`
	Address   string `json:"address"`
}

// ActDetails contains comprehensive information about a legislative act
type ActDetails struct {
	ID               string      `json:"ELI"`
	Title            string      `json:"title"`
	Status           string      `json:"status"`
	Published        string      `json:"promulgation"`
	Type             string      `json:"type"`
	Address          string      `json:"address"`
	DisplayAddress   string      `json:"displayAddress"`
	Position         int         `json:"pos"`
	Year             int         `json:"year"`
	AnnouncementDate string      `json:"announcementDate"`
	ChangeDate       string      `json:"changeDate"`
	Publisher        string      `json:"publisher"`
	TextHTML         bool        `json:"textHTML"`
	TextPDF          bool        `json:"textPDF"`
	Volume           int         `json:"volume"`
	EntryIntoForce   string      `json:"entryIntoForce"`
	InForce          string      `json:"inForce"`
	Keywords         []string    `json:"keywords"`
	KeywordsNames    []string    `json:"keywordsNames"`
	ReleasedBy       []string    `json:"releasedBy"`
	Texts            []Text      `json:"texts"`
	References       References  `json:"references"`
	AuthorizedBody   []string    `json:"authorizedBody"`
	Directives       any `json:"directives"`
	Obligated        []string    `json:"obligated"`
	PreviousTitle    []string    `json:"previousTitle"`
	Prints           any `json:"prints"`
}

// Text represents a text version of an act
// Type can be:
//   - O: Original text (Tekst oryginalny)
//   - I: Consolidated text (Tekst ujednolicony)
//   - T: Translation (Tłumaczenie)
//   - U: Unofficial translation (Tłumaczenie nieoficjalne)
type Text struct {
	FileName string `json:"fileName"`
	Type     string `json:"type"`
}

// References contains legal references to related acts
type References struct {
	RepealedActs        []reference `json:"Akty uznane za uchylone"`
	AmendingActs        []reference `json:"Akty zmieniające"`
	LegalBasis          []reference `json:"Podstawa prawna"`
	LegalBasisWithArt   []reference `json:"Podstawa prawna z art."`
	TekstJednolity      []reference `json:"Tekst jednolity dla aktu"`
	InfOTekstJednolitym []reference `json:"Inf. o tekście jednolitym"`
}

type reference struct {
	ID   string `json:"id"`
	Date string `json:"date,omitempty"`
	Art  string `json:"art,omitempty"`
}


type apiResponse struct {
	Items      []Act `json:"items"`
	Offset     int   `json:"offset"`
	TotalCount int   `json:"totalCount"`
}

// NewClient creates a new Sejm API client
func NewClient() *Client {
	return &Client{
		httpClient:     &http.Client{},
		baseURL:        baseURL,
		sejmAPIBaseURL: sejmAPIBaseURL,
	}
}

// NewClientWithURL creates a new client with a custom base URL (primarily for testing)
func NewClientWithURL(baseURL string) *Client {
	return &Client{
		httpClient:     &http.Client{},
		baseURL:        baseURL,
		sejmAPIBaseURL: sejmAPIBaseURL,
	}
}

// GetActs retrieves all acts for a specific year
func (c *Client) GetActs(ctx context.Context, year int) ([]Act, error) {
	url := fmt.Sprintf("%s/acts/DU/%d", c.baseURL, year)
	slog.Debug("Fetching acts", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching acts: %w", err)
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

	var apiResponse apiResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	slog.Debug("Successfully fetched acts", "year", year, "count", len(apiResponse.Items))
	return apiResponse.Items, nil
}

// GetActDetails retrieves detailed information about a specific act
func (c *Client) GetActDetails(ctx context.Context, id string) (*ActDetails, error) {
	url := fmt.Sprintf("%s/acts/%s", c.baseURL, id)
	slog.Debug("Fetching act details", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching act details: %w", err)
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

	var details ActDetails
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, fmt.Errorf("failed to parse act details: %v", err)
	}

	slog.Debug("Successfully fetched act details", "id", id)
	return &details, nil
}

// GetYearString returns the year as a string
func (a *Act) GetYearString() string {
	return strconv.Itoa(a.Year)
}

// GetProcessInfo retrieves process information for a specific bill
func (c *Client) GetProcessInfo(ctx context.Context, term int, printNumber string) (*ProcessInfo, error) {
	url := fmt.Sprintf("%s/term%d/processes/%s", c.sejmAPIBaseURL, term, printNumber)
	slog.Debug("Fetching process info", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching process info: %w", err)
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

	var processInfo ProcessInfo
	if err := json.Unmarshal(body, &processInfo); err != nil {
		return nil, fmt.Errorf("failed to parse process info: %v", err)
	}

	slog.Debug("Successfully fetched process info", "term", term, "printNumber", printNumber)
	return &processInfo, nil
}

// GetVotingInfo retrieves voting information for a specific proceeding and vote
func (c *Client) GetVotingInfo(ctx context.Context, term int, proceeding int, voting int) (*VotingInfo, error) {
	url := fmt.Sprintf("%s/term%d/votings/%d/%d", c.sejmAPIBaseURL, term, proceeding, voting)
	slog.Debug("Fetching voting info", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching voting info: %w", err)
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

	var votingInfo VotingInfo
	if err := json.Unmarshal(body, &votingInfo); err != nil {
		return nil, fmt.Errorf("failed to parse voting info: %v", err)
	}

	slog.Debug("Successfully fetched voting info", "term", term, "proceeding", proceeding, "voting", voting)
	return &votingInfo, nil
}

// GetPrints retrieves prints (bills) for a specific term
func (c *Client) GetPrints(ctx context.Context, term int) ([]any, error) {
	url := fmt.Sprintf("%s/term%d/prints", c.sejmAPIBaseURL, term)
	slog.Debug("Fetching prints", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching prints: %w", err)
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

	var prints []any
	if err := json.Unmarshal(body, &prints); err != nil {
		return nil, fmt.Errorf("failed to parse prints: %v", err)
	}

	slog.Debug("Successfully fetched prints", "term", term, "count", len(prints))
	return prints, nil
}

// GetVotings retrieves all votings for a specific term and proceeding
func (c *Client) GetVotings(ctx context.Context, term int, proceeding int) ([]any, error) {
	url := fmt.Sprintf("%s/term%d/votings/%d", c.sejmAPIBaseURL, term, proceeding)
	slog.Debug("Fetching votings", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching votings: %w", err)
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

	var votings []any
	if err := json.Unmarshal(body, &votings); err != nil {
		return nil, fmt.Errorf("failed to parse votings: %v", err)
	}

	slog.Debug("Successfully fetched votings", "term", term, "proceeding", proceeding, "count", len(votings))
	return votings, nil
}
