package authorisation

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/brinick/github"
)

// ------------------------------------------------------------------

// APICalls represents the remaining/limit API calls for a given token
type APICalls struct {
	Remaining int
	Limit     int
}

// ------------------------------------------------------------------

// TokenRetriever represents a type that fetches Github Tokens
type TokenRetriever interface {
	LoadToken() string
	Token() string
}

// ------------------------------------------------------------------

// GithubToken represents a particular Github token
type GithubToken struct {
	value    string
	apiCalls *APICalls
}

// LoadToken fetches the token from the GITHUB_TOKEN env var
func (gt *GithubToken) LoadToken() string {
	val, present := os.LookupEnv("GITHUB_TOKEN")
	if !present {
		panic("Please set the GITHUB_TOKEN env var")
	}

	return strings.TrimSpace(val)
}

// Token returns the Github token value
func (gt *GithubToken) Token() string {
	return gt.value
}

// ------------------------------------------------------------------

// NewToken constructs a GithubToken object
func NewToken() *GithubToken {
	t := new(GithubToken)
	t.value = t.LoadToken()
	return t
}

// ------------------------------------------------------------------

var (
	ErrNilTokenRetriever      = errors.New("Cannot provide a nil TokenRetriever")
	ErrRateLimitUnretrievable = errors.New("Unable to retrieve rate limit for the current token")
	ErrHTTPRequestFailure     = errors.New("Unable to create HTTP Request")
)

// Headers gets the HTTP headers pertaining to authorisation
func Headers(tr TokenRetriever, useStableAPI bool) (map[string]string, error) {
	if tr == nil {
		return nil, ErrNilTokenRetriever
	}

	acceptVal := ""
	if useStableAPI {
		acceptVal += github.APIURLs.STABLE
	} else {
		acceptVal += github.APIURLs.PREVIEW
	}

	tokenVal := "token " + tr.Token()
	return map[string]string{"Accept": acceptVal, "Authorization": tokenVal}, nil
}

// ------------------------------------------------------------------

// RateLimiting gets the remaining/limit API calls for the given token
func RateLimiting(token TokenRetriever) (*APICalls, error) {
	url := filepath.Join(github.APIURLs.URL, "/rate_limit")
	headers, err := Headers(token, true)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, ErrHTTPRequestFailure
	}

	for key, val := range headers {
		req.Header.Set(key, val)
	}

	c := &http.Client{}
	resp, err := c.Do(req)

	if err != nil {
		return nil, ErrRateLimitUnretrievable
	}
	defer resp.Body.Close()

	remaining, limit := -1, -1
	if resp.StatusCode == http.StatusOK {
		val, keyExists := resp.Header["X-RateLimit-Remaining"]
		if keyExists && len(val) > 0 {
			remaining, err = strconv.Atoi(val[0])
			if err != nil {
				remaining = -1
			}
		}
		val, keyExists = resp.Header["X-RateLimit-Limit"]
		if keyExists && len(val) > 0 {
			limit, err = strconv.Atoi(val[0])
			if err != nil {
				limit = -1
			}
		}
	}

	return &APICalls{remaining, limit}, nil
}
