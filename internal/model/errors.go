package model

import "errors"

var ErrRepoNotFound = errors.New("not inside a GitHub repo")

type GhError struct {
	Msg    string
	Stderr string
}

func (e *GhError) Error() string { return e.Msg }

type UsageError struct {
	Msg string
}

func (e *UsageError) Error() string { return e.Msg }
