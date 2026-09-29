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
	if err := fetchTemplates(client, baseURL, key, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// fetchTemplates pages through the exercise templates at apiBaseURL, keeps the non-custom
// ones, and writes them to out as indented JSON sorted by title. Progress goes to log.
func fetchTemplates(client *http.Client, apiBaseURL string, key string, out io.Writer, log io.Writer) error {
	var all []template

	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/exercise_templates?page=%d&pageSize=100", apiBaseURL, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("api-key", key)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("calling Hevy: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Hevy returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var r templatesResponse
		if err := json.Unmarshal(body, &r); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		for _, t := range r.ExerciseTemplates {
			if !t.IsCustom {
				all = append(all, t)
			}
		}
		fmt.Fprintf(log, "fetched page %d/%d (%d non-custom so far)\n", r.Page, r.PageCount, len(all))

		if page >= r.PageCount || r.PageCount == 0 {
			break
		}
	}

	sort.Slice(all, func(i, j int) bool { return strings.ToLower(all[i].Title) < strings.ToLower(all[j].Title) })

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(all); err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}
	fmt.Fprintf(log, "done: %d non-custom templates\n", len(all))
	return nil
}
