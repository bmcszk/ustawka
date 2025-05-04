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

// baseURL is the base URL for the Sejm API
var baseURL = "https://api.sejm.gov.pl/eli"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

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

type ActDetails struct {
	ID        string `json:"ELI"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Published string `json:"promulgation"`
}

type APIResponse struct {
	Items      []Act `json:"items"`
	Offset     int   `json:"offset"`
	TotalCount int   `json:"totalCount"`
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    baseURL,
	}
}

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	slog.Debug("Successfully fetched acts", "year", year, "count", len(apiResponse.Items))
	return apiResponse.Items, nil
}

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
	defer resp.Body.Close()

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
