package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"testing"

	"github.com/BTBurke/snapshot"
	"github.com/segmentio/textio"
	"github.com/stretchr/testify/assert"
)

func TestFixtures(t *testing.T) {
	files, _ := ioutil.ReadDir("./fixtures")
	for _, f := range files {
		cmd := exec.Command("sh", "run.sh")
		cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s:../../../dist/gcp-sema_%s_%s", os.Getenv("PATH"), runtime.GOOS, runtime.GOARCH))
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
