package cueutil

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

type Unifier struct {
	ctx *cue.Context
}

func NewUnifier() *Unifier {
	return &Unifier{
		ctx: cuecontext.New(),
	}
}

func (u *Unifier) Unify(schemas []string, datas []any) (cue.Value, error) {
	v := u.ctx.CompileString("")
	for _, s := range schemas {
		v = v.Unify(u.ctx.CompileString(s))
		if v.Err() != nil {
			return v, v.Err()
		}
	}
	for _, d := range datas {
		v = v.Unify(u.ctx.Encode(d))
		if v.Err() != nil {
			return v, v.Err()
		}
	}
	if err := v.Validate(); err != nil {
		return v, err
	}
	return v, nil
}

func (u *Unifier) Decode(v cue.Value, out any) error {
	return v.Decode(out)
}

func FormatError(err error) string {
	return fmt.Sprintf("CUE error: %v", err)
}
