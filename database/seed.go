package database

import (
	"time"

	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func ptr(s string) *string {
	return &s
}

var defaultActions = []models.Action{
	// --- Cardio / Moving ---
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("2d597949-e670-4a4d-a5e1-e67c3837c8a1"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Exercise",
		NorwegianName: "Trening",
		Description:   "Any type of heart rate elevating activity that makes the body work.",
		Type:          "timing",
		BodyPart:      "general",
		StravaName:    "Workout",
		PastTenseVerb: ptr("worked out"),
		HasLogo:       false,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("a3fe89d4-bbb9-40af-9bd1-e72ec448705c"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Run",
		NorwegianName: "Løping",
		Description:   "Outdoor or treadmill running at any pace, from easy jogs to hard intervals.",
		Type:          "moving",
		BodyPart:      "cardio",
		StravaName:    "Run",
		PastTenseVerb: ptr("went for a run"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("9a9230b0-82d1-4419-9cbe-c8ffe99cb62f"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Bicycling",
		NorwegianName: "Sykling",
		Description:   "Road, gravel or mountain biking - any cycling done outdoors or on a stationary bike.",
		Type:          "moving",
		BodyPart:      "cardio",
		StravaName:    "Ride",
		PastTenseVerb: ptr("went for a bike ride"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("1ec8f181-475a-4544-8902-676c9dbb3bfc"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Hike",
		NorwegianName: "Fottur",
		Description:   "A long walk in nature, typically on trails or in the mountains.",
		Type:          "moving",
		BodyPart:      "cardio",
		StravaName:    "Hike",
		PastTenseVerb: ptr("went for a hike"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("59211f92-3207-46d0-9e2a-33fe27ead513"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Walking",
		NorwegianName: "Tur",
		Description:   "A leisurely walk, whether around the neighborhood or along a longer route.",
		Type:          "moving",
		BodyPart:      "cardio",
		StravaName:    "Walk",
		PastTenseVerb: ptr("went for a walk"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("21a4666d-d1ee-40a2-802c-f09ef3bce895"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Swim",
		NorwegianName: "Svømming",
		Description:   "Pool or open-water swimming, including lap swimming and triathlons.",
		Type:          "moving",
		BodyPart:      "cardio",
		StravaName:    "Swim",
		PastTenseVerb: ptr("went for a swim"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("bb22d5af-a854-49e6-b14c-197ebd0c5dde"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Cross-Country Skiing",
		NorwegianName: "Langrenn",
		Description:   "Classic or skate-style Nordic skiing on prepared or unprepared tracks.",
		Type:          "moving",
		BodyPart:      "cardio",
		StravaName:    "NordicSki",
		PastTenseVerb: ptr("went cross-country skiing"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("157ad372-b375-4b70-bace-349cb422d85b"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Alpine Skiing",
		NorwegianName: "Alpint",
		Description:   "Downhill skiing on groomed pistes or off-piste terrain.",
		Type:          "moving",
		BodyPart:      "cardio",
		StravaName:    "AlpineSki",
		PastTenseVerb: ptr("went alpine skiing"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("4fa35db9-181f-4cc1-b30e-1995c8a479b6"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Snowboard",
		NorwegianName: "Snøbrett",
		Description:   "Riding a snowboard down snow-covered slopes.",
		Type:          "moving",
		BodyPart:      "cardio",
		StravaName:    "Snowboard",
		PastTenseVerb: ptr("went snowboarding"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("067a3de0-0171-43cc-b95e-99da5e07e546"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Elliptical",
		NorwegianName: "Elliptisk maskin",
		Description:   "Low-impact cardio on an elliptical cross-trainer machine.",
		Type:          "moving",
		BodyPart:      "cardio",
		StravaName:    "Elliptical",
		PastTenseVerb: ptr("used the elliptical"),
		HasLogo:       false,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("a460eb20-0fd9-4ca4-af8e-40a880fe6dfd"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Rowing",
		NorwegianName: "Roing",
		Description:   "On-water rowing or ergometer (indoor rowing machine) sessions.",
		Type:          "moving",
		BodyPart:      "back",
		StravaName:    "Rowing",
		PastTenseVerb: ptr("went rowing"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("653e3ae5-4ab0-4b69-bbe5-c8e6c1c24c78"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Skateboard",
		NorwegianName: "Rullebrettkjøring",
		Description:   "Skateboarding on streets, parks or skate parks.",
		Type:          "moving",
		BodyPart:      "cardio",
		StravaName:    "Skateboard",
		PastTenseVerb: ptr("went skateboarding"),
		HasLogo:       true,
	},

	// --- Timing / Sports ---
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("620d9b9e-323d-4892-9718-7f1969ee760a"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Padel",
		NorwegianName: "Padel",
		Description:   "A racket sport combining elements of tennis and squash, played in an enclosed court.",
		Type:          "timing",
		BodyPart:      "cardio",
		StravaName:    "Padel",
		PastTenseVerb: ptr("played some padel"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("63348e1a-4307-4b93-aa79-475f0f4fc2cb"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Tennis",
		NorwegianName: "Tennis",
		Description:   "Racket sport played on a court, either singles or doubles.",
		Type:          "timing",
		BodyPart:      "cardio",
		StravaName:    "",
		PastTenseVerb: ptr("played some tennis"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("d59923b6-c6f3-45c9-9e33-f6c22ae8be18"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Football (Soccer)",
		NorwegianName: "Fotball",
		Description:   "The world's most popular sport - eleven a side on a grass or artificial pitch.",
		Type:          "timing",
		BodyPart:      "cardio",
		StravaName:    "Soccer",
		PastTenseVerb: ptr("played some football"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("67c18faf-1e38-4633-8806-f7bdb2e5d84f"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Golf",
		NorwegianName: "Golf",
		Description:   "Hitting a ball into a series of holes on a course in as few strokes as possible.",
		Type:          "timing",
		BodyPart:      "cardio",
		StravaName:    "Golf",
		PastTenseVerb: ptr("played some golf"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("2ecb9036-5483-4609-bb47-dc375ee88516"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Crossfit",
		NorwegianName: "Crossfit",
		Description:   "High-intensity functional fitness combining weightlifting, gymnastics and cardio.",
		Type:          "timing",
		BodyPart:      "cardio",
		StravaName:    "Crossfit",
		PastTenseVerb: ptr("did some crossfit"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("c8010ef4-6bf2-4429-8fbb-30f430b924a8"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Weight Training",
		NorwegianName: "Styrketrening",
		Description:   "Structured resistance training sessions in the gym.",
		Type:          "timing",
		BodyPart:      "full body",
		StravaName:    "WeightTraining",
		PastTenseVerb: ptr("did some weight training"),
		HasLogo:       true,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("6afc4e91-ed77-4f30-a38a-d10ea2aeba94"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Yoga",
		NorwegianName: "Yoga",
		Description:   "A practice combining physical postures, breathing exercises and meditation.",
		Type:          "timing",
		BodyPart:      "core",
		StravaName:    "Yoga",
		PastTenseVerb: ptr("did some yoga"),
		HasLogo:       true,
	},

	// --- Lifting ---
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("0397d83a-b1ad-41d2-9bfe-2f92f655cfd2"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Bicep Curl",
		NorwegianName: "Bicepscurl",
		Description:   "Isolation exercise targeting the biceps brachii with a dumbbell or barbell.",
		Type:          "lifting",
		BodyPart:      "arms",
		StravaName:    "",
		PastTenseVerb: nil,
		HasLogo:       false,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("21c5abe8-b18c-4233-a2be-aa2b8ec7a1bc"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Low Row",
		NorwegianName: "Sittende roing",
		Description:   "Cable or machine row performed from a seated position, targeting the mid-back.",
		Type:          "lifting",
		BodyPart:      "back",
		StravaName:    "",
		PastTenseVerb: nil,
		HasLogo:       false,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("2f67d91b-b608-4430-a092-b850560c79fb"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Vertical Pulldown (alternative grip)",
		NorwegianName: "Nedtrekk (alternativt grep)",
		Description:   "Lat pulldown with a neutral or underhand grip to vary back muscle recruitment.",
		Type:          "lifting",
		BodyPart:      "back",
		StravaName:    "",
		PastTenseVerb: nil,
		HasLogo:       false,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("665624f9-db55-4fb8-8445-d8db8359f16f"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Vertical Pulldown",
		NorwegianName: "Nedtrekk",
		Description:   "Classic lat pulldown with a wide overhand grip, targeting the latissimus dorsi.",
		Type:          "lifting",
		BodyPart:      "back",
		StravaName:    "",
		PastTenseVerb: nil,
		HasLogo:       false,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("e3b0849b-6cf0-4f07-a8a1-48ee84cbc83d"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Plank",
		NorwegianName: "Planke",
		Description:   "Isometric core exercise held in a push-up position to build stability and endurance.",
		Type:          "lifting",
		BodyPart:      "core",
		StravaName:    "",
		PastTenseVerb: ptr("did the plank"),
		HasLogo:       false,
	},
	{
		GormModel:     models.GormModel{ID: uuid.MustParse("9be08d4e-a5ec-49df-9c5b-718cd5bbf46e"), CreatedAt: time.Now()},
		Enabled:       true,
		Name:          "Pilates",
		NorwegianName: "Pilates",
		Description:   "Low-impact, mind-body exercise method focusing on core strength, flexibility, posture, and body awareness.",
		Type:          "lifting",
		BodyPart:      "core",
		StravaName:    "Pilates",
		PastTenseVerb: ptr("did some pilates"),
		HasLogo:       false,
	},
}

// SeedActions inserts any default actions that are not already present in the database.
// It matches on ID so re-running at startup is safe and idempotent.
func SeedActions() {
	existing, err := GetAllEnabledActions()
	if err != nil {
		logger.Log.Printf("could not fetch existing actions: %v", err)
		return
	}

	existingIDs := make(map[string]models.Action, len(existing))
	for _, a := range existing {
		existingIDs[a.ID.String()] = a
	}

	seeded := 0
	for _, action := range defaultActions {
		existingAction, found := existingIDs[action.ID.String()]
		if found {
			action.Enabled = existingAction.Enabled
			action.CreatedAt = existingAction.CreatedAt
		}
		if _, err := UpdateActionInDB(action); err != nil {
			logger.Log.Printf("failed to insert action %q: %v", action.Name, err)
		} else {
			logger.Log.Printf("inserted action: %s", action.Name)
			seeded++
		}
	}

	if seeded == 0 {
		logger.Log.Println("all default actions already present, nothing to insert")
	} else {
		logger.Log.Printf("inserted %d default action(s)", seeded)
	}
}
