package object

import (
	"context"
	"fmt"

	"github.com/brinick/github/client"
)

type teamMembersIterator struct {
	Err          error
	it           client.PageIterator
	currentPage  []*TeamMember
	current      *TeamMember
	currentIndex int
}

func (i *teamMembersIterator) Item() *TeamMember {
	return i.current
}

func (i *teamMembersIterator) HasNext() bool {
	i.NextWithContext(context.TODO())
	return i.current != nil
}

func (i *teamMembersIterator) HasNextWithContext(ctx context.Context) bool {
	i.NextWithContext(ctx)
	return i.current != nil
}

func (i *teamMembersIterator) Next() {
	i.NextWithContext(context.TODO())
}

func (i *teamMembersIterator) NextWithContext(ctx context.Context) {
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

func (i *teamMembersIterator) load(ctx context.Context) error {
	var err error
	i.currentPage, err = i.nextPage(ctx)
	if err != nil && err != NoMorePages {
		i.Err = err
	} else {
		i.currentIndex = 0
	}

	return err
}

func (i *teamMembersIterator) nextPage(ctx context.Context) ([]*TeamMember, error) {
	var items []*TeamMember
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

// TeamMember is a team member representation
type TeamMember struct {
	ID    int
	Login string
}

func (tm TeamMember) String() string {
	return fmt.Sprintf("%s(%d)", tm.Login, tm.ID)
}
