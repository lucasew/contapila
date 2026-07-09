package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("empty user config", func(t *testing.T) {
		cfg, err := Load("")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DefaultPrecision != 5 {
			t.Errorf("got precision %d, want 5", cfg.DefaultPrecision)
		}
	})

	t.Run("override precision", func(t *testing.T) {
		cfg, err := Load("default_precision: 2")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DefaultPrecision != 2 {
			t.Errorf("got precision %d, want 2", cfg.DefaultPrecision)
		}
	})

	t.Run("invalid cue", func(t *testing.T) {
		_, err := Load("invalid syntax")
		if err == nil {
			t.Error("expected error for invalid cue")
		}
	})
}
