package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type fieldError struct{ Field, Message string }

func (e *fieldError) Error() string { return e.Message }

// checkText never trims or rewrites v: bad input is refused, not cleaned (constitution II).
func checkText(field, v string, required bool, maxChars int, multiline bool) *fieldError {
	n := utf8.RuneCountInString(v)
	if required && (strings.TrimSpace(v) == "" || n > maxChars) {
		return &fieldError{field, fmt.Sprintf("%s must be 1 to %d characters", field, maxChars)}
	}
	if n > maxChars {
		return &fieldError{field, fmt.Sprintf("%s must be at most %d characters", field, maxChars)}
	}
	for _, r := range v {
		switch {
		case r == '\n' && !multiline:
			return &fieldError{field, field + " must be a single line"}
		case r != '\n' && unicode.IsControl(r):
			return &fieldError{field, field + " must not contain control characters"}
		}
	}
	return nil
}

var statuses = []string{"todo", "in_progress", "in_review", "done"}

func checkStatus(v string) *fieldError {
	if slices.Contains(statuses, v) {
		return nil
	}
	return &fieldError{"status", "status must be one of todo, in_progress, in_review, done"}
}

func validUser(id int) bool { return id >= 1 && id <= len(users) }

// 0 means unassigned: null is refused for every field, so "no one" needs a real value.
func checkAssignee(id int) *fieldError {
	if id == 0 || validUser(id) {
		return nil
	}
	return &fieldError{"assigneeId", "assigneeId must be 0 or a user ID from 1 to 5"}
}

// parseConfig errors name the flag but never repeat the value given.
func parseConfig(args []string) (port int, dataPath string, err error) {
	fs := flag.NewFlagSet("taskify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	portArg := fs.String("port", "8080", "")
	fs.StringVar(&dataPath, "data", "taskify.json", "")
	if fs.Parse(args) != nil || fs.NArg() > 0 {
		return 0, "", errors.New("argument is not allowed; use only -port and -data")
	}
	port, err = strconv.Atoi(*portArg)
	if err != nil || port < 1 || port > 65535 {
		return 0, "", errors.New("-port must be from 1 to 65535")
	}
	if dataPath == "" {
		return 0, "", errors.New("-data must not be empty")
	}
	if fi, err := os.Stat(dataPath); err == nil && fi.IsDir() {
		return 0, "", errors.New("-data must not be a directory")
	}
	if fi, err := os.Stat(filepath.Dir(dataPath)); err != nil || !fi.IsDir() {
		return 0, "", errors.New("-data must be in an existing directory")
	}
	return port, dataPath, nil
}
