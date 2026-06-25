package skills

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const exaApi = "https://api.exa.ai/search"

type Exa struct {
	API_KEY string
	Limit   int
}

type Results struct {
	Results            []ResultBody `json:"results"`
	Output             *Output      `json:"output"`
	RequestId          string       `json:"requestid"`
	ResolvedSearchType string       `json:"resolvedSearchType"`
	Context            string       `json:"context"`
	CostDollars        CostDollars  `json:"costDollars"`
}

type ResultBody struct {
	Title           string    `json:"title"`
	URL             string    `json:"url"`
	PublishedAt     string    `json:"publishedDate"`
	Author          string    `json:"author"`
	ID              string    `json:"id"`
	Image           string    `json:"image"`
	Favicon         string    `json:"favicon"`
	Text            string    `json:"text"`
	Highlights      []string  `json:"highlights"`
	HighlightScores []float64 `json:"highlightScores"`
	Summary         string    `json:"summary"`
	Subpages        []Subpage `json:"subpages"`
	Entities        []Entity  `json:"entities"`
	Extras          Extras    `json:"extras"`
}

type Subpage struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	PublishedAt string `json:"publishedDate"`
	Author      string `json:"author"`
	ID          string `json:"id"`
	Image       string `json:"image"`
	Favicon     string `json:"favicon"`
}

type Entity struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"`
	Version    int              `json:"version"`
	Properties EntityProperties `json:"properties"`
}

type EntityProperties struct {
	Name         string       `json:"name"`
	FoundedYear  int          `json:"foundedYear"`
	Description  string       `json:"description"`
	Workforce    Workforce    `json:"workforce"`
	Headquarters Headquarters `json:"headquarters"`
	Financials   Financials   `json:"financials"`
	WebTraffic   WebTraffic   `json:"webTraffic"`
}

type Workforce struct {
	Total int `json:"total"`
}

type Headquarters struct {
	Address    string `json:"address"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

type Financials struct {
	RevenueAnnual      float64      `json:"revenueAnnual"`
	FundingTotal       float64      `json:"fundingTotal"`
	FundingLatestRound FundingRound `json:"fundingLatestRound"`
}

type FundingRound struct {
	Name   string  `json:"name"`
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

type WebTraffic struct {
	VisitsMonthly      int              `json:"visitsMonthly"`
	CountryRank        int              `json:"countryRank"`
	AvgDurationSeconds int              `json:"avgDurationSeconds"`
	History            []TrafficHistory `json:"history"`
}

type TrafficHistory struct {
	Value    int    `json:"value"`
	DateFrom string `json:"dateFrom"`
	DateTo   string `json:"dateTo"`
}

type Extras struct {
	Links []string `json:"links"`
}

type Output struct {
	Content   string      `json:"content"`
	Grounding []Grounding `json:"grounding"`
}

type Grounding struct {
	Field     string     `json:"field"`
	Citations []Citation `json:"citations"`
}

type Citation struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type CostDollars struct {
	Total  float64 `json:"total"`
	Search struct {
		Neural float64 `json:"neural"`
	} `json:"search"`
}

type ExaQuery struct {
	Query    string `json:"query"`
	Contents struct {
		Highlights bool `json:"highlights"`
	} `json:"contents"`
}

func (exa *Exa) Query(q string) (*Results, error) {
	query := ExaQuery{
		Query: q,
		Contents: struct {
			Highlights bool `json:"highlights"`
		}{
			Highlights: true,
		},
	}

	jsonBytes, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", exaApi, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", exa.API_KEY)

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResult Results
	if err := json.NewDecoder(resp.Body).Decode(&apiResult); err != nil {
		return nil, err
	}

	apiResult.Results = exa.LimitItems(apiResult.Results)

	return &apiResult, nil
}

func (exa *Exa) LimitItems(items []ResultBody) []ResultBody {
	if exa.Limit <= 0 || len(items) <= exa.Limit {
		return items
	}

	return items[:exa.Limit]
}

func (exa *Exa) GetReferences(items []ResultBody) []References {
	var refs []References
	for i, item := range items {
		snippet := CleanSnippet(item.Summary)
		if snippet == "" && len(item.Highlights) > 0 {
			snippet = CleanSnippet(item.Highlights[0])
		}
		if len(snippet) > 150 {
			snippet = snippet[:150] + "…"
		}
		refs = append(refs, References{
			Title:   StripMarkdown(item.Title),
			Index:   i + 1,
			Url:     item.URL,
			Snippet: snippet,
		})
	}
	return refs
}

func CleanSnippet(s string) string {
	s = strings.TrimSpace(s)
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "..." || line == "…" || line == "•" {
			continue
		}
		lines = append(lines, line)
	}
	return StripMarkdown(strings.Join(lines, " "))
}

func StripMarkdown(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "~~", "")
	s = strings.ReplaceAll(s, "||", "")
	s = strings.ReplaceAll(s, "```", "")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.TrimLeft(s, "# ")
	s = strings.ReplaceAll(s, "\n> ", " ")
	s = strings.TrimPrefix(s, "> ")
	return strings.TrimSpace(s)
}
