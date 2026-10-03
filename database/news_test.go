package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeNews inserts a news post and returns it.
func makeNews(t *testing.T, title string, date time.Time) models.News {
	t.Helper()
	news := models.News{Title: title, Body: "body", Date: date, Enabled: true}
	news.ID = uuid.New()
	if err := Instance.Create(&news).Error; err != nil {
		t.Fatalf("failed to create news: %v", err)
	}
	return news
}

func TestGetNewsPosts(t *testing.T) {
	newTestDB(t)

	older := makeNews(t, "Older", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	newer := makeNews(t, "Newer", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))

	// A disabled post must be excluded.
	disabled := makeNews(t, "Gone", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC))
	if err := DeleteNewsPost(disabled.ID); err != nil {
		t.Fatalf("DeleteNewsPost returned error: %v", err)
	}

	posts, err := GetNewsPosts()
	if err != nil {
		t.Fatalf("GetNewsPosts returned error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("got %d posts, want 2", len(posts))
	}
	// Ordered by date desc: newer first.
	if posts[0].ID != newer.ID || posts[1].ID != older.ID {
		t.Errorf("posts not ordered by date desc: got %q then %q", posts[0].Title, posts[1].Title)
	}
}

func TestGetNewsPostByNewsID(t *testing.T) {
	newTestDB(t)

	news := makeNews(t, "Headline", time.Now())

	found, err := GetNewsPostByNewsID(news.ID)
	if err != nil {
		t.Fatalf("GetNewsPostByNewsID returned error: %v", err)
	}
	if found.Title != "Headline" {
		t.Errorf("news title: got %q, want %q", found.Title, "Headline")
	}

	missing, err := GetNewsPostByNewsID(uuid.New())
	if err != nil || missing != nil {
		t.Errorf("unknown news id: got (%v, %v), want (nil, nil)", missing, err)
	}
}

func TestDeleteNewsPost(t *testing.T) {
	newTestDB(t)

	news := makeNews(t, "Temp", time.Now())

	if err := DeleteNewsPost(news.ID); err != nil {
		t.Fatalf("DeleteNewsPost returned error: %v", err)
	}

	// Now soft-deleted: lookup fails.
	if deleted, err := GetNewsPostByNewsID(news.ID); err != nil || deleted != nil {
		t.Errorf("expected disabled news post to be unfindable, got (%v, %v)", deleted, err)
	}

	// Deleting a non-existent post affects no rows and errors.
	if err := DeleteNewsPost(uuid.New()); err == nil {
		t.Errorf("expected error deleting unknown news post")
	}
}
