package lib

import (
	"context"

	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

type Google struct {
	APIKey     string
	CXEngineID string
	Limit      int
}

func GoogelClient(g Google) *Google {
	return &Google{
		APIKey:     g.APIKey,
		CXEngineID: g.CXEngineID,
		Limit:      g.Limit,
	}
}

func (g *Google) Search(q string) (*customsearch.Search, error) {
	context := context.Background()
	svc, err := customsearch.NewService(context, option.WithAPIKey(g.APIKey))
	if err != nil {
		return nil, err
	}

	return svc.Cse.List().Cx(g.CXEngineID).Q(q).Do()
}

func (g *Google) LimitItems(items []*customsearch.Result) []*customsearch.Result {
	if g.Limit <= 0 {
		return items
	}

	if len(items) <= g.Limit {
		return items
	}

	return items[:g.Limit]
}

type References struct {
	Title   string
	Index   int
	Url     string
	Snippet string
}

func (g *Google) GetReferences(items []*customsearch.Result) []References {
	if len(items) == 0 {
		return []References{}
	}

	var references []References

	for idx, item := range items {
		references = append(references, References{Title: item.Title, Url: item.Link, Index: idx + 1, Snippet: item.Snippet})
	}

	return references
}
