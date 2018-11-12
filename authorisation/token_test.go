package authorisation

import (
	"testing"

	"github.com/brinick/github"
)

type MockGithubToken struct {
	GithubToken
}

func (m *MockGithubToken) LoadToken() string {
	return "abc123"
}

func (m *MockGithubToken) Token() string {
	return "abc123"
}

// ------------------------------------------------------------------

// TestHeaders tests that the correct headers are returned
func TestHeaders(t *testing.T) {
	tt := []struct {
		name         string
		useStableAPI bool
		token        TokenRetriever
		accept       string
	}{
		{"Get stable", true, &MockGithubToken{GithubToken: GithubToken{}}, github.APIURLs.STABLE},
		{"Get preview", false, &MockGithubToken{GithubToken: GithubToken{}}, github.APIURLs.PREVIEW},
		{"Pass nil token 1", true, nil, ""},
		{"Pass nil token 2", false, nil, ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			dict, _ := Headers(tc.token, tc.useStableAPI)
			if tc.token == nil {
				if dict != nil {
					t.Errorf("Expected nil map, got %v", dict)
				}
			} else {
				if dict["Accept"] != tc.accept {
					t.Errorf("Expected %v, got %v", tc.accept, dict["Accept"])
				}
			}
		})
	}
}

/*
func TestRateLimiting(t *testing.T) {
	token := &MockGithubToken{GithubToken: GithubToken{}}
	calls := RateLimiting(token)
}
*/
