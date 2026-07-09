package config

import (
	"testing"
	"cuelang.org/go/cue"
)

func TestLoad(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		cfg, err := Load([]byte("{}"), "test.cue")
		if err != nil {
			t.Fatalf("failed to load empty config: %v", err)
		}

		val := cfg.Value.LookupPath(cue.ParsePath("commodities"))
		if !val.Exists() {
			t.Errorf("expected commodities to exist in unified config")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		_, err := Load([]byte("invalid cue"), "test.cue")
		if err == nil {
			t.Errorf("expected error for invalid CUE, got nil")
		}
	})

	t.Run("override precision", func(t *testing.T) {
		user := `
commodities: {
	BRL: { precision: 2 }
}
`
		cfg, err := Load([]byte(user), "test.cue")
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		precision := cfg.Value.LookupPath(cue.ParsePath("commodities.BRL.precision"))
		v, err := precision.Int64()
		if err != nil {
			t.Fatalf("failed to get precision as int: %v", err)
		}
		if v != 2 {
			t.Errorf("expected precision 2, got %d", v)
		}

		// Check default precision for another commodity
		// Note: CUE open maps will only have the key if it's explicitly set or if we access it.
		// Since commodities: [string]: #Commodity, it's an open map.
		// To check what USD WOULD have, we can fill it with an empty struct.
		usd := cfg.Value.FillPath(cue.ParsePath("commodities.USD"), map[string]interface{}{})
		usdPrecision := usd.LookupPath(cue.ParsePath("commodities.USD.precision"))
		v, err = usdPrecision.Int64()
		if err != nil {
			t.Fatalf("failed to get default precision for USD: %v", err)
		}
		if v != 5 {
			t.Errorf("expected default precision 5 for USD, got %d", v)
		}
	})
}
