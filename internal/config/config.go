package config

import (
	_ "embed"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

//go:embed prelude.cue
var Prelude string

type Config struct {
	OperatingCurrency string `json:"operating_currency"`
	DefaultPrecision  int    `json:"default_precision"`
}

func Load(userCUE string) (*Config, error) {
	ctx := cuecontext.New()

	// Unify prelude and user CUE
	v := ctx.CompileString(Prelude, cue.Filename("prelude.cue"))
	if v.Err() != nil {
		return nil, fmt.Errorf("error compiling prelude: %w", v.Err())
	}

	if userCUE != "" {
		uv := ctx.CompileString(userCUE, cue.Filename("contapila.cue"))
		if uv.Err() != nil {
			return nil, fmt.Errorf("error compiling user cue: %w", uv.Err())
		}
		v = v.Unify(uv)
	}

	if err := v.Validate(); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	var cfg Config
	if err := v.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("error decoding config: %w", err)
	}

	return &cfg, nil
}
