package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"strings"
)

func sanityCheck() error {
	return exec.Command("git", "status").Run()
}

func git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitOutput(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func currentBranch() (string, error) {
	data, err := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	br := strings.TrimSpace(string(data))
	if br == "HEAD" {
		return "", errors.New("not in a local branch")
	}
	return br, nil
}

func checkUncommittedFiles() (bool, error) {
	data, err := gitOutput("status", "-s")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(data)) == "", nil
}

func firstUnmerged() (hash string, summary string, body string, err error) {
	var data []byte
	data, err = gitOutput("show", "HEAD", "--not", "origin/master", "--format=%H%n%s%n%b", "--no-patch")
	if err != nil {
		return
	}

	fields := strings.Split(string(data), "\n")
	if len(fields) < 3 {
		err = errors.New("no commits to be pushed")
		return
	}

	return fields[0], fields[1], fields[2], nil
}

func repoName() (string, error) {
	data, err := gitOutput("remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	name := path.Base(strings.TrimSpace(string(data)))
	return strings.Split(name, ".")[0], nil
}
