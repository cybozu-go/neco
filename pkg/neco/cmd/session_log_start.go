package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	osuser "os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var sessionLogStartCmd = &cobra.Command{
	Use:   "start",
	Short: "start shell and session log recording",
	Long: `Start shell and session log recording.

recorded session log is put into object bucket created by Ceph RGW.`,

	Args: cobra.ExactArgs(0),
	Run:  sessionLogStartRun,
}

func init() {
	sessionLogCmd.AddCommand(sessionLogStartCmd)
}

const s3gwEndpoint = "http://s3gw.session-log.svc"
const sftpServer = "/usr/lib/openssh/sftp-server"

// sessionLogStartGetParam calculates child process invocation param
// note: tmpdir == "" implies Mkdirtmp failure
func sessionLogStartGetParam(originalCommand, tmpdir string) ( /* cmdline */ []string /* tarMemberFiles */, []string) {
	var cmdline []string
	tarMemberFiles := []string{}
	runViaScript := false

	if tmpdir == "" {
		// even if session log recording fails, we have to invoke shell
		cmdline = []string{"/usr/bin/bash"} // XXX
	} else {
		typescriptPath := filepath.Join(tmpdir, "typescript")
		timingfilePath := filepath.Join(tmpdir, "timingfile")
		cmdline = []string{"/usr/bin/script", "-q", "-t" + timingfilePath, "-e", typescriptPath}
		runViaScript = true
	}
	if originalCommand != "" {
		if tmpdir != "" {
			os.WriteFile(filepath.Join(tmpdir, "SSH_ORIGINAL_COMMAND"), []byte(originalCommand), 0600)
			tarMemberFiles = append(tarMemberFiles, "SSH_ORIGINAL_COMMAND")
		}
		// those should not run via script
		if originalCommand == "internal-sftp" {
			originalCommand = sftpServer
			cmdline = []string{"/bin/sh"}
			runViaScript = false
		} else if originalCommand == sftpServer || strings.HasPrefix(originalCommand, "scp -t") {
			cmdline = []string{"/bin/sh"}
			runViaScript = false
		}
		cmdline = append(cmdline, "-c", originalCommand)
	}
	if runViaScript {
		tarMemberFiles = append(tarMemberFiles, "typescript", "timingfile")
	}

	return cmdline, tarMemberFiles
}

func sessionLogStartRun(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	startTime := time.Now()
	startTimeStr := startTime.Format("20060102T150405")

	originalCommand := os.Getenv("SSH_ORIGINAL_COMMAND")

	tmpdir, err := os.MkdirTemp("/tmp", "session-log-"+startTimeStr+"-")
	if err != nil {
		tmpdir = ""
		fmt.Fprintf(os.Stderr, "failed to create tmp dir: %s\ncontinuing without session recording...\n", err.Error())
	}
	cmdline, tarMemberFiles := sessionLogStartGetParam(originalCommand, tmpdir)

	shellCmd := exec.CommandContext(ctx, cmdline[0], cmdline[1:]...)
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr
	runErr := shellCmd.Run()

	if tmpdir != "" {
		func() {
			os.Chdir(tmpdir)
			duration := time.Since(startTime)
			durationStr := duration.Truncate(time.Second).String()
			user, err := osuser.Current()
			if err != nil {
				user.Username = "unknown"
			}
			filename := fmt.Sprintf("session-log-%s-%d-%s-%s.tar.gz", startTimeStr, os.Getpid(), durationStr, user.Username)
			cmdline := []string{"/bin/tar", "zcvf", filename}
			cmdline = append(cmdline, tarMemberFiles...)
			err = exec.CommandContext(ctx, cmdline[0], cmdline[1:]...).Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to archive session log: %s\n", err.Error())
				return
			}

			client := &http.Client{}
			reader, err := os.Open(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open session log: %s\n", err.Error())
				return
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, s3gwEndpoint+"/bucket/"+filename, reader)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create PUT request: %s\n", err.Error())
				return
			}
			res, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to put session log: %s\n", err.Error())
				return
			}
			res.Body.Close()
			os.RemoveAll(tmpdir) // Not defer. In case of put failure, we should retain session log files.
			fmt.Fprintf(os.Stderr, "succeeded in putting session log %s.\n", filename)
		}()
	}

	if runErr == nil {
		os.Exit(0)
	} else if exitErr, ok := runErr.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	} else {
		os.Exit(255)
	}
}
