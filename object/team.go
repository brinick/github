package object

import (
	"context"
	"fmt"

	"github.com/brinick/github/client"
)

// ------------------------------------------------------------------

type teamsIterator struct {
	Err          error
	it           client.PageIterator
	currentPage  []*Team
	current      *Team
	currentIndex int
}

func (i *teamsIterator) Item() *Team {
	return i.current
}

func (i *teamsIterator) HasNext() bool {
	i.NextWithContext(context.TODO())
	return i.current != nil
}

func (i *teamsIterator) HasNextWithContext(ctx context.Context) bool {
	i.NextWithContext(ctx)
	return i.current != nil
}

func (i *teamsIterator) Next() {
	i.NextWithContext(context.TODO())
}

func (i *teamsIterator) NextWithContext(ctx context.Context) {
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

func (i *teamsIterator) load(ctx context.Context) error {
	var err error
	i.currentPage, err = i.nextPage(ctx)
	if err != nil && err != NoMorePages {
		i.Err = err
	} else {
		i.currentIndex = 0
	}

	return err
}

func (i *teamsIterator) nextPage(ctx context.Context) ([]*Team, error) {
	var items []*Team
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

// Team is a Github team of people
type Team struct {
	ID          int                `json:"id,omitempty"`
	URL         string             `json:"url,omitempty"`
	Name        string             `json:"name,omitempty"`
	Slug        string             `json:"slug,omitempty"`
	Description string             `json:"description,omitempty"`
	NMembers    int                `json:"members_count,omitempty"`
	NRepos      int                `json:"repos_count,omitempty"`
	Org         GithubOrganisation `json:"organization,omitempty"`
	Parent      *Team              `json:"parent,omitempty"`
}

// ------------------------------------------------------------------

// NewTeam creates a new team instance
func NewTeam(id int, name string) *Team {
	return &Team{
		ID:   id,
		Name: name,
	}
}

// Members will fetch an iterator over the members of a Github team
func (t Team) Members() (*teamMembersIterator, error) {
	url := join(t.URL, "members")
	it := PageIterator(url, HTTPClient())
	return &teamMembersIterator{it: it}, nil
}

// IsMember checks for a particular user's membership of a team
func (t Team) IsMember(login string) (bool, error) {
	url := join(t.URL, "memberships", login)
	page := HTTPClient().Get(url, true)
	return page.StatusCode == 200, page.Err
}

func (t Team) String() string {
	return fmt.Sprintf("%s(%d)", t.Name, t.ID)
}

// ------------------------------------------------------------------
