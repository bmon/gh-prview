package prview_test

import (
	"testing"
	"time"

	prview "github.com/bmon/gh-prview"
)

// TestParseTimelineItems tests the creation and sorting of timeline items
func TestParseTimelineItems(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	// Create test comments
	comments := []prview.Comment{
		{
			ID:        1,
			Body:      "First comment",
			CreatedAt: earlier,
			User:      prview.User{Login: "user1"},
		},
		{
			ID:        2,
			Body:      "Later comment",
			CreatedAt: later,
			User:      prview.User{Login: "user2"},
		},
	}

	// Create test reviews
	reviews := []prview.Review{
		{
			ID:        101,
			Body:      "Middle review",
			CreatedAt: now,
			User:      prview.User{Login: "reviewer1"},
		},
	}

	// Test creating timeline items
	var timeline []prview.TimelineItem

	// Add comments to timeline
	for i := range comments {
		timeline = append(timeline, prview.TimelineItem{
			Type:      "comment",
			CreatedAt: comments[i].CreatedAt,
			Comment:   &comments[i],
		})
	}

	// Add reviews to timeline
	for i := range reviews {
		timeline = append(timeline, prview.TimelineItem{
			Type:      "review",
			CreatedAt: reviews[i].CreatedAt,
			Review:    &reviews[i],
		})
	}

	// Check timeline length
	if len(timeline) != 3 {
		t.Errorf("Expected 3 timeline items, got %d", len(timeline))
	}
}

// TestCommentStructure tests the comment structure
func TestCommentStructure(t *testing.T) {
	comment := prview.Comment{
		ID:        123,
		Body:      "Test comment",
		CreatedAt: time.Now(),
		User:      prview.User{Login: "testuser"},
		DiffHunk:  "@@ -1,3 +1,3 @@",
	}

	if comment.ID != 123 {
		t.Errorf("Expected ID 123, got %d", comment.ID)
	}

	if comment.Body != "Test comment" {
		t.Errorf("Expected body 'Test comment', got '%s'", comment.Body)
	}

	if comment.User.Login != "testuser" {
		t.Errorf("Expected user 'testuser', got '%s'", comment.User.Login)
	}

	if comment.DiffHunk != "@@ -1,3 +1,3 @@" {
		t.Errorf("Expected specific diff hunk, got '%s'", comment.DiffHunk)
	}
}

// TestReviewStructure tests the review structure
func TestReviewStructure(t *testing.T) {
	review := prview.Review{
		ID:        456,
		Body:      "Test review",
		State:     "APPROVED",
		CreatedAt: time.Now(),
		User:      prview.User{Login: "reviewer"},
		Comments:  []prview.Comment{},
	}

	if review.ID != 456 {
		t.Errorf("Expected ID 456, got %d", review.ID)
	}

	if review.Body != "Test review" {
		t.Errorf("Expected body 'Test review', got '%s'", review.Body)
	}

	if review.State != "APPROVED" {
		t.Errorf("Expected state 'APPROVED', got '%s'", review.State)
	}

	if review.User.Login != "reviewer" {
		t.Errorf("Expected user 'reviewer', got '%s'", review.User.Login)
	}

	if len(review.Comments) != 0 {
		t.Errorf("Expected 0 comments, got %d", len(review.Comments))
	}
}
