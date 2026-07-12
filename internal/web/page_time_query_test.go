package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/lucasew/contapila-go/internal/engine"
)

func TestParsePageTimeQuery(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	t.Run("time wins over from/to", func(t *testing.T) {
		q := url.Values{"time": {"2024"}, "from": {"2023"}, "to": {"2023"}}
		got, err := parsePageTimeQuery(q, now, true, true)
		if err != nil {
			t.Fatal(err)
		}
		if got.Time != "2024" {
			t.Fatalf("Time=%q want 2024", got.Time)
		}
	})

	t.Run("from/to composition when fromTo enabled", func(t *testing.T) {
		q := url.Values{"from": {"2024-01"}, "to": {"2024-03"}}
		got, err := parsePageTimeQuery(q, now, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if got.Time != "2024-01 - 2024-03" {
			t.Fatalf("Time=%q", got.Time)
		}
	})

	t.Run("from/to ignored when fromTo disabled", func(t *testing.T) {
		q := url.Values{"from": {"2024-01"}, "to": {"2024-03"}}
		got, err := parsePageTimeQuery(q, now, false, false)
		if err != nil {
			t.Fatal(err)
		}
		if got.Time != "" {
			t.Fatalf("Time=%q want empty", got.Time)
		}
		if !got.AsOf.Equal(engine.AsOfLatest) {
			t.Fatalf("AsOf=%v want AsOfLatest", got.AsOf)
		}
		if got.AsOfStr != "" {
			t.Fatalf("AsOfStr=%q want empty", got.AsOfStr)
		}
	})

	t.Run("from only", func(t *testing.T) {
		q := url.Values{"from": {"2024-01"}}
		got, err := parsePageTimeQuery(q, now, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if got.Time != "2024-01" {
			t.Fatalf("Time=%q", got.Time)
		}
	})

	t.Run("to only", func(t *testing.T) {
		q := url.Values{"to": {"2024-03"}}
		got, err := parsePageTimeQuery(q, now, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if got.Time != "2024-03" {
			t.Fatalf("Time=%q", got.Time)
		}
	})

	t.Run("as-of from period end", func(t *testing.T) {
		q := url.Values{"time": {"2024-03"}}
		got, err := parsePageTimeQuery(q, now, false, false)
		if err != nil {
			t.Fatal(err)
		}
		wantEnd := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
		if !got.AsOf.Equal(wantEnd) {
			t.Fatalf("AsOf=%v want %v", got.AsOf, wantEnd)
		}
		if got.AsOfStr != "2024-03-31" {
			t.Fatalf("AsOfStr=%q", got.AsOfStr)
		}
	})

	t.Run("explicit as-of wins", func(t *testing.T) {
		q := url.Values{"time": {"2024-03"}, "as-of": {"2024-02-10"}}
		got, err := parsePageTimeQuery(q, now, true, true)
		if err != nil {
			t.Fatal(err)
		}
		want := time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC)
		if !got.AsOf.Equal(want) {
			t.Fatalf("AsOf=%v want %v", got.AsOf, want)
		}
		if got.AsOfStr != "2024-02-10" {
			t.Fatalf("AsOfStr=%q", got.AsOfStr)
		}
	})

	t.Run("explicit as-of ignored when disabled", func(t *testing.T) {
		q := url.Values{"time": {"2024-03"}, "as-of": {"2024-02-10"}}
		got, err := parsePageTimeQuery(q, now, false, false)
		if err != nil {
			t.Fatal(err)
		}
		wantEnd := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
		if !got.AsOf.Equal(wantEnd) {
			t.Fatalf("AsOf=%v want period end %v", got.AsOf, wantEnd)
		}
	})

	t.Run("invalid as-of", func(t *testing.T) {
		q := url.Values{"as-of": {"not-a-date"}}
		_, err := parsePageTimeQuery(q, now, true, true)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty filter uses AsOfLatest", func(t *testing.T) {
		got, err := parsePageTimeQuery(url.Values{}, now, true, true)
		if err != nil {
			t.Fatal(err)
		}
		if !got.AsOf.Equal(engine.AsOfLatest) {
			t.Fatalf("AsOf=%v want AsOfLatest", got.AsOf)
		}
		if got.AsOfStr != "" {
			t.Fatalf("AsOfStr=%q", got.AsOfStr)
		}
	})
}
