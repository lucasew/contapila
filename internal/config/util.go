package config

import "cuelang.org/go/cue"

func ParsePath(s string) cue.Path {
	return cue.ParsePath(s)
}
