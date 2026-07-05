package controllers

import (
	"sort"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// exerciseHasSoundtrack reports whether any listening history is matched to the
// session. It is a cheap presence check used by the operation-level read path
// (assembleSingleActivity), which — unlike the day-tree list path — doesn't already
// carry the session's media. It short-circuits to false when Media is disabled, so
// no query runs on installs without the feature.
func exerciseHasSoundtrack(exerciseID uuid.UUID) bool {
	if !files.ConfigFile.Media.Enabled {
		return false
	}
	playback, err := database.GetMediaPlaybackForExercise(exerciseID)
	if err != nil {
		return false
	}
	return len(playback) > 0
}

// assembleWorkoutSoundtrack returns the listening history matched to the session that
// owns the given activity (operation). The soundtrack is a session-level fact, so any
// activity id of the session yields the same tracks. It mirrors assembleWorkoutStreams:
// fetched on demand (it can be long) and returns HasSoundtrack=false with a message
// when the feature is off or nothing was matched, rather than an error.
func assembleWorkoutSoundtrack(userID uuid.UUID, activityID uuid.UUID) (models.MCPWorkoutSoundtrack, error) {
	if !files.ConfigFile.Media.Enabled {
		return models.MCPWorkoutSoundtrack{
			HasSoundtrack: false,
			Message:       "Media integration is disabled on this server.",
		}, nil
	}

	// Ownership check: this resolves the activity only if it belongs to the user.
	operation, err := database.GetOperationByIDAndUserID(activityID, userID)
	if err != nil {
		return models.MCPWorkoutSoundtrack{}, err
	}

	playback, err := database.GetMediaPlaybackForExercise(operation.ExerciseID)
	if err != nil {
		return models.MCPWorkoutSoundtrack{}, err
	}

	out := models.MCPWorkoutSoundtrack{HasSoundtrack: len(playback) > 0}
	if exercise, err := database.GetExerciseByIDAndUserID(operation.ExerciseID, userID); err == nil && exercise != nil {
		out.RetrievedAt = exercise.MediaRetrievedAt
	}

	if len(playback) == 0 {
		out.Message = "No listening history is matched to this session. Either nothing was playing during the workout window, no media provider is connected, or the session hasn't been synced yet."
		return out, nil
	}

	for _, item := range playback {
		out.Tracks = append(out.Tracks, models.MCPSoundtrackTrack{
			Type:               item.MediaType,
			Title:              item.Title,
			Artist:             item.Artist,
			Album:              item.Album,
			Provider:           item.Provider,
			StartedAt:          item.StartedAt,
			EndedAt:            item.EndedAt,
			TrackLengthSeconds: item.TrackLength,
		})
	}

	// Play order: earliest first, so the timeline reads top-to-bottom.
	sort.Slice(out.Tracks, func(i, j int) bool {
		return out.Tracks[i].StartedAt.Before(out.Tracks[j].StartedAt)
	})

	return out, nil
}
