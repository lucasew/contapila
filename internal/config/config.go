package config

import (
	"embed"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

//go:embed prelude.cue
var fs embed.FS

const PreludeFilename = "prelude.cue"

type Config struct {
	Value cue.Value
}

func Load(userCue []byte, userFilename string) (*Config, error) {
	ctx := cuecontext.New()

	preludeBytes, err := fs.ReadFile(PreludeFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded prelude: %w", err)
	}

	prelude := ctx.CompileBytes(preludeBytes, cue.Filename(PreludeFilename))
	if err := prelude.Err(); err != nil {
		return nil, fmt.Errorf("failed to compile prelude: %w", err)
	}

	user := ctx.CompileBytes(userCue, cue.Filename(userFilename))
	if err := user.Err(); err != nil {
		return nil, fmt.Errorf("failed to compile user config: %w", err)
	}

	unified := prelude.Unify(user)
	if err := unified.Validate(); err != nil {
		return nil, fmt.Errorf("config unification failed: %w", err)
	}

	return &Config{Value: unified}, nil
}
