package exif

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/rs/zerolog"
)

var (
	instance *PersistentExiftool
	once     sync.Once
	initErr  error
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
	mu      sync.Mutex
	timeout time.Duration
}

func GetExiftool(ctx context.Context) (*PersistentExiftool, error) {
	once.Do(func() {
		cfg := config.Global().Sync.Exiftool
		instance, initErr = newPersistentExiftool(
			ctx,
			cfg.ResolvedPath,
			cfg.Timeout,
		)
	})

	return instance, initErr
}

func newPersistentExiftool(ctx context.Context, exiftoolPath string, timeout time.Duration) (*PersistentExiftool, error) {
	cmd := exec.CommandContext(
		ctx,
		exiftoolPath,
		"-stay_open", "True",
		"-@", "-",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &PersistentExiftool{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdoutPipe),
		timeout: timeout,
	}, nil
}
func ShutdownExiftool() error {
	if instance == nil {
		return nil
	}
	return instance.Close()
}

func (p *PersistentExiftool) Read(ctx context.Context, imagePath string) (RawMetadata, error) {

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if err := p.sendCommand(imagePath); err != nil {
		return nil, err
	}

	data, err := p.readResponse(ctx)
	if err != nil {
		return nil, err
	}

	raw, err := parseExiftoolJSON(data)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
func (p *PersistentExiftool) sendCommand(imagePath string) error {
	_, err := fmt.Fprintf(p.stdin,
		"-j\n-G1\n-struct\n-a\n%s\n-execute\n",
		imagePath,
	)
	return err
}
func (p *PersistentExiftool) readResponse(ctx context.Context) ([]byte, error) {
	var buf bytes.Buffer

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line, err := p.stdout.ReadString('\n')
		if err != nil {
			return nil, err
		}

		if strings.TrimSpace(line) == "{ready}" {
			break
		}

		buf.WriteString(line)
	}

	return buf.Bytes(), nil
}
func (p *PersistentExiftool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stdin != nil {
		fmt.Fprint(p.stdin, "-stay_open\nFalse\n")
		_ = p.stdin.Close()
	}

	return p.cmd.Wait()
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
