package cmd

import (
	"errors"
	"os"
	"os/exec"
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

func defaultBranch() (string, error) {
	data, err := gitOutput("symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", err
	}
	items := strings.Split(strings.TrimSpace(string(data)), "/")
	if len(items) < 1 {
		return "", errors.New("HEAD is empty")
	}
	return items[len(items)-1], nil
}

func checkUncommittedFiles() (bool, error) {
	data, err := gitOutput("status", "-s")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(data)) == "", nil
}

func firstUnmerged(defBranch string) (hash string, summary string, body string, err error) {
	var data []byte
	data, err = gitOutput("log", "HEAD", "--not", "origin/"+defBranch, "--format=%H", "--reverse")
	if err != nil {
		return
	}
	commits := strings.Fields(string(data))
	if len(commits) == 0 {
		err = errors.New("no commits to be pushed")
		return
	}

	hash = commits[0]

	data, err = gitOutput("show", "--format=%s%n%b", "-s", hash)
	if err != nil {
		return
	}

	fields := strings.SplitN(string(data), "\n", 2)
	return hash, fields[0], strings.TrimSpace(fields[1]), nil
}

func originURL() (string, error) {
	data, err := gitOutput("remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
