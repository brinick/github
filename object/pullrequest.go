package object

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/brinick/logging"

	"github.com/brinick/github/client"
)

//
var legalPRStates = [3]string{"open", "closed", "all"}

func isLegalPullRequestState(state string) bool {
	for _, legalState := range legalPRStates {
		if state == legalState {
			return true
		}
	}

	return false
}

// ------------------------------------------------------------------

type pullsIterator struct {
	Err          error
	it           client.PageIterator
	currentPage  []*PullRequest
	current      *PullRequest
	currentIndex int
}

func (i *pullsIterator) Item() *PullRequest {
	return i.current
}

func (i *pullsIterator) HasNext() bool {
	i.NextWithContext(context.TODO())
	return i.current != nil
}

func (i *pullsIterator) HasNextWithContext(ctx context.Context) bool {
	i.NextWithContext(ctx)
	select {
	case <-ctx.Done():
		i.Err = ctx.Err()
		return false
	default:
		return i.current != nil
	}
}

func (i *pullsIterator) Next() {
	i.NextWithContext(context.TODO())
}

func (i *pullsIterator) NextWithContext(ctx context.Context) {
	if i.Err != nil {
		i.current = nil
		return
	}

	if i.currentPage == nil || i.currentIndex == len(i.currentPage) {
		if err := i.load(ctx); err != nil {
			i.current = nil
			return
		}
	}

	i.current = i.currentPage[i.currentIndex]
	i.currentIndex++
}

func (i *pullsIterator) load(ctx context.Context) error {
	var err error
	i.currentPage, err = i.nextPage(ctx)
	if err != nil && err != NoMorePages {
		i.Err = err
	} else {
		i.currentIndex = 0
	}

	return err
}

func (i *pullsIterator) nextPage(ctx context.Context) ([]*PullRequest, error) {
	var items []*PullRequest
	page := i.it.NextWithContext(ctx)
	if page == nil {
		// no more data
		return nil, NoMorePages
	} else if page.NoContent() {
		noContent := fmt.Errorf(fmt.Sprintf("No content (status %d)", page.StatusCode))
		return nil, noContent
	}

	if page.Err == nil {
		parseJSON(page.Content.Data, &items)
	}

	return items, i.it.Error()
}

// ------------------------------------------------------------------

type PullsGetter interface {
	Pulls(string, string, string) (*pullsIterator, error)
	Pull(int) (*PullRequest, error)
}

// ------------------------------------------------------------------

type pullrequestBranch struct {
	SHA string `json:"sha,omitempty"`
}

// PullRequest is a repository pull request object
type PullRequest struct {
	Number    int                `json:"number,omitempty"`
	State     string             `json:"state,omitempty"`
	Title     string             `json:"title,omitempty"`
	Body      string             `json:"body,omitempty"`
	Head      *pullrequestBranch `json:"head,omitempty"`
	URL       string             `json:"url,omitempty"`
	HTMLURL   string             `json:"html_url,omitempty"`
	CreatedAt time.Time          `json:"created_at,omitempty"`
	UpdatedAt time.Time          `json:"updated_at,omitempty"`
	ClosedAt  time.Time          `json:"closed_at,omitempty"`
	Author    *User              `json:"user,omitempty"`
	Assignee  *User              `json:"assignee,omitempty"`
}

// IsOpen returns true if the pull request has state "open"
func (p PullRequest) IsOpen() bool {
	return p.State == "open"
}

func (p PullRequest) toURL(suffix ...string) string {
	return format("%s/%s", p.URL, filepath.Join(suffix...))
}

// Commits returns the list of all commits for this pull request
func (p PullRequest) Commits() (*commitsIterator, error) {
	url := p.toURL("commits")
	it := PageIterator(url, HTTPClient())
	return &commitsIterator{it: it}, nil
}

func (p PullRequest) HeadCommit() (*RepoCommit, error) {
	return p.HeadCommitWithContext(context.TODO())
}

func (p PullRequest) HeadCommitWithContext(ctx context.Context) (*RepoCommit, error) {
	var commit *RepoCommit

	commits, err := p.Commits()
	if err != nil {
		return nil, err
	}

	// loop to the last one, which is the head commit
	for commits.HasNextWithContext(ctx) {
		commit = commits.Item()
	}

	return commit, commits.Err
}

/*
// Files retrieves the list of files changed by this pull request
func (p PullRequest) Files() ([]*CommitFile, error) {
	var (
		err     error
		commits []*RepoCommit
		files   = []*CommitFile{}
	)

	commits, err = p.Commits()
	for commit := commits.Next(); commit != nil; {
		files = append(files, commit.Files...)
	}

	// TODO: files list may have  > 1 entry for a given file
	// TODO: commit may be nil (thus breaking the loop) if there is an error,
	// we should check for that
	return files, err
}
*/
// ------------------------------------------------------------------

func (p PullRequest) String() string {
	author := "<n/a>"
	if p.Author != nil {
		author = p.Author.Login
	}
	return format("[%d:%s:%s] %s", p.Number, p.State, author, p.Title)
}
