package prview

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"
	"time"
)

type TimelineItem struct {
	Type      string
	CreatedAt time.Time
	Comment   *Comment
	Review    *Review
	Commit    *Commit
}

func LoadPR(prNumber int) (PullRequest, error) {
	repo, err := GetCurrentRepo()
	if err != nil {
		return PullRequest{}, fmt.Errorf("error getting repository information: %w", err)
	}

	client, err := GetRESTClient()
	if err != nil {
		return PullRequest{}, fmt.Errorf("error creating GitHub client: %w", err)
	}

	if prNumber == 0 {
		prNumber, err = GetCurrentPR(client, repo)
		if err != nil {
			return PullRequest{}, fmt.Errorf("error determining PR number: %w", err)
		}
	}

	pr, err := FetchPR(client, repo, prNumber)
	if err != nil {
		return PullRequest{}, fmt.Errorf("error fetching PR #%d: %w", prNumber, err)
	}

	comments, err := FetchPRComments(client, repo, prNumber)
	if err != nil {
		return PullRequest{}, fmt.Errorf("error fetching comments for PR #%d: %w", prNumber, err)
	}
	pr.Comments = comments

	reviews, err := FetchPRReviews(client, repo, prNumber)
	if err != nil {
		return PullRequest{}, fmt.Errorf("error fetching reviews for PR #%d: %w", prNumber, err)
	}

	reviewComments, err := FetchAllReviewComments(client, repo, prNumber)
	if err != nil {
		return PullRequest{}, fmt.Errorf("error fetching review comments for PR #%d: %w", prNumber, err)
	}

	threads := groupIntoThreads(reviewComments)
	threadsByReview := make(map[int64][]CommentThread)
	for _, thread := range threads {
		if len(thread.Comments) > 0 {
			reviewID := thread.Comments[0].PullRequestReviewID
			threadsByReview[reviewID] = append(threadsByReview[reviewID], thread)
		}
	}

	replyCountByReview := make(map[int64]int)
	for _, c := range reviewComments {
		if c.InReplyToID != nil {
			replyCountByReview[c.PullRequestReviewID]++
		}
	}

	for i := range reviews {
		reviews[i].Threads = threadsByReview[reviews[i].ID]
		reviews[i].ReplyCount = replyCountByReview[reviews[i].ID]
	}
	pr.Reviews = reviews

	commits, err := FetchCommits(client, repo, prNumber)
	if err != nil {
		return PullRequest{}, fmt.Errorf("error fetching commits for PR #%d: %w", prNumber, err)
	}
	for i := range commits {
		commits[i].Checks = FetchCommitChecks(client, repo, commits[i].SHA)
	}
	pr.Commits = commits

	return pr, nil
}

func groupIntoThreads(comments []Comment) []CommentThread {
	commentByID := make(map[int64]*Comment)
	for i := range comments {
		commentByID[comments[i].ID] = &comments[i]
	}

	rootToReplies := make(map[int64][]Comment)
	var rootIDs []int64

	for _, c := range comments {
		if c.InReplyToID == nil {
			rootIDs = append(rootIDs, c.ID)
			rootToReplies[c.ID] = []Comment{c}
		} else {
			rootID := *c.InReplyToID
			for {
				parent, ok := commentByID[rootID]
				if !ok || parent.InReplyToID == nil {
					break
				}
				rootID = *parent.InReplyToID
			}
			rootToReplies[rootID] = append(rootToReplies[rootID], c)
		}
	}

	sort.Slice(rootIDs, func(i, j int) bool {
		return commentByID[rootIDs[i]].CreatedAt.Before(commentByID[rootIDs[j]].CreatedAt)
	})

	var threads []CommentThread
	for _, rootID := range rootIDs {
		replies := rootToReplies[rootID]
		sort.Slice(replies, func(i, j int) bool {
			return replies[i].CreatedAt.Before(replies[j].CreatedAt)
		})
		threads = append(threads, CommentThread{Comments: replies})
	}

	return threads
}

func RenderPR(w io.Writer, pr PullRequest) error {
	headerTmpl := `PR #{{ .Number }}: {{ .Title }}
Author: {{ .User.Login }}
Created: {{ .CreatedAt.Format "2006-01-02 15:04:05" }}

{{ .Body }}
`
	tmpl, err := template.New("pr-header").Parse(headerTmpl)
	if err != nil {
		return fmt.Errorf("error creating template: %w", err)
	}

	err = tmpl.Execute(w, pr)
	if err != nil {
		return fmt.Errorf("error rendering PR header: %w", err)
	}

	fmt.Fprintln(w, strings.Repeat("-", 80))

	var timeline []TimelineItem

	for i := range pr.Comments {
		comment := pr.Comments[i]
		timeline = append(timeline, TimelineItem{
			Type:      "comment",
			CreatedAt: comment.CreatedAt,
			Comment:   &comment,
		})
	}

	for i := range pr.Reviews {
		review := pr.Reviews[i]
		timeline = append(timeline, TimelineItem{
			Type:      "review",
			CreatedAt: review.SubmittedAt,
			Review:    &review,
		})
	}

	for i := range pr.Commits {
		commit := pr.Commits[i]
		timeline = append(timeline, TimelineItem{
			Type:      "commit",
			CreatedAt: commit.CreatedAt,
			Commit:    &commit,
		})
	}

	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].CreatedAt.Before(timeline[j].CreatedAt)
	})

	for _, item := range timeline {
		if item.Type == "comment" {
			renderIssueComment(w, *item.Comment)
		} else if item.Type == "review" {
			renderReview(w, *item.Review)
		} else if item.Type == "commit" {
			renderCommit(w, *item.Commit)
		}
		fmt.Fprintln(w, strings.Repeat("-", 80))
	}

	return nil
}

func renderIssueComment(w io.Writer, comment Comment) {
	fmt.Fprintf(w, "%s COMMENTED at %s\n\n", comment.User.Login, comment.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintln(w, comment.Body)
}

func renderReview(w io.Writer, review Review) {
	fmt.Fprintf(w, "%s %s at %s", review.User.Login, review.State, review.SubmittedAt.Format("2006-01-02 15:04:05"))

	if review.Body == "" && len(review.Threads) == 0 && review.ReplyCount > 0 {
		noun := "comments"
		if review.ReplyCount == 1 {
			noun = "comment"
		}
		fmt.Fprintf(w, " (%d %s under existing threads)\n", review.ReplyCount, noun)
		return
	}
	fmt.Fprintln(w)

	if review.Body != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, review.Body)
	}

	for _, thread := range review.Threads {
		fmt.Fprintln(w)
		renderThread(w, thread)
	}
}

func renderCommit(w io.Writer, commit Commit) {
	shortSHA := commit.SHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	author := commit.Author.Login
	if author == "" {
		author = "unknown"
	}

	fmt.Fprintf(w, "%s COMMITTED %s: %s", author, shortSHA, commit.Message)

	c := commit.Checks
	total := c.Succeeded + c.Failed + c.Pending + c.Skipped
	if total > 0 {
		var parts []string
		if c.Succeeded > 0 {
			parts = append(parts, fmt.Sprintf("%d succeeded", c.Succeeded))
		}
		if c.Failed > 0 {
			parts = append(parts, fmt.Sprintf("%d failed", c.Failed))
		}
		if c.Pending > 0 {
			parts = append(parts, fmt.Sprintf("%d pending", c.Pending))
		}
		if c.Skipped > 0 {
			parts = append(parts, fmt.Sprintf("%d skipped", c.Skipped))
		}
		fmt.Fprintf(w, " [%s]", strings.Join(parts, ", "))
	}
	fmt.Fprintln(w)
}

func renderThread(w io.Writer, thread CommentThread) {
	if len(thread.Comments) == 0 {
		return
	}

	root := thread.Comments[0]

	if root.DiffHunk != "" {
		fmt.Fprintf(w, "  %s", root.Path)
		if root.CommitID != "" {
			shortCommit := root.CommitID
			if len(shortCommit) > 7 {
				shortCommit = shortCommit[:7]
			}
			fmt.Fprintf(w, " @ %s", shortCommit)
		}
		isOutdated := root.Line == nil && root.OriginalLine != nil
		if isOutdated {
			fmt.Fprintf(w, " [outdated]")
		}
		fmt.Fprintln(w)
		diffLines := strings.Split(root.DiffHunk, "\n")
		for _, line := range diffLines {
			fmt.Fprintf(w, "    %s\n", line)
		}
	}

	for _, comment := range thread.Comments {
		fmt.Fprintf(w, "  @%s at %s:\n", comment.User.Login, comment.CreatedAt.Format("2006-01-02 15:04:05"))
		bodyLines := strings.Split(comment.Body, "\n")
		for _, line := range bodyLines {
			fmt.Fprintf(w, "    %s\n", line)
		}
		fmt.Fprintln(w)
	}
}
