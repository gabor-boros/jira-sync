package main

import (
	"gabor-boros/jira-sync/cmd"
)

var version string
var commit string

func main() {
	cmd.Execute(version, commit)
}
