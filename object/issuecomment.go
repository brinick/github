package object

import (
	"context"
	"fmt"
	"github.com/brinick/github/client"
)

// ------------------------------------------------------------------

type issueCommentsIterator struct {
	Err          error
	it           client.PageIterator
	currentPage  []*IssueComment
	current      *IssueComment
	currentIndex int
}

func (i *issueCommentsIterator) Item() *IssueComment {
	return i.current
}

func (i *issueCommentsIterator) HasNext() bool {
	i.NextWithContext(context.TODO())
	return i.current != nil
}

func (i *issueCommentsIterator) HasNextWithContext(ctx context.Context) bool {
	i.NextWithContext(ctx)
	return i.current != nil
}

func (i *issueCommentsIterator) Next() {
	i.NextWithContext(context.TODO())
}

func (i *issueCommentsIterator) NextWithContext(ctx context.Context) {
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

func (i *issueCommentsIterator) load(ctx context.Context) error {
	var err error
	i.currentPage, err = i.nextPage(ctx)
	if err != nil && err != NoMorePages {
		i.Err = err
	} else {
		i.currentIndex = 0
	}

	return err
}

func (i *issueCommentsIterator) nextPage(ctx context.Context) ([]*IssueComment, error) {
	var items []*IssueComment
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

// IssueComment is a comment associated with a given Github issue
type IssueComment struct {
	ID        int
	URL       string
	Author    string
	Body      string
	CreatedAt int
	UpdatedAt int
}

// ------------------------------------------------------------------

func (ic IssueComment) Update(data map[string]string) (int, error) {
	return ic.UpdateWithContext(context.TODO(), data)
}

func (ic IssueComment) UpdateWithContext(ctx context.Context, data map[string]string) (int, error) {
	url := format("%s/%d", ic.URL, ic.ID)
	return HTTPClient().PatchWithContext(ctx, url, true, data)
}

func (ic IssueComment) String() string {
	return fmt.Sprintf("%d: %s", ic.ID, ic.Author)
}
