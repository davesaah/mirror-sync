package main

import (
	"fmt"
	"os"
)

type errOptions struct {
	err  error
	exit bool
}

func newErrOptions() *errOptions {
	return &errOptions{
		exit: false,
	}
}

type ErrOption func(*errOptions)

func WithErr(v error) ErrOption {
	return func(o *errOptions) {
		o.err = v
	}
}

func WithExit(v bool) ErrOption {
	return func(o *errOptions) {
		o.exit = v
	}
}

func check(opts ...ErrOption) {
	o := newErrOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.err != nil {
		fmt.Println(o.err)
		if o.exit {
			os.Exit(1)
		}
	}
}
