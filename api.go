package prview

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

// User represents a GitHub user
type User struct {
	Login string `json:"login"`
}

// Comment represents a PR comment (issue comment or review comment)
type Comment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user"`
	DiffHunk  string    `json:"diff_hunk,omitempty"`
}

// Review represents a PR review
type Review struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user"`
	Comments  []Comment `json:"-"` // Filled in later
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user"`
	Comments  []Comment `json:"-"` // Filled in later
	Reviews   []Review  `json:"-"` // Filled in later
}

// GetCurrentRepo returns the current repository information
func GetCurrentRepo() (repository.Repository, error) {
	return repository.Current()
}

// GetRESTClient returns a GitHub REST API client
func GetRESTClient() (*api.RESTClient, error) {
	clientOpts := api.ClientOptions{EnableCache: true}
	return api.NewRESTClient(clientOpts)
}

// GetCurrentBranch returns the name of the current git branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCurrentPR tries to determine the PR number for the current branch
func GetCurrentPR(client *api.RESTClient, repo repository.Repository) (int, error) {
	branch, err := GetCurrentBranch()
	if err != nil {
		return 0, err
	}

	// Find PR for current branch
	var prs []map[string]interface{}
	err = client.Get(fmt.Sprintf("repos/%s/%s/pulls?head=%s:%s&state=open",
		repo.Owner, repo.Name, repo.Owner, branch), &prs)
	if err != nil {
		return 0, err
	}

	if len(prs) == 0 {
		return 0, fmt.Errorf("no open PR found for current branch: %s", branch)
	}

	// Convert the number to int
	prNum, ok := prs[0]["number"]
	if !ok {
		return 0, fmt.Errorf("could not find PR number in response")
	}

	switch n := prNum.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case string:
		return strconv.Atoi(n)
	default:
		return 0, fmt.Errorf("unexpected type for PR number: %T", prNum)
	}
}

// FetchPR retrieves a pull request by number
func FetchPR(client *api.RESTClient, repo repository.Repository, prNumber int) (PullRequest, error) {
	var pr PullRequest
	err := client.Get(fmt.Sprintf("repos/%s/%s/pulls/%d", repo.Owner, repo.Name, prNumber), &pr)
	return pr, err
}

// FetchPRComments retrieves issue comments for a pull request
func FetchPRComments(client *api.RESTClient, repo repository.Repository, prNumber int) ([]Comment, error) {
	var comments []Comment
	err := client.Get(fmt.Sprintf("repos/%s/%s/issues/%d/comments", repo.Owner, repo.Name, prNumber), &comments)
	return comments, err
}

// FetchPRReviews retrieves reviews for a pull request
func FetchPRReviews(client *api.RESTClient, repo repository.Repository, prNumber int) ([]Review, error) {
	var reviews []Review
	err := client.Get(fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", repo.Owner, repo.Name, prNumber), &reviews)
	return reviews, err
}

// FetchReviewComments retrieves comments for a specific review
func FetchReviewComments(client *api.RESTClient, repo repository.Repository, prNumber int, reviewID int64) ([]Comment, error) {
	var comments []Comment
	err := client.Get(fmt.Sprintf("repos/%s/%s/pulls/%d/reviews/%d/comments",
		repo.Owner, repo.Name, prNumber, reviewID), &comments)
	return comments, err
}
