package object

import (
	"context"
	"path/filepath"
	"time"

	"github.com/brinick/github/client"
)

// ------------------------------------------------------------------

type issuesIterator struct {
	Err          error
	it           client.PageIterator
	currentPage  []*RepoIssue
	current      *RepoIssue
	currentIndex int
}

func (i *issuesIterator) Item() *RepoIssue {
	return i.current
}

func (i *issuesIterator) HasNext() bool {
	i.NextWithContext(context.TODO())
	return i.current != nil
}

func (i *issuesIterator) HasNextWithContext(ctx context.Context) bool {
	i.NextWithContext(ctx)
	return i.current != nil
}

func (i *issuesIterator) Next() {
	i.NextWithContext(context.TODO())
}

func (i *issuesIterator) NextWithContext(ctx context.Context) {
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

func (i *issuesIterator) load(ctx context.Context) error {
	var err error
	i.currentPage, err = i.nextPage(ctx)
	if err != nil && err != NoMorePages {
		i.Err = err
	} else {
		i.currentIndex = 0
	}

	return err
}

func (i *issuesIterator) nextPage(ctx context.Context) ([]*RepoIssue, error) {
	var items []*RepoIssue
	page := i.it.NextWithContext(ctx)
	if page == nil || page.NoContent() {
		// page is nil, no results
		return nil, NoMorePages
	}

	if page.Err == nil {
		parseJSON(page.Content.Data, &items)
	}

	return items, i.it.Error()
}

// ------------------------------------------------------------------

type IssuesGetter interface {
	Issues(string, string, string, bool) (*issuesIterator, error)
	Issue(int) (*RepoIssue, error)
}

// ------------------------------------------------------------------

// RepoIssue represents a repository issue
type RepoIssue struct {
	Number    int       `json:"number,omitempty"`
	URL       string    `json:"url,omitempty"`
	State     string    `json:"state,omitempty"`
	Title     string    `json:"title,omitempty"`
	Body      string    `json:"body,omitempty"`
	Assignee  *User     `json:"assignee,omitempty"`
	Assignees []*User   `json:"assignees,omitempty"`
	Author    *User     `json:"user,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	ClosedAt  time.Time `json:"closed_at,omitempty"`
}

// ------------------------------------------------------------------

// Comments returns the lists of comments associated with this issue
func (i RepoIssue) Comments() (*issueCommentsIterator, error) {
	url := join(i.URL, "comments")

	it := PageIterator(url, HTTPClient())
	return &issueCommentsIterator{it: it}, nil
}

// PostComment posts a new comment with the given body to the issue
func (i RepoIssue) PostComment(data map[string]string) (int, error) {
	return i.PostCommentWithContext(context.TODO(), data)
}

func (i RepoIssue) PostCommentWithContext(ctx context.Context, data map[string]string) (int, error) {
	url := i.toURL("comments")
	return HTTPClient().PostWithContext(ctx, url, true, data)
}

func (i RepoIssue) toURL(suffix ...string) string {
	return format("%s/%s", i.URL, filepath.Join(suffix...))
}

func (i RepoIssue) String() string {
	author := NotAvailable
	if i.Author != nil {
		author = i.Author.Login
	}
	return format(
		"[%d:%s:%s] %s",
		i.Number,
		author,
		i.State,
		i.Title,
	)
}
