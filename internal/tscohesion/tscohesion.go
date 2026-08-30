package tscohesion

import "boy-scout/internal/srcfiles"

type Violation struct {
	File      string  `json:"file"`
	Line      int     `json:"line"`
	Class     string  `json:"class"`
	LCOM4     int     `json:"lcom4"`
	LCOM4Level string `json:"lcom4Level"`
	TCC       float64 `json:"tcc"`
	TCCLevel  string  `json:"tccLevel"`
	LCC       float64 `json:"lcc"`
	LCCLevel  string  `json:"lccLevel"`
}

type SkippedFile = srcfiles.SkippedFile

type Options struct {
	ExcludeFiles []string
	Debug        bool
}

type Report struct {
	Violations []Violation   `json:"violations"`
	Skipped    []SkippedFile `json:"skipped"`
}

func Check(paths []string, opts Options) (Report, error) {
	// TODO: implement TypeScript cohesion checking
	return Report{
		Violations: []Violation{},
		Skipped:    []SkippedFile{},
	}, nil
}
