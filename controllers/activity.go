package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// activityFeedSorts is the whitelist of sort keys the feed accepts. Anything else is a 400,
// so the DB layer never sees an unmapped column.
var activityFeedSorts = map[string]bool{
	"date":     true,
	"distance": true,
	"duration": true,
	"weight":   true,
	"reps":     true,
}

// parseActivityFeedTime accepts either a full RFC3339 timestamp or a bare YYYY-MM-DD date
// (treated as that day at midnight), which is what the date-range inputs on the page send.
func parseActivityFeedTime(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	return time.Parse("2006-01-02", value)
}

// APIGetActivityFeed powers the /exercises timeline: a filtered, sorted, paginated list of
// activities (operations) with per-activity metrics aggregated from their sets. See
// database.GetActivityFeedForUser and docs/wip.md.
func APIGetActivityFeed(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	filter := models.ActivityFeedFilter{
		Sort:   "date",
		Order:  "desc",
		Limit:  30,
		Offset: 0,
	}

	if value := strings.TrimSpace(context.Query("action_id")); value != "" {
		actionID, err := uuid.Parse(value)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action id."})
			context.Abort()
			return
		}
		filter.ActionID = &actionID
	}

	if value := strings.TrimSpace(context.Query("start")); value != "" {
		parsed, err := parseActivityFeedTime(value)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date."})
			context.Abort()
			return
		}
		filter.Start = &parsed
	}

	if value := strings.TrimSpace(context.Query("end")); value != "" {
		parsed, err := parseActivityFeedTime(value)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date."})
			context.Abort()
			return
		}
		filter.End = &parsed
	}

	filter.Query = strings.TrimSpace(context.Query("q"))

	if strings.EqualFold(strings.TrimSpace(context.Query("has_distance")), "true") {
		filter.HasDistance = true
	}

	if value := strings.ToLower(strings.TrimSpace(context.Query("sort"))); value != "" {
		if !activityFeedSorts[value] {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sort."})
			context.Abort()
			return
		}
		filter.Sort = value
	}

	if value := strings.ToLower(strings.TrimSpace(context.Query("order"))); value != "" {
		if value != "asc" && value != "desc" {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order."})
			context.Abort()
			return
		}
		filter.Order = value
	}

	if value := strings.TrimSpace(context.Query("limit")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			filter.Limit = parsed
		}
	}
	if filter.Limit < 1 {
		filter.Limit = 1
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if value := strings.TrimSpace(context.Query("offset")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			filter.Offset = parsed
		}
	}

	items, total, err := database.GetActivityFeedForUser(userID, filter)
	if err != nil {
		logger.Log.Info("Failed to get activity feed. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get activities."})
		context.Abort()
		return
	}

	if items == nil {
		items = []models.ActivityFeedItem{}
	}
	hasMore := int64(filter.Offset+len(items)) < total

	context.JSON(http.StatusOK, gin.H{
		"message":    "Activities retrieved.",
		"activities": items,
		"total":      total,
		"has_more":   hasMore,
	})
}
