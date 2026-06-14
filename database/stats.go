package database

import (
	"time"

	"github.com/aunefyren/treningheten/models"
)

// GetAdminStatsCounts gathers the raw counts behind the admin panel statistics.
// "Active user" is an enabled user (`enabled = 1`); all metrics are scoped to
// enabled rows. `now` decides which seasons are considered ongoing.
func GetAdminStatsCounts(now time.Time) (models.AdminStatsCounts, error) {
	var counts models.AdminStatsCounts

	// Total enabled users
	if record := Instance.Model(&models.User{}).
		Where("`users`.enabled = ?", 1).
		Count(&counts.TotalUsers); record.Error != nil {
		return counts, record.Error
	}

	// Distinct enabled users with an enabled goal in a currently ongoing season
	if record := Instance.Model(&models.Goal{}).
		Joins("JOIN seasons ON `goals`.season_id = `seasons`.ID").
		Joins("JOIN users ON `goals`.user_id = `users`.ID").
		Where("`goals`.enabled = ?", 1).
		Where("`seasons`.enabled = ?", 1).
		Where("`users`.enabled = ?", 1).
		Where("`seasons`.`start` <= ?", now).
		Where("`seasons`.`end` >= ?", now).
		Distinct("`goals`.user_id").
		Count(&counts.UsersInSeasonNow); record.Error != nil {
		return counts, record.Error
	}

	// Distinct enabled users with at least one push subscription.
	// A user can have several subscriptions (one per device/browser), so we
	// must count distinct user_id explicitly — `Distinct(col).Count()` is not
	// reliably translated to COUNT(DISTINCT ...) by GORM and would otherwise
	// count subscription rows, inflating the figure above the user total.
	if record := Instance.Model(&models.Subscription{}).
		Joins("JOIN users ON `subscriptions`.user_id = `users`.ID").
		Where("`users`.enabled = ?", 1).
		Select("COUNT(DISTINCT subscriptions.user_id)").
		Count(&counts.UsersWithNotifications); record.Error != nil {
		return counts, record.Error
	}

	// Enabled users with a Strava connection
	if record := Instance.Model(&models.User{}).
		Where("`users`.enabled = ?", 1).
		Where("`users`.strava_code IS NOT NULL").
		Where("`users`.strava_code != ?", "").
		Count(&counts.UsersWithStrava); record.Error != nil {
		return counts, record.Error
	}

	// Total enabled achievements
	if record := Instance.Model(&models.Achievement{}).
		Where("`achievements`.enabled = ?", 1).
		Count(&counts.AchievementsTotal); record.Error != nil {
		return counts, record.Error
	}

	// Distinct (user, achievement) delegations across enabled users and
	// achievements. `Distinct(a, b).Count()` is not portably translated to a
	// distinct count by GORM (and COUNT(DISTINCT a, b) is MySQL-only), so we
	// count the rows of a DISTINCT subquery instead — correct on every backend.
	distinctDelegations := Instance.Model(&models.AchievementDelegation{}).
		Joins("JOIN users ON `achievement_delegations`.user_id = `users`.ID").
		Joins("JOIN achievements ON `achievement_delegations`.achievement_id = `achievements`.ID").
		Where("`achievement_delegations`.enabled = ?", 1).
		Where("`users`.enabled = ?", 1).
		Where("`achievements`.enabled = ?", 1).
		Distinct("`achievement_delegations`.user_id", "`achievement_delegations`.achievement_id")
	if record := Instance.Table("(?) AS distinct_delegations", distinctDelegations).
		Count(&counts.AchievementDelegations); record.Error != nil {
		return counts, record.Error
	}

	return counts, nil
}
