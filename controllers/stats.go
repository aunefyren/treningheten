package controllers

import (
	"math"
	"net/http"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
)

// APIGetAdminStats returns aggregate, non-personal usage statistics for the
// admin panel. "Active user" is defined as an enabled user.
func APIGetAdminStats(context *gin.Context) {
	counts, err := database.GetAdminStatsCounts(time.Now())
	if err != nil {
		logger.Log.Error("Failed to gather admin statistics. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to gather statistics."})
		context.Abort()
		return
	}

	stats := models.AdminStatsReply{
		TotalUsers:             counts.TotalUsers,
		UsersInSeasonNow:       counts.UsersInSeasonNow,
		UsersWithNotifications: counts.UsersWithNotifications,
		UsersWithStrava:        counts.UsersWithStrava,
		AchievementsTotal:      counts.AchievementsTotal,

		UsersInSeasonNowPct:       percentageOf(counts.UsersInSeasonNow, counts.TotalUsers),
		UsersWithNotificationsPct: percentageOf(counts.UsersWithNotifications, counts.TotalUsers),
		UsersWithStravaPct:        percentageOf(counts.UsersWithStrava, counts.TotalUsers),
		// Average fraction of all achievements unlocked across all active users.
		AvgAchievementCompletionPct: percentageOf(counts.AchievementDelegations, counts.TotalUsers*counts.AchievementsTotal),
	}

	context.JSON(http.StatusOK, gin.H{"message": "Statistics retrieved.", "stats": stats})
}

// percentageOf returns part/whole as a percentage rounded to one decimal,
// guarding against a zero denominator.
func percentageOf(part int64, whole int64) float64 {
	if whole <= 0 {
		return 0
	}
	return math.Round(float64(part)/float64(whole)*1000) / 10
}
