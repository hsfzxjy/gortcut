package main

import (
	_ "embed"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func ParseConfig(fp string) ([]*Job, error) {
	var r = cuecontext.New()
	s := r.CompileString(schemaCue)
	if s.Err() != nil {
		return nil, s.Err()
	}
	content, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	v := r.CompileBytes(content)
	if v.Err() != nil {
		return nil, v.Err()
	}
	u := s.Unify(v)
	if u.Err() != nil {
		return nil, u.Err()
	}
	if err := u.Validate(cue.Concrete(true), cue.Hidden(true)); err != nil {
		return nil, err
	}
	var jobs []*Job
	err = u.LookupPath(cue.ParsePath("jobs")).Decode(&jobs)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

//go:embed schema.cue
var schemaCue string
