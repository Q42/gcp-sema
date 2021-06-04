package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/BTBurke/snapshot"
	"github.com/segmentio/textio"
	"github.com/stretchr/testify/assert"
)

func TestFixtures(t *testing.T) {
	build := exec.Command("go", "build", "-ldflags", "-w -s", "-o", "./gcp-sema", "..")
	build.Env = append(os.Environ(), "GO111MODULE=on", "CGO_ENABLED=0")
	build.Stderr = textio.NewPrefixWriter(os.Stderr, "stderr: ")
	build.Stdout = textio.NewPrefixWriter(os.Stderr, "stdout: ")
	err := build.Run()
	assert.NoError(t, err)

	files, _ := ioutil.ReadDir("./fixtures")
	for _, f := range files {
		cmd := exec.Command("sh", "run.sh")
		cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s:./", os.Getenv("PATH")))
		cmd.Dir = path.Join("./fixtures", f.Name())
		t.Run(cmd.Dir, func(t *testing.T) {
			out := bytes.NewBuffer(nil)
			cmd.Stderr = textio.NewPrefixWriter(out, "stderr: ")
			cmd.Stdout = textio.NewPrefixWriter(out, "stdout: ")
			snaps, err := snapshot.New(snapshot.SnapDirectory(cmd.Dir), snapshot.ContextLines(2))
			assert.NoError(t, err)
			err = cmd.Run()
			assert.NoError(t, err)
			snaps.Assert(t, out.Bytes())
		})
	}
}
