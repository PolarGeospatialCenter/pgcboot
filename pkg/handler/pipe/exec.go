package pipe

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"

	"github.com/honeycombio/beeline-go/trace"
)

// PipeExec reads from stdin passing the data into stdin of the command.
// stdout is written to the provided writers.
// The content type is set as directed
type PipeExec struct {
	Command     []string
	ContentType string
}

func (p *PipeExec) Transform(ctx context.Context, r *http.Response) error {
	var out bytes.Buffer

	err := p.run(ctx, &out, r.Body)
	if err != nil {
		return fmt.Errorf("error running command '%s': %v", strings.Join(p.Command, " "), err)
	}
	r.Body = ioutil.NopCloser(&out)
	r.Header.Set("Content-type", p.ContentType)
	return nil
}

func (p *PipeExec) run(ctx context.Context, stdout io.Writer, stdin io.Reader) error {
	span := trace.GetSpanFromContext(ctx)
	if span != nil {
		span.AddField("command", p.Command)
	}

	cmd := exec.Command(p.Command[0], p.Command[1:]...)
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		copiedFromStdIn, err := io.Copy(stdinPipe, stdin)
		if span != nil {
			span.AddField("stdin.bytes_copied", copiedFromStdIn)
			span.AddField("stdin.err", err)
		}
		stdinPipe.Close()
	}()

	done := make(chan struct{})
	go func() {
		copiedFromStdOut, err := io.Copy(stdout, stdoutPipe)
		if span != nil {
			span.AddField("stdout.bytes_copied", copiedFromStdOut)
			span.AddField("stdout.err", err)
		}
		stdoutPipe.Close()
		close(done)
	}()

	err = cmd.Wait()
	<-done

	return err
}
