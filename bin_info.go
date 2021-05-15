package main

import (
	"strconv"
	"strings"
)

// Vars only for git sha and diff handling
var GitSHA1 string = ""
var GitDirty string = "0"

func toolGitSHA1() string {
	return GitSHA1
}

func toolGitDirty() (dirty bool) {
	dirty = false
	dirtyLines, err := strconv.Atoi(strings.TrimSpace(GitDirty))
	if err == nil {
		dirty = (dirtyLines != 0)
	}
	return
}
