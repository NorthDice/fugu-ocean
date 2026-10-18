package config

import "errors"

var (
	ErrReadFile      = errors.New("config: failed to read file")
	ErrParseFile     = errors.New("config: failed to parse file")
	ErrInvalidConfig = errors.New("config: invalid configuration")
)
