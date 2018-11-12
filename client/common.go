package client

import (
	"context"
	"fmt"
	"github.com/brinick/github/authorisation"
)

// ------------------------------------------------------------------

// IGithubClient provides a Github client interface
type IGithubClient interface {
	Post(string, bool, map[string]string) int
	Patch(string, bool, map[string]string) int
	PageGetter
}

// ------------------------------------------------------------------

// PageGetter defines an interface for retrieving Github results pages
type PageGetter interface {
	Get(string, bool) *Page
	GetWithContext(context.Context, string, bool) *Page
}

// ------------------------------------------------------------------

// PageIterator defines the interface for iterating
// over multi-page results from an HTTP GET
type PageIterator interface {
	Next() *Page
	NextWithContext(context.Context) *Page
	Error() error
}

// ------------------------------------------------------------------

// GithubPageIterator is an iterator over Github results pages
type GithubPageIterator struct {
	err      error
	StartURL string
	Current  *Page
	g        PageGetter
}

// NewGithubPageIterator creates a new PageIterator type for iterating over
// Github pages of results
func NewGithubPageIterator(startURL string, pageGetter PageGetter) *GithubPageIterator {
	return &GithubPageIterator{g: pageGetter, StartURL: startURL}
}

func (i *GithubPageIterator) Next() *Page {
	return i.NextWithContext(context.TODO())
}

// NextWithContext will return the next page in the results, and a boolean
// to indicate if there is a subsequent page
func (i *GithubPageIterator) NextWithContext(ctx context.Context) *Page {
	url := i.StartURL

	if i.Current != nil {
		if i.Current.IsLast() {
			// no more pages
			return nil
		}

		url = i.Current.Content.NextLink
	}

	useStableAPI := true
	page := i.g.GetWithContext(ctx, url, useStableAPI)
	i.Current = page
	i.err = page.Err
	return page
}

func (i *GithubPageIterator) Error() error {
	return i.err
}

// ------------------------------------------------------------------

// Page holds links to the current, next and last pages of results
// Github results
type Page struct {
	URL        string
	Content    *Payload
	Err        error
	StatusCode int
}

func (p *Page) NoContent() bool {
	return p.Content.Empty()
}

// IsLast indicates if there is another page after this one
func (p *Page) IsLast() bool {
	return p.Content.NextLink == ""
}

func (p *Page) String() string {
	return fmt.Sprintf(
		"%s:%d -- IsLast? %t",
		p.URL,
		p.StatusCode,
		p.IsLast(),
	)
}

// ------------------------------------------------------------------

// Payload represents the returned data from a Github GET
// It is a "page" data.
type Payload struct {
	Data         string // json string
	ETag         string
	LastModified string
	NextLink     string
}

func (p *Payload) Empty() bool {
	return p == nil || p.Data == "" || p.Data == "[]"
}

// ------------------------------------------------------------------

// GetHeaders gets the headers required for a GET request
func GetHeaders(tr authorisation.TokenRetriever, useStableAPI bool, etag, lastModified string) map[string]string {
	headers := authorisation.Headers(tr, useStableAPI)
	if etag != "" {
		headers["If-None-Match"] = etag
	}
	if lastModified != "" {
		headers["If-Modified-Since"] = lastModified
	}

	return headers
}

// ------------------------------------------------------------------

// PostHeaders gets the headers required for a POST request
func PostHeaders(tr authorisation.TokenRetriever, useStableAPI bool) map[string]string {
	return authorisation.Headers(tr, useStableAPI)
}
