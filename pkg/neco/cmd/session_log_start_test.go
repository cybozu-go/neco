package cmd

import (
	"reflect"
	"testing"
)

func TestSessionLogStart(t *testing.T) {
	cases := []struct {
		Name                   string
		InputOriginalCommand   string
		InputTmpdir            string
		ExpectedCmdline        []string
		ExpectedTarMemberFiles []string
	}{
		{
			Name:                   "normal login",
			InputOriginalCommand:   "",
			InputTmpdir:            "/tmp/a",
			ExpectedCmdline:        []string{"/usr/bin/script", "-q", "-t/tmp/a/timingfile", "-e", "/tmp/a/typescript"},
			ExpectedTarMemberFiles: []string{"typescript", "timingfile"},
		},
		{
			Name:                   "normal remote command",
			InputOriginalCommand:   "ls",
			InputTmpdir:            "/tmp/a",
			ExpectedCmdline:        []string{"/usr/bin/script", "-q", "-t/tmp/a/timingfile", "-e", "/tmp/a/typescript", "-c", "ls"},
			ExpectedTarMemberFiles: []string{"SSH_ORIGINAL_COMMAND", "typescript", "timingfile"},
		},
		{
			Name:                   "normal internal-sftp (1)",
			InputOriginalCommand:   "internal-sftp",
			InputTmpdir:            "/tmp/a",
			ExpectedCmdline:        []string{"/bin/sh", "-c", "/usr/lib/openssh/sftp-server"},
			ExpectedTarMemberFiles: []string{"SSH_ORIGINAL_COMMAND"},
		},
		{
			Name:                   "normal internal-sftp (2)",
			InputOriginalCommand:   "/usr/lib/openssh/sftp-server",
			InputTmpdir:            "/tmp/a",
			ExpectedCmdline:        []string{"/bin/sh", "-c", "/usr/lib/openssh/sftp-server"},
			ExpectedTarMemberFiles: []string{"SSH_ORIGINAL_COMMAND"},
		},
		{
			Name:                   "normal scp",
			InputOriginalCommand:   "scp -t .",
			InputTmpdir:            "/tmp/a",
			ExpectedCmdline:        []string{"/bin/sh", "-c", "scp -t ."},
			ExpectedTarMemberFiles: []string{"SSH_ORIGINAL_COMMAND"},
		},
		{
			Name:                   "mkdirtemp failed login",
			InputOriginalCommand:   "",
			InputTmpdir:            "",
			ExpectedCmdline:        []string{"/usr/bin/bash"},
			ExpectedTarMemberFiles: []string{},
		},
		{
			Name:                   "mkdirtemp failed remote command",
			InputOriginalCommand:   "ls",
			InputTmpdir:            "",
			ExpectedCmdline:        []string{"/usr/bin/bash", "-c", "ls"},
			ExpectedTarMemberFiles: []string{},
		},
		{
			Name:                   "mkdirtemp failed internal-sftp (1)",
			InputOriginalCommand:   "internal-sftp",
			InputTmpdir:            "",
			ExpectedCmdline:        []string{"/bin/sh", "-c", "/usr/lib/openssh/sftp-server"},
			ExpectedTarMemberFiles: []string{},
		},
		{
			Name:                   "mkdirtemp failed internal-sftp (2)",
			InputOriginalCommand:   "/usr/lib/openssh/sftp-server",
			InputTmpdir:            "",
			ExpectedCmdline:        []string{"/bin/sh", "-c", "/usr/lib/openssh/sftp-server"},
			ExpectedTarMemberFiles: []string{},
		},
		{
			Name:                   "mkdirtemp failed scp",
			InputOriginalCommand:   "scp -t .",
			InputTmpdir:            "",
			ExpectedCmdline:        []string{"/bin/sh", "-c", "scp -t ."},
			ExpectedTarMemberFiles: []string{},
		},
	}

	for _, c := range cases {
		cmdline, tarMemberFiles := sessionLogStartGetParam(c.InputOriginalCommand, c.InputTmpdir)
		if !reflect.DeepEqual(cmdline, c.ExpectedCmdline) || !reflect.DeepEqual(tarMemberFiles, c.ExpectedTarMemberFiles) {
			t.Errorf("case \"%s\": actual cmdline = %#v, actual tarMemberFiles = %#v", c.Name, cmdline, tarMemberFiles)
		}
	}
}
