// Command hevyseed fetches all non-custom Hevy exercise templates and writes them as
// JSON to stdout, sorted by title. It is a one-off developer tool used to generate the
// seed catalog baked into database/seed.go — it is NOT part of the running application.
//
// Usage:
//
//	HEVY_API_KEY=your_pro_key go run ./scripts/hevyseed > scripts/hevyseed/templates.json
//
// (a Hevy PRO API key is required; find it under Settings in the Hevy app).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const baseURL = "https://api.hevyapp.com/v1"

type template struct {
	ID                    string   `json:"id"`
	Title                 string   `json:"title"`
	Type                  string   `json:"type"`
	PrimaryMuscleGroup    string   `json:"primary_muscle_group"`
	SecondaryMuscleGroups []string `json:"secondary_muscle_groups"`
	IsCustom              bool     `json:"is_custom"`
}

type templatesResponse struct {
	Page              int        `json:"page"`
	PageCount         int        `json:"page_count"`
	ExerciseTemplates []template `json:"exercise_templates"`
}

func main() {
	keyFlag := flag.String("key", "", "Hevy PRO API key (defaults to HEVY_API_KEY env)")
	flag.Parse()

	key := strings.TrimSpace(*keyFlag)
	if key == "" {
		key = strings.TrimSpace(os.Getenv("HEVY_API_KEY"))
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "error: provide a key via -key or HEVY_API_KEY")
		os.Exit(1)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var all []template

	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/exercise_templates?page=%d&pageSize=100", baseURL, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error building request:", err)
			os.Exit(1)
		}
		req.Header.Set("api-key", key)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error calling Hevy:", err)
			os.Exit(1)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "error: Hevy returned %d: %s\n", resp.StatusCode, strings.TrimSpace(string(body)))
			os.Exit(1)
		}

		var r templatesResponse
		if err := json.Unmarshal(body, &r); err != nil {
			fmt.Fprintln(os.Stderr, "error parsing response:", err)
			os.Exit(1)
		}

		for _, t := range r.ExerciseTemplates {
			if !t.IsCustom {
				all = append(all, t)
			}
		}
		fmt.Fprintf(os.Stderr, "fetched page %d/%d (%d non-custom so far)\n", r.Page, r.PageCount, len(all))

		if page >= r.PageCount || r.PageCount == 0 {
			break
		}
	}

	sort.Slice(all, func(i, j int) bool { return strings.ToLower(all[i].Title) < strings.ToLower(all[j].Title) })

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(all); err != nil {
		fmt.Fprintln(os.Stderr, "error writing JSON:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "done: %d non-custom templates\n", len(all))
}
