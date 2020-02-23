package object

import (
	"context"
	"fmt"
	"github.com/brinick/github/client"
	"time"
)

// ------------------------------------------------------------------

type CommitsGetter interface {
	Commits() (*commitsIterator, error)
	Commit(string) (*RepoCommit, error)
}

// ------------------------------------------------------------------

type commitsIterator struct {
	Err          error
	it           client.PageIterator
	currentPage  []*RepoCommit
	current      *RepoCommit
	currentIndex int
}

func (i *commitsIterator) Item() *RepoCommit {
	return i.current
}

func (i *commitsIterator) HasNext() bool {
	i.NextWithContext(context.TODO())
	return i.current != nil
}

func (i *commitsIterator) HasNextWithContext(ctx context.Context) bool {
	i.NextWithContext(ctx)
	select {
	case <-ctx.Done():
		i.Err = ctx.Err()
		return false
	default:
		return i.current != nil
	}
}

func (i *commitsIterator) Next() {
	i.NextWithContext(context.TODO())
}

func (i *commitsIterator) NextWithContext(ctx context.Context) {
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

func (i *commitsIterator) load(ctx context.Context) error {
	var err error
	i.currentPage, err = i.nextPage(ctx)
	if err != nil && err != NoMorePages {
		i.Err = err
	} else {
		i.currentIndex = 0
	}

	return err
}

func (i *commitsIterator) nextPage(ctx context.Context) ([]*RepoCommit, error) {
	var items []*RepoCommit
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

// RepoCommit is a repository commit
type RepoCommit struct {
	SHA       string `json:"sha,omitempty"`
	URL       string `json:"url,omitempty"`
	HTMLURL   string `json:"html_url,omitempty"`
	Author    *User  `json:"author,omitempty"`
	Message   string
	Committer *User         `json:"committer,omitempty"`
	Commit    *innerCommit  `json:"commit,omitempty"`
	Stats     *CommitStats  `json:"stats,omitempty"`
	Files     []*CommitFile `json:"files,omitempty"`
}

type innerCommit struct {
	Message string          `json:"message,omitempty"`
	Info    *committerBrief `json:"committer,omitempty"`
}

type committerBrief struct {
	Name      string    `json:"name,omitempty"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"date,omitempty"`
}

// CommitStats represents the statistics for the commit
type CommitStats struct {
	Additions int `json:"additions,omitempty"`
	Deletions int `json:"deletions,omitempty"`
	Total     int `json:"total,omitempty"`
}

// CommitFile is a file committed
type CommitFile struct {
	Name      string `json:"filename,omitempty"`
	Additions int    `json:"additions,omitempty"`
	Deletions int    `json:"deletions,omitempty"`
	Changes   int    `json:"changes,omitempty"`
}

// ------------------------------------------------------------------

// Statuses retrieves the list of statuses associated with the commit
func (c *RepoCommit) Statuses() (*commitStatusesIterator, error) {
	url := format("%s/%s", c.URL, "statuses")
	it := PageIterator(url, HTTPClient())
	return &commitStatusesIterator{it: it}, nil

}

// HasStatus checks if the commit has the exact commit status passed in
func (c *RepoCommit) HasStatus(cs *CommitStatus) bool {
	statuses, _ := c.Statuses()

	for statuses.HasNext() {
		if statuses.Item().Equal(cs) {
			return true
		}
	}

	return false
}

// SetStatusWithContext creates the given status via HTTP POST,
// returning the HTTP status code
func (c *RepoCommit) SetStatusWithContext(ctx context.Context, status *CommitStatus) (int, error) {
	if c.HasStatus(status) {
		return 0, fmt.Errorf("Status already exists")
	}

	url := format("%s/%s", c.URL, "statuses")
	fmt.Println(url)

	return HTTPClient().PostWithContext(ctx, url, true, status.toDict())
}

// SetStatus creates the given status via HTTP POST,
// returning the HTTP status code
func (c *RepoCommit) SetStatus(status *CommitStatus) (int, error) {
	return c.SetStatusWithContext(context.TODO(), status)
}

func (c *RepoCommit) String() string {
	author := "<n/a>"
	if c.Author != nil {
		author = c.Author.Login
	}

	var message string

	// depends if njectMessage has been called yet
	if c.Commit != nil {
		message = c.Commit.Message
	} else {
		message = c.Message
	}
	return format("[%s:%s] %s", c.SHA, author, message)
}
