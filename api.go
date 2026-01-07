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
	ID                  int64     `json:"id"`
	Body                string    `json:"body"`
	CreatedAt           time.Time `json:"created_at"`
	User                User      `json:"user"`
	DiffHunk            string    `json:"diff_hunk,omitempty"`
	Path                string    `json:"path,omitempty"`
	CommitID            string    `json:"commit_id,omitempty"`
	OriginalCommitID    string    `json:"original_commit_id,omitempty"`
	Line                *int      `json:"line,omitempty"`
	OriginalLine        *int      `json:"original_line,omitempty"`
	InReplyToID         *int64    `json:"in_reply_to_id,omitempty"`
	PullRequestReviewID int64     `json:"pull_request_review_id,omitempty"`
}

// Review represents a PR review
type Review struct {
	ID          int64           `json:"id"`
	Body        string          `json:"body"`
	State       string          `json:"state"`
	SubmittedAt time.Time       `json:"submitted_at"`
	User        User            `json:"user"`
	Threads     []CommentThread `json:"-"`
	ReplyCount  int             `json:"-"`
}

// CommentThread represents a thread of comments on a single diff location
type CommentThread struct {
	Comments []Comment
}

// Commit represents a git commit
type Commit struct {
	SHA       string `json:"sha"`
	Message   string
	Author    User
	Checks    CheckCounts `json:"-"`
	CreatedAt time.Time   `json:"-"`
}

// CheckCounts holds counts of check runs by status
type CheckCounts struct {
	Succeeded int
	Failed    int
	Pending   int
	Skipped   int
}

type commitResponse struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message   string `json:"message"`
		Committer struct {
			Date time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
	Author User `json:"author"`
}

type checkRunsResponse struct {
	CheckRuns []struct {
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
	} `json:"check_runs"`
}

type statusResponse struct {
	State string `json:"state"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user"`
	Comments  []Comment `json:"-"`
	Reviews   []Review  `json:"-"`
	Commits   []Commit  `json:"-"`
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

// FetchAllReviewComments retrieves all review comments for a pull request
func FetchAllReviewComments(client *api.RESTClient, repo repository.Repository, prNumber int) ([]Comment, error) {
	var comments []Comment
	err := client.Get(fmt.Sprintf("repos/%s/%s/pulls/%d/comments",
		repo.Owner, repo.Name, prNumber), &comments)
	return comments, err
}

// FetchCommits retrieves commits for a pull request
func FetchCommits(client *api.RESTClient, repo repository.Repository, prNumber int) ([]Commit, error) {
	var responses []commitResponse
	err := client.Get(fmt.Sprintf("repos/%s/%s/pulls/%d/commits", repo.Owner, repo.Name, prNumber), &responses)
	if err != nil {
		return nil, err
	}

	commits := make([]Commit, len(responses))
	for i, r := range responses {
		msg := r.Commit.Message
		if idx := strings.Index(msg, "\n"); idx != -1 {
			msg = msg[:idx]
		}
		commits[i] = Commit{
			SHA:       r.SHA,
			Message:   msg,
			Author:    r.Author,
			CreatedAt: r.Commit.Committer.Date,
		}
	}
	return commits, nil
}

// FetchCommitChecks retrieves check run counts for a commit
func FetchCommitChecks(client *api.RESTClient, repo repository.Repository, sha string) CheckCounts {
	var counts CheckCounts

	var checkRuns checkRunsResponse
	err := client.Get(fmt.Sprintf("repos/%s/%s/commits/%s/check-runs", repo.Owner, repo.Name, sha), &checkRuns)
	if err != nil {
		return counts
	}

	for _, run := range checkRuns.CheckRuns {
		if run.Status != "completed" {
			counts.Pending++
		} else {
			switch run.Conclusion {
			case "success":
				counts.Succeeded++
			case "failure", "cancelled", "timed_out":
				counts.Failed++
			case "skipped":
				counts.Skipped++
			}
		}
	}

	return counts
}
