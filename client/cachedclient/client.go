package cachedclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/brinick/github"
	"github.com/brinick/github/authorisation"
	"github.com/brinick/github/client"
	"github.com/brinick/logging"
)

// TODO: which Github API version? (useStableAPI)

// ------------------------------------------------------------------

// NewClient creates and initialises a new PickledCachedClient
func NewClient() *PickledCachedClient {
	c := new(PickledCachedClient)
	c.APIToken = authorisation.NewToken()
	c.APIURL = github.APIURLs.URL
	cache, _ := NewCache()
	c.cache = cache
	return c
}

// ---------------------------------------------------------------

// PickledCachedClient represents a Github client that caches data
type PickledCachedClient struct {
	APIToken *authorisation.GithubToken
	APIURL   string
	cache    *PickledCache
}

func (c PickledCachedClient) makeURL(urlTpl string, kwds ...interface{}) string {
	if strings.HasPrefix(urlTpl, "/") {
		urlTpl = urlTpl[1:]
	}

	suffix := fmt.Sprintf(urlTpl, kwds...)

	u, _ := url.Parse(c.APIURL)
	u.Path = filepath.Join(u.Path, suffix)
	return u.String()
}

func (c PickledCachedClient) rateLimiting() (*authorisation.APICalls, error) {
	return authorisation.RateLimiting(c.APIToken)
}

// ---------------------------------------------------------------

// Post executes an HTTP POST operation, returning a status code and error.
func (c *PickledCachedClient) Post(
	url string,
	useStableAPI bool,
	data map[string]string,
) (int, error) {
	return c.PostWithContext(context.TODO(), url, useStableAPI, data)
}

// PostWithContext executes an HTTP POST operation, returning a status code.
// It may be cancelled via a context
func (c *PickledCachedClient) PostWithContext(
	ctx context.Context,
	url string,
	useStableAPI bool,
	data map[string]string,
) (int, error) {

	headers, err := client.PostHeaders(c.APIToken, useStableAPI)
	if err != nil {
		return 0, err
	}

	// url = c.makeURL(url, kwds...)
	jsonStr, err := json.Marshal(data)
	if err != nil {
		logging.Error(
			"Unable to JSON encode the HTTP post data",
			logging.F("err", err),
		)
		return 0, err
	}

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	for key, val := range headers {
		req.Header.Set(key, val)
	}
	// req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		logging.Error(
			"Unable to make HTTP POST",
			logging.F("err", err),
		)
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// ---------------------------------------------------------------

// Patch executes an HTTP PATCH operation, returning a status code.
func (c *PickledCachedClient) Patch(
	url string,
	useStableAPI bool,
	data map[string]string,
) (int, error) {
	return c.PatchWithContext(context.TODO(), url, useStableAPI, data)
}

// PatchWithContext executes an HTTP PATCH operation, returning a status code.
// It may be cancelled via the context.
func (c *PickledCachedClient) PatchWithContext(
	ctx context.Context,
	url string,
	useStableAPI bool,
	data map[string]string,
) (int, error) {

	headers, err := client.PostHeaders(c.APIToken, useStableAPI)
	if err != nil {
		return 0, err
	}

	// url = c.makeURL(url, kwds...)
	jsonStr, err := json.Marshal(data)
	if err != nil {
		logging.Error(
			"Unable to JSON encode the HTTP post data",
			logging.F("err", err),
		)
		return 0, err
	}

	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonStr))

	if ctx != nil {
		req = req.WithContext(ctx)
	}
	for key, val := range headers {
		req.Header.Set(key, val)
	}
	// req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		logging.Error(
			"Error making HTTP PATCH action",
			logging.F("err", err),
		)
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

// ---------------------------------------------------------------

// Get executes an HTTP GET operation, returning a Page object
// (that may link to further pages if there are more results to come).
func (c *PickledCachedClient) Get(
	url string,
	useStableAPI bool,
) *client.Page {
	return c.GetWithContext(context.TODO(), url, useStableAPI)
}

// GetWithContext executes an HTTP GET operation,
// returning a Page object (that may link to further pages if there are
// more results to come). It may be cancelled via the context.
func (c *PickledCachedClient) GetWithContext(
	ctx context.Context,
	url string,
	useStableAPI bool,
) *client.Page {

	logging.Info("GET", logging.F("url", url))

	entry := [2]string{"url", url}
	input := [][2]string{entry}
	cacheKey := c.cache.generateCacheID(input)
	cacheValue, keyFound := c.cache.get(cacheKey)
	etag := ""
	last := ""
	if keyFound {
		etag = cacheValue.ETag
		last = cacheValue.LastModified
	}

	req, _ := http.NewRequest("GET", url, nil)
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	headers, err := client.GetHeaders(c.APIToken, useStableAPI, etag, last)
	if err != nil {
		return &client.Page{URL: url, Err: err}
	}

	for key, val := range headers {
		req.Header.Set(key, val)
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)

	if err != nil {
		select {
		case <-ctx.Done():
			logging.Info(
				"Context done, cancelling HTTP GET",
				logging.F("url", url),
			)
			err = ctx.Err()

		default:
			logging.Error(
				"Unable to make GET request",
				logging.F("err", err),
				logging.F("url", url),
			)
		}

		return &client.Page{URL: url, Err: err}
	}

	// TODO: We're ignoring the context from here onwards, as it's assumed
	// that it does not take long to run

	defer resp.Body.Close()

	statusCode := resp.StatusCode
	// -----------------------------------------
	// Exists, but nothing changed
	if statusCode == http.StatusNotModified {
		nextLink := cacheValue.NextLink
		data := cacheValue.Data
		return &client.Page{
			URL: url,
			Content: &client.Payload{
				Data:         data,
				ETag:         etag,
				LastModified: last,
				NextLink:     nextLink,
			},
			StatusCode: http.StatusNotModified,
		}
	}
	// If we get here, then we have a cache miss
	c.cache.delete(cacheKey)

	// -----------------------------------------
	// Inexistant
	if statusCode == http.StatusNotFound {
		return &client.Page{
			URL:        url,
			Err:        errors.New("Not found"),
			StatusCode: http.StatusNotFound,
		}
	}

	// -----------------------------------------
	// Nope
	if statusCode == http.StatusForbidden {
		logging.Info("Forbidden", logging.F("statuscode", http.StatusForbidden))
		return &client.Page{
			URL:        url,
			Err:        errors.New("Forbidden"),
			StatusCode: http.StatusForbidden,
		}
	}

	// -----------------------------------------
	// All ok
	if statusCode == http.StatusOK {
		var bytesArray []byte
		bytesArray, err = ioutil.ReadAll(resp.Body)

		if err != nil {
			errmsg := "Unable to read HTTP response body"
			logging.Error(
				errmsg,
				logging.F("err", err),
				logging.F("statuscode", http.StatusOK),
			)
			return &client.Page{
				URL:        url,
				Err:        errors.New(errmsg),
				StatusCode: http.StatusOK,
			}
		}

		// json string
		payload := string(bytesArray)
		etag = resp.Header.Get("ETag")
		last = resp.Header.Get("Last-Modified")
		nextLink := parseNextLink(resp.Header.Get("Link"))

		cacheValue = client.Payload{
			Data:         payload,
			ETag:         etag,
			LastModified: last,
			NextLink:     nextLink,
		}

		c.cache.update(
			map[string]client.Payload{
				cacheKey: cacheValue,
			},
			true,
		)

		return &client.Page{
			URL:        url,
			Content:    &cacheValue,
			StatusCode: http.StatusOK,
		}

	}

	// -----------------------------------------
	// No content
	if statusCode == http.StatusNoContent {
		cacheValue = client.Payload{
			Data:         "",
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
		}

		c.cache.update(
			map[string]client.Payload{
				cacheKey: cacheValue,
			},
			true,
		)

		return &client.Page{
			Content:    &cacheValue,
			StatusCode: http.StatusNoContent,
		}
	}

	// -----------------------------------------

	return &client.Page{
		Err:        fmt.Errorf("HTTP GET request returned status code %d", statusCode),
		StatusCode: statusCode,
	}

}

func parseNextLink(nextLink string) string {
	if nextLink == "" {
		// If there is only one page of results, the Link header is empty
		return ""
	}

	links := strings.Split(nextLink, ",")
	for _, link := range links {
		tokens := strings.Split(link, ";")
		url, what := tokens[0], tokens[1]
		what = strings.TrimSpace(what)
		if strings.HasPrefix(what, "rel=\"next\"") {
			url = strings.TrimSpace(url)
			url = strings.Trim(url, "<>")
			return url
		}
	}

	return ""
}

func parseHeaderLink(hl string) string {
	// Header Link value is constructed from one or more comma-separated terms like:
	// <https://api.github.com/user/repos?page=3&per_page=100>; rel="next"

	if hl == "" {
		return hl
	}

	next := regexp.MustCompile("<(https://.*?)>; rel=\"next\"")
	for _, link := range strings.Split(hl, ",") {
		link = strings.TrimSpace(link)
		if next.MatchString(link) {
			return link
		}
	}

	return ""
}
