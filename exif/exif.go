package exif

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ignisVeneficus/logging"
	"github.com/rs/zerolog"
)

type RawMetadata map[string]RawMetaValue

type RawMetaValue struct {
	Value  any
	Source string
}

func (md *RawMetadata) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if md == nil || *md == nil {
		return
	}
	if level <= zerolog.DebugLevel {
		arr := zerolog.Arr()
		for key, mv := range *md {
			arr.Dict(zerolog.Dict().
				Str("key", key).
				Interface("value", mv.Value).
				Str("source", mv.Source))
		}
		e.Array("", arr)
	}
}

type PersistentExiftool struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	timeout time.Duration
	seq     uint64
}

func NewPersistentExiftool(c context.Context, exiftoolPath string, timeout time.Duration) (*PersistentExiftool, error) {
	logScope, ctx := logging.Enter(c, "exiftool/create", exiftoolPath, map[string]any{
		"path": exiftoolPath,
	})
	cmd := exec.CommandContext(
		ctx,
		exiftoolPath,
		"-stay_open", "True",
		"-@", "-",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}

	logging.Exit(logScope, "ok", nil)
	return &PersistentExiftool{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdoutPipe),
		timeout: timeout,
	}, nil
}

func (p *PersistentExiftool) Close() error {

	if p.stdin != nil {
		fmt.Fprint(p.stdin, "-stay_open\nFalse\n")
		_ = p.stdin.Close()
	}

	if p.cmd == nil {
		return nil
	}

	return p.cmd.Wait()
}

func (p *PersistentExiftool) kill() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	_ = p.stdin.Close()

	err := p.cmd.Process.Kill()

	// reap
	_ = p.cmd.Wait()

	return err
}

func (p *PersistentExiftool) Read(ctx context.Context, imagePath string) (RawMetadata, error) {
	if strings.Contains(imagePath, "\n") {
		return nil, errors.New("invalid path")
	}

	var err error
	p.seq++
	id := p.seq

	ready := fmt.Sprintf("{ready%d}", id)

	_, err = fmt.Fprintf(p.stdin,
		"-j\n-G1\n-struct\n-a\n%s\n-execute%d\n",
		imagePath, id,
	)
	if err != nil {
		return nil, err
	}

	var data bytes.Buffer
	var line string
	for {
		line, err = p.stdout.ReadString('\n')
		if err != nil {
			break
		}
		if strings.TrimSpace(line) == ready {
			break
		}
		data.WriteString(line)
	}

	if err != nil {
		return nil, err
	}

	raw, err := parseExiftoolJSON(data.Bytes())
	if err != nil {
		return nil, err
	}
	return raw, nil
}
func (p *PersistentExiftool) IsAlive() bool {
	return p.cmd != nil &&
		p.cmd.Process != nil &&
		(p.cmd.ProcessState == nil ||
			!p.cmd.ProcessState.Exited())
}

func parseExiftoolJSON(rawData []byte) (RawMetadata, error) {
	var parsed []map[string]any

	if err := json.Unmarshal(rawData, &parsed); err != nil {
		return nil, err
	}

	if len(parsed) == 0 {
		return nil, fmt.Errorf("empty exiftool result")
	}

	out := make(RawMetadata)

	for key, val := range parsed[0] {
		ns := strings.ToLower(strings.SplitN(key, ":", 2)[0])
		out[strings.ToLower(key)] = RawMetaValue{
			Value:  val,
			Source: ns,
		}
	}

	return out, nil
}
