package prview

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"
	"time"
)

// TimelineItem combines comments and reviews for chronological sorting
type TimelineItem struct {
	Type      string // "comment" or "review"
	CreatedAt time.Time
	Comment   *Comment
	Review    *Review
}

// LoadPR loads and parses PR data and returns it
func LoadPR(prNumber int) (PullRequest, error) {
	// Get repository information
	repo, err := GetCurrentRepo()
	if err != nil {
		return PullRequest{}, fmt.Errorf("error getting repository information: %w", err)
	}

	// Create API client
	client, err := GetRESTClient()
	if err != nil {
		return PullRequest{}, fmt.Errorf("error creating GitHub client: %w", err)
	}

	// If prNumber is 0, try to determine the current PR
	if prNumber == 0 {
		prNumber, err = GetCurrentPR(client, repo)
		if err != nil {
			return PullRequest{}, fmt.Errorf("error determining PR number: %w", err)
		}
	}

	// Fetch PR details
	pr, err := FetchPR(client, repo, prNumber)
	if err != nil {
		return PullRequest{}, fmt.Errorf("error fetching PR #%d: %w", prNumber, err)
	}

	// Fetch PR comments (issue comments)
	comments, err := FetchPRComments(client, repo, prNumber)
	if err != nil {
		return PullRequest{}, fmt.Errorf("error fetching comments for PR #%d: %w", prNumber, err)
	}
	pr.Comments = comments

	// Fetch PR reviews
	reviews, err := FetchPRReviews(client, repo, prNumber)
	if err != nil {
		return PullRequest{}, fmt.Errorf("error fetching reviews for PR #%d: %w", prNumber, err)
	}

	// Fetch review comments for each review
	for i := range reviews {
		reviewComments, err := FetchReviewComments(client, repo, prNumber, reviews[i].ID)
		if err != nil {
			return PullRequest{}, fmt.Errorf("error fetching review comments for review #%d: %w", reviews[i].ID, err)
		}
		reviews[i].Comments = reviewComments
	}
	pr.Reviews = reviews

	return pr, nil
}

// RenderPR renders a pull request with all its comments and reviews
func RenderPR(w io.Writer, pr PullRequest) error {
	// Render PR header
	headerTmpl := `
PR #{{ .Number }}: {{ .Title }}
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

	// Create a timeline of all comments and reviews sorted by time
	var timeline []TimelineItem

	// Add PR comments to timeline
	for i := range pr.Comments {
		comment := pr.Comments[i]
		timeline = append(timeline, TimelineItem{
			Type:      "comment",
			CreatedAt: comment.CreatedAt,
			Comment:   &comment,
		})
	}

	// Add reviews to timeline
	for i := range pr.Reviews {
		review := pr.Reviews[i]
		timeline = append(timeline, TimelineItem{
			Type:      "review",
			CreatedAt: review.CreatedAt,
			Review:    &review,
		})
	}

	// Sort timeline by created time
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].CreatedAt.Before(timeline[j].CreatedAt)
	})

	// Render each timeline item
	for _, item := range timeline {
		if item.Type == "comment" {
			err = RenderComment(w, *item.Comment, 0)
			if err != nil {
				return fmt.Errorf("error rendering comment: %w", err)
			}
		} else if item.Type == "review" {
			err = RenderReview(w, *item.Review)
			if err != nil {
				return fmt.Errorf("error rendering review: %w", err)
			}
		}
		fmt.Fprintln(w, strings.Repeat("-", 80))
	}

	return nil
}

// RenderComment renders a single comment with its diff if present
func RenderComment(w io.Writer, comment Comment, indent int) error {
	indentStr := strings.Repeat(" ", indent)

	// Template for comment
	commentTmpl := `{{ .indentStr }}Comment by {{ .comment.User.Login }} on {{ .comment.CreatedAt.Format "2006-01-02 15:04:05" }}
{{ .indentStr }}
{{ .bodyIndented }}
`
	tmpl, err := template.New("comment").Parse(commentTmpl)
	if err != nil {
		return fmt.Errorf("error creating comment template: %w", err)
	}

	// Indent each line of the body
	bodyLines := strings.Split(comment.Body, "\n")
	indentedBodyLines := make([]string, len(bodyLines))
	for i, line := range bodyLines {
		indentedBodyLines[i] = indentStr + line
	}
	bodyIndented := strings.Join(indentedBodyLines, "\n")

	// Execute template
	err = tmpl.Execute(w, map[string]interface{}{
		"indentStr":    indentStr,
		"comment":      comment,
		"bodyIndented": bodyIndented,
	})
	if err != nil {
		return fmt.Errorf("error rendering comment: %w", err)
	}

	// If there's a diff hunk, indent and append it
	if comment.DiffHunk != "" {
		fmt.Fprintf(w, "\n%sDiff:\n", indentStr)
		diffLines := strings.Split(comment.DiffHunk, "\n")
		diffIndent := indentStr + "  "
		for _, line := range diffLines {
			fmt.Fprintf(w, "%s%s\n", diffIndent, line)
		}
	}

	return nil
}

// RenderReview renders a review with all its comments
func RenderReview(w io.Writer, review Review) error {
	// Template for review header
	reviewTmpl := `Review by {{ .User.Login }} on {{ .CreatedAt.Format "2006-01-02 15:04:05" }}
State: {{ .State }}
{{ if .Body }}
{{ .Body }}
{{ else }}
(No summary comment)
{{ end }}
`
	tmpl, err := template.New("review").Parse(reviewTmpl)
	if err != nil {
		return fmt.Errorf("error creating review template: %w", err)
	}

	err = tmpl.Execute(w, review)
	if err != nil {
		return fmt.Errorf("error rendering review: %w", err)
	}

	if len(review.Comments) > 0 {
		fmt.Fprintln(w, "\nReview comments:")
		for _, comment := range review.Comments {
			err = RenderComment(w, comment, 2)
			if err != nil {
				return fmt.Errorf("error rendering review comment: %w", err)
			}
		}
	}

	return nil
}
