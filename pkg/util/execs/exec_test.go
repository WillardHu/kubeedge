/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package execs

import (
	"testing"
)

func TestCommand_GetStdOut(t *testing.T) {
	cases := []struct {
		name     string
		cmd      Command
		expected string
	}{
		{
			name: "case1",
			cmd: Command{
				StdOut: []byte("success\n"),
			},
			expected: "success",
		},
		{
			name: "case2",
			cmd: Command{
				StdOut: nil,
			},
			expected: "",
		},
	}
	for _, c := range cases {
		out := c.cmd.GetStdOut()
		if out != c.expected {
			t.Errorf("expected %v, got %v", c.expected, out)
		}
	}
}

func TestCommand_GetStdErr(t *testing.T) {
	cases := []struct {
		name     string
		cmd      Command
		expected string
	}{
		{
			name: "case1",
			cmd: Command{
				StdErr: []byte("failed\n"),
			},
			expected: "failed",
		},
		{
			name: "case2",
			cmd: Command{
				StdErr: nil,
			},
			expected: "",
		},
	}
	for _, c := range cases {
		out := c.cmd.GetStdErr()
		if out != c.expected {
			t.Errorf("%v: expected %v, got %v", c.name, c.expected, out)
		}
	}
}

func TestExecutorNoArgs(t *testing.T) {
	cmd := NewCommand("true")
	err := cmd.Exec()
	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
	if len(cmd.StdOut) != 0 || len(cmd.StdErr) != 0 {
		t.Errorf("expected no output, got stdout: %q, stderr: %q", string(cmd.StdOut), string(cmd.StdErr))
	}

	cmd = NewCommand("false")
	err = cmd.Exec()
	if err == nil {
		t.Errorf("expected failure, got nil error")
	}
	if len(cmd.StdOut) != 0 || len(cmd.StdErr) != 0 {
		t.Errorf("expected no output, got stdout: %q, stderr: %q", string(cmd.StdOut), string(cmd.StdErr))
	}

	if cmd.ExitCode != 1 {
		t.Errorf("expected exit status 1, got %d", cmd.ExitCode)
	}
}

func TestExecutorWithArgs(t *testing.T) {
	cmd := NewCommand("echo stdout")

	if err := cmd.Exec(); err != nil {
		t.Errorf("expected success, got %+v", err)
	}
	if string(cmd.StdOut) != "stdout\n" {
		t.Errorf("unexpected output: %q", string(cmd.StdOut))
	}

	cmd = NewCommand("echo stderr > /dev/stderr")
	if err := cmd.Exec(); err != nil {
		t.Errorf("expected success, got %+v", err)
	}
	if string(cmd.StdErr) != "stderr\n" {
		t.Errorf("unexpected output: %q", string(cmd.StdErr))
	}
}

func TestExecutableNotFound(t *testing.T) {
	cmd := NewCommand("fake_executable_name")
	err := cmd.Exec()
	if err == nil {
		t.Errorf("expected err, got nil")
	}
}
