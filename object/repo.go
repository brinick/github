package object

import (
	"context"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/brinick/github"
)

// ------------------------------------------------------------------

type BaseRepoer interface {
	Path() string
	FullPath() string
}

type PullsRepoer interface {
	BaseRepoer
	PullsGetter
}

type RepoFetcher interface {
	RepoCollaborator
	PullsRepoer
}

type RepoCollaborator interface {
	IsCollaborator(string) bool
}

type Repoer interface {
	BaseRepoer
	PullsGetter
	IssuesGetter
	CommitsGetter
	BranchesGetter
	RepoCollaborator
}

// ------------------------------------------------------------------

// Repository is a Github repo
type Repository struct {
	owner string
	name  string
}

// NewRepo creates a new repository instance
func NewRepo(owner, name string) *Repository {
	return &Repository{
		owner: owner,
		name:  name,
	}
}

func NewRepoFromPath(path string) *Repository {
	tokens := strings.Split(path, "/")
	if len(tokens) != 2 {
		return nil
	}

	owner, name := tokens[0], tokens[1]
	return NewRepo(owner, name)
}

func (r Repository) Owner() string {
	return r.owner
}

func (r Repository) Name() string {
	return r.name
}

// Path gets the path fragment joining the repository's owner and name
func (r Repository) Path() string {
	return filepath.Join(r.owner, r.name)
}

// FullPath is the full URL to the repository
func (r Repository) FullPath() string {
	return format("%s/%s/%s", github.APIURLs.URL, "repos", r.Path())
}

func (r Repository) toURL(suffix ...string) string {
	return format("%s/%s", r.FullPath(), filepath.Join(suffix...))
}

// Pulls gets an iterator over the repository's pull requests with
// given state and author, and in the given branch
func (r Repository) Pulls(branch, state string) (*pullsIterator, error) {
	return r.PullsWithContext(
		context.TODO(),
		branch,
		state,
	)
}

// PullsWithContext gets an iterator over the repository's
// pull requests in a given branch and with the given state.
// A context done error will be returned if the context is done.
func (r Repository) PullsWithContext(
	ctx context.Context,
	branch, state string,
) (*pullsIterator, error) {

	/*
		if !isLegalPullRequestState(state) {
			illegalStateError := errors.New(
				format("repo: unknown pull request state %s", state),
			)
			return nil, illegalStateError
		}

		exists, err := r.BranchExistsWithContext(ctx, branch)

		if err != nil {
			switch err {
			case context.Canceled, context.DeadlineExceeded:
				return nil, err
			default:
				str := format("%s: unknown branch %s", r.Path(), branch)
				return nil, errors.New(str)
			}
		}

	*/
	url := r.toURL(format("pulls?base=%s&state=%s", branch, state))
	it := PageIterator(url, HTTPClient())
	return &pullsIterator{it: it}, nil
}

// ------------------------------------------------------------------

// Pull retrieves the pull request with the given number
func (r Repository) Pull(number int) (*PullRequest, error) {
	var pull *PullRequest
	url := r.toURL("pulls", strconv.Itoa(number))
	page := HTTPClient().Get(url, true)
	if page.Err == nil {
		parseJSON(page.Content.Data, &pull)
	}
	return pull, page.Err
}

// Issues will fetch an iterator over the issues associated with this repository.
// Note that every pull request is an issue, but not every issue is
// a pull request! By default, the Github API will return both.
// To fetch only issues that are not pull requests, set includePRs to false.
// To retrieve only unassigned issues, set the assignee to "".
func (r Repository) Issues(state, author, assignee string, includePRs bool) (*issuesIterator, error) {
	// TODO: incorporate the includePRs flag in the params
	params := []string{}
	params = append(params, format("state=%s", state))

	if strings.TrimSpace(assignee) == "" {
		assignee = "none"
	}
	params = append(params, format("assignee=%s", assignee))

	if strings.TrimSpace(author) != "" {
		params = append(params, format("creator=%s", author))
	}
	paramsStr := strings.Join(params, "&")

	url := r.toURL(format("issues?%s", paramsStr))
	it := PageIterator(url, HTTPClient())
	return &issuesIterator{it: it}, nil
}

// Issue retrieves the repository issue with the given number
func (r *Repository) Issue(number int) (*RepoIssue, error) {
	var issue *RepoIssue
	url := r.toURL("issues", strconv.Itoa(number))
	page := HTTPClient().Get(url, true)
	if page.Err == nil {
		parseJSON(page.Content.Data, &issue)
	}
	return issue, page.Err
}

// Branches returns an iterator over the branches within the repository
func (r *Repository) Branches() (*branchesIterator, error) {
	url := r.toURL("branches")
	it := PageIterator(url, HTTPClient())
	return &branchesIterator{it: it}, nil
}

// Branch fetches the branch with the given name
func (r *Repository) Branch(branchName string) (*RepoBranch, error) {
	return r.BranchWithContext(context.TODO(), branchName)
}

// BranchWithContext fetches the branch with the given name
func (r *Repository) BranchWithContext(
	ctx context.Context,
	branchName string,
) (*RepoBranch, error) {

	var b *RepoBranch
	url := r.toURL("branches", branchName)
	page := HTTPClient().GetWithContext(ctx, url, true)
	if page.Err == nil {
		parseJSON(page.Content.Data, &b)
	}
	return b, page.Err
}

// BranchExists indicates if a given branch exists
func (r *Repository) BranchExists(branchName string) (bool, error) {
	return r.BranchExistsWithContext(context.TODO(), branchName)
}

func (r *Repository) BranchExistsWithContext(
	ctx context.Context,
	branchName string,
) (bool, error) {

	if _, err := r.BranchWithContext(ctx, branchName); err != nil {
		return false, err
	}
	return true, nil
}

// ------------------------------------------------------------------

// Commits gets the list of commits for this branch.
func (r *Repository) Commits(branchName string) (*commitsIterator, error) {
	url := r.toURL(format("commits?sha=%s", branchName))
	it := PageIterator(url, HTTPClient())
	return &commitsIterator{it: it}, nil
}

// ------------------------------------------------------------------

// Commit will get the commit with the given SHA
func (r *Repository) Commit(sha string) (*RepoCommit, error) {
	var commit *RepoCommit
	url := r.toURL("commits", sha)
	page := HTTPClient().Get(url, true)
	if page.Err == nil {
		parseJSON(page.Content.Data, &commit)
	}
	return commit, page.Err
}

// ------------------------------------------------------------------

// IsCollaborator checks if a given Github account is a collaborator
// in this repository
func (r *Repository) IsCollaborator(login string) bool {
	// TODO: check permissions or check simply if user is a collaborator?
	// https://developer.github.com/v3/repos/collaborators/#check-if-a-user-is-a-collaborator
	url := r.toURL("collaborators", login)
	page := HTTPClient().Get(url, true)
	return page.StatusCode == http.StatusNoContent // 404 = no such user
}

// ------------------------------------------------------------------
