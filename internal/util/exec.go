package util

import (
	"bytes"
	"os/exec"
)

type Command struct {
	cmd    *exec.Cmd
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func (cmd *Command) Run() error {
	return cmd.cmd.Run()
}

func (cmd *Command) Output() string {
	return cmd.stdout.String()
}

func (cmd *Command) Error() string {
	return cmd.stderr.String()
}

func NewCommand(name string, args ...string) *Command {
	cmd := Command{}
	cmd.cmd = exec.Command(name, args...)
	cmd.cmd.Stdout, cmd.cmd.Stderr = &cmd.stdout, &cmd.stderr
	return &cmd
}
