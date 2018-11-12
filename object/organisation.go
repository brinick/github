package object

import (
	"github.com/brinick/github"
)

// GithubOrganisation is a Github organisation
type GithubOrganisation struct {
	Name        string `json:"name,omitempty"`
	ID          int    `json:"id,omitempty"`
	Login       string `json:"login,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	HTMLURL     string `json:"html_url,omitempty"`
	Company     string `json:"company,omitempty"`
	NMembers    int    `json:"collaborators,omitempty"`
}

// Organisation will fetch the GithubOrganisation with the given name
// or return an error if the organisation can not be found
func Organisation(name string) (*GithubOrganisation, error) {
	var org *GithubOrganisation
	url := join(github.APIURLs.URL, "orgs", name)
	page := HTTPClient().Get(url, true)
	if page.Err == nil {
		parseJSON(page.Content.Data, &org)
	}

	return org, page.Err
}

// Teams returns an iterator over the organisation's teams
func (o GithubOrganisation) Teams() (*teamsIterator, error) {
	url := join(o.URL, "teams")
	it := PageIterator(url, HTTPClient())
	return &teamsIterator{it: it}, nil
}
