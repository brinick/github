package object

import (
	"context"
	"github.com/brinick/github/client"
	"reflect"
)

type commitStatusesIterator struct {
	Err          error
	it           client.PageIterator
	currentPage  []*CommitStatus
	current      *CommitStatus
	currentIndex int
}

func (i *commitStatusesIterator) Item() *CommitStatus {
	return i.current
}

func (i *commitStatusesIterator) HasNext() bool {
	i.NextWithContext(context.TODO())
	return i.current != nil
}

func (i *commitStatusesIterator) HasNextWithContext(ctx context.Context) bool {
	i.NextWithContext(ctx)
	select {
	case <-ctx.Done():
		i.Err = ctx.Err()
		return false
	default:
		return i.current != nil
	}
}

func (i *commitStatusesIterator) Next() {
	i.NextWithContext(context.TODO())
}

func (i *commitStatusesIterator) NextWithContext(ctx context.Context) {
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

func (i *commitStatusesIterator) load(ctx context.Context) error {
	var err error
	i.currentPage, err = i.nextPage(ctx)
	if err != nil && err != NoMorePages {
		i.Err = err
	} else {
		i.currentIndex = 0
	}

	return err
}

func (i *commitStatusesIterator) nextPage(ctx context.Context) ([]*CommitStatus, error) {
	var items []*CommitStatus
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

// CommitStatus is a status associated with a commit
type CommitStatus struct {
	State       string `json:"state,omitempty"`
	TargetURL   string `json:"target_url,omitempty"`
	Description string `json:"description,omitempty"`
	Context     string `json:"context,omitempty"`
	Creator     *User  `json:"creator,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// Get a map representation of this commit status that
// can be used for updating the parent commit in the repository
func (cs CommitStatus) toDict() map[string]string {
	return map[string]string{
		"state":       cs.State,
		"target_url":  cs.TargetURL,
		"description": cs.Description,
		"context":     cs.Context,
	}
}

// Equal tests if two CommitStatus objects are considered equal
func (cs CommitStatus) Equal(other *CommitStatus) bool {
	return reflect.DeepEqual(cs.toDict(), other.toDict())
}

func (cs CommitStatus) String() string {
	return format(
		"Context: %s, State: %s, Description: %s, URL: %s",
		cs.Context,
		cs.State,
		cs.Description,
		cs.TargetURL,
	)
}
