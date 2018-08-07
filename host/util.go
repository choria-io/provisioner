package host

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"
)

func runDecodedHelper(ctx context.Context, helper string, args []string, input string, output interface{}) error {
	o, err := runHelper(ctx, helper, args, input)
	if err != nil {
		return err
	}

	err = json.Unmarshal(o, output)
	if err != nil {
		return fmt.Errorf("cannot decode output from %s: %s", helper, err)
	}

	return nil
}

func runHelper(ctx context.Context, helper string, args []string, input string) ([]byte, error) {
	tctx, cancel := context.WithTimeout(ctx, time.Duration(5*time.Second))
	defer cancel()

	execution := exec.CommandContext(tctx, helper, args...)

	stdin, err := execution.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot create stdin for %s: %s", helper, err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot open STDOUT for %s: %s", helper, err)
	}
	defer stdout.Close()

	err = execution.Start()
	if err != nil {
		return nil, fmt.Errorf("cannot start %s: %s", helper, err)
	}

	buf := new(bytes.Buffer)

	n, err := buf.ReadFrom(stdout)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s output: %s", helper, err)
	}
	if n == 0 {
		return nil, fmt.Errorf("cannot read %s output: zero bytes received", helper)
	}

	go func() {
		execution.Wait()
	}()

	return buf.Bytes(), nil
}
