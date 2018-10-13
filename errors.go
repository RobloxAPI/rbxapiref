package main

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"strings"
)

type Errors struct {
	Msg  string
	Errs []error
}

func (err Errors) Error() string {
	s := make([]string, len(err.Errs)+1)
	if err.Msg != "" {
		s[0] = err.Msg
	} else {
		s[0] = "\n\t"
	}
	for i, e := range err.Errs {
		s[i+1] = e.Error()
	}
	return strings.Join(s, "\n\t")
}

func (err Errors) Errors() []error {
	return err.Errs
}

////////////////////////////////////////////////////////////////

func IfError(err error, args ...interface{}) bool {
	if err != nil {
		if len(args) > 0 {
			err = errors.Wrap(err, fmt.Sprint(args...))
		}
		fmt.Fprintln(os.Stderr, err)
		return true
	}
	return false
}

func IfErrorf(err error, args ...interface{}) bool {
	if err != nil {
		if len(args) > 0 {
			if format, ok := args[0].(string); ok {
				err = errors.Wrapf(err, format, args[1:])
			}
		}
		fmt.Fprintln(os.Stderr, err)
		return true
	}
	return false
}

func IfFatal(err error, args ...interface{}) {
	if err != nil {
		IfError(err, args...)
		os.Exit(1)
	}
}

func IfFatalf(err error, format string, args ...interface{}) {
	if err != nil {
		IfErrorf(err, args...)
		os.Exit(1)
	}
}

func Log(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
}

func Logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}
