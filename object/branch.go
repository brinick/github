package object

import (
	"context"

	"github.com/brinick/github/client"
)

type BranchesGetter interface {
	Branches() (*branchesIterator, error)
	Branch(string) (*RepoBranch, error)
}

// ------------------------------------------------------------------

type branchesIterator struct {
	Err          error
	it           client.PageIterator
	currentPage  []*RepoBranch
	current      *RepoBranch
	currentIndex int
}

func (i *branchesIterator) Item() *RepoBranch {
	return i.current
}

func (i *branchesIterator) HasNext() bool {
	i.NextWithContext(context.TODO())
	return i.current != nil
}

func (i *branchesIterator) HasNextWithContext(ctx context.Context) bool {
	i.NextWithContext(ctx)
	return i.current != nil
}

func (i *branchesIterator) Next() {
	i.NextWithContext(context.TODO())
}

func (i *branchesIterator) NextWithContext(ctx context.Context) {
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

func (i *branchesIterator) load(ctx context.Context) error {
	var err error
	i.currentPage, err = i.nextPage(ctx)
	if err != nil && err != NoMorePages {
		i.Err = err
	} else {
		i.currentIndex = 0
	}

	return err
}

func (i *branchesIterator) nextPage(ctx context.Context) ([]*RepoBranch, error) {
	var items []*RepoBranch
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

// RepoBranch is a repository branch
type RepoBranch struct {
	Name string      `json:"name,omitempty"`
	Head *RepoCommit `json:"commit,omitempty"`
}

func (b RepoBranch) HeadCommit() *RepoCommit {
	return b.Head
}

func (b RepoBranch) String() string {
	return b.Name
}
