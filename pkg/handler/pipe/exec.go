package pipe

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
)

// PipeExec reads from stdin passing the data into stdin of the command.
// stdout is written to the provided writers.
// The content type is set as directed
type PipeExec struct {
	Command     []string
	ContentType string
}

func (p *PipeExec) Transform(r *http.Response) error {
	var out bytes.Buffer

	err := p.run(&out, r.Body)
	if err != nil {
		return fmt.Errorf("error running command '%s': %v", strings.Join(p.Command, " "), err)
	}
	r.Body = ioutil.NopCloser(&out)
	r.Header.Set("Content-type", p.ContentType)
	return nil
}

func (p *PipeExec) run(stdout io.Writer, stdin io.Reader) error {
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
		io.Copy(stdinPipe, stdin)
		stdinPipe.Close()
	}()

	done := make(chan struct{})
	go func() {
		io.Copy(stdout, stdoutPipe)
		stdoutPipe.Close()
		close(done)
	}()

	err = cmd.Wait()
	<-done

	return err
}
