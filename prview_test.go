package prview_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	prview "github.com/bmon/gh-prview"
)

// createMockPR creates a sample PR object for testing
func createMockPR() prview.PullRequest {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	evenEarlier := now.Add(-2 * time.Hour)

	pr := prview.PullRequest{
		Number:    123,
		Title:     "Test PR",
		Body:      "This is a test PR body",
		CreatedAt: evenEarlier,
	}
	pr.User = prview.User{Login: "testuser"}

	// Add some comments
	pr.Comments = []prview.Comment{
		{
			ID:        1,
			Body:      "This is a regular comment",
			CreatedAt: earlier,
			User:      prview.User{Login: "commenter1"},
		},
		{
			ID:        2,
			Body:      "This is a comment with a diff",
			CreatedAt: now,
			User:      prview.User{Login: "commenter2"},
			DiffHunk:  "@@ -1,5 +1,7 @@\n function test() {\n-  return false;\n+  // Added a comment\n+  return true;\n }",
		},
	}

	// Add a review
	review := prview.Review{
		ID:        101,
		Body:      "Here's my review",
		State:     "APPROVED",
		CreatedAt: earlier.Add(30 * time.Minute),
		User:      prview.User{Login: "reviewer1"},
	}

	// Add review comments
	review.Comments = []prview.Comment{
		{
			ID:        201,
			Body:      "This looks good",
			CreatedAt: earlier.Add(31 * time.Minute),
			User:      prview.User{Login: "reviewer1"},
			DiffHunk:  "@@ -10,4 +10,6 @@\n function another() {\n+  // New function\n+  return 42;\n }",
		},
	}

	pr.Reviews = []prview.Review{review}

	return pr
}

func TestRenderPR(t *testing.T) {
	// Create a mock PR
	pr := createMockPR()

	// Render to a buffer
	var buf bytes.Buffer
	err := prview.RenderPR(&buf, pr)
	if err != nil {
		t.Fatalf("RenderPR returned an error: %v", err)
	}

	output := buf.String()

	// Basic verification
	expectedStrings := []string{
		"PR #123: Test PR",
		"Author: testuser",
		"This is a test PR body",
		"Comment by commenter1",
		"This is a regular comment",
		"Comment by commenter2",
		"This is a comment with a diff",
		"Diff:",
		"@@ -1,5 +1,7 @@",
		"Review by reviewer1",
		"State: APPROVED",
		"Here's my review",
		"This looks good",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain: %s", expected)
		}
	}
}

func TestRenderComment(t *testing.T) {
	// Create a test comment
	comment := prview.Comment{
		ID:        42,
		Body:      "Test comment\nwith multiple lines",
		CreatedAt: time.Now(),
		User:      prview.User{Login: "test-user"},
		DiffHunk:  "@@ -1,3 +1,4 @@\n line1\n+added line\n line2\n line3",
	}

	// Render to a buffer
	var buf bytes.Buffer
	err := prview.RenderComment(&buf, comment, 2)
	if err != nil {
		t.Fatalf("renderComment returned an error: %v", err)
	}

	output := buf.String()

	// Verify content
	expectedStrings := []string{
		"  Comment by test-user",
		"  Test comment",
		"  with multiple lines",
		"  Diff:",
		"    @@ -1,3 +1,4 @@",
		"    line1",
		"    +added line",
		"    line2",
		"    line3",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain: %s", expected)
		}
	}
}

func TestRenderReview(t *testing.T) {
	// Create a test review
	review := prview.Review{
		ID:        101,
		Body:      "Review comment",
		State:     "CHANGES_REQUESTED",
		CreatedAt: time.Now(),
		User:      prview.User{Login: "reviewer"},
		Comments: []prview.Comment{
			{
				ID:        201,
				Body:      "Comment in review",
				CreatedAt: time.Now(),
				User:      prview.User{Login: "reviewer"},
				DiffHunk:  "@@ -5,7 +5,8 @@\n context\n+added\n context",
			},
		},
	}

	// Render to a buffer
	var buf bytes.Buffer
	err := prview.RenderReview(&buf, review)
	if err != nil {
		t.Fatalf("renderReview returned an error: %v", err)
	}

	output := buf.String()

	// Verify content
	expectedStrings := []string{
		"Review by reviewer",
		"State: CHANGES_REQUESTED",
		"Review comment",
		"Review comments:",
		"  Comment by reviewer",
		"  Comment in review",
		"  Diff:",
		"    @@ -5,7 +5,8 @@",
		"    context",
		"    +added",
		"    context",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain: %s", expected)
		}
	}
}
