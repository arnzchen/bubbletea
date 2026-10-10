package tea

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

type readerCycleDone struct{ err error }

type readerCycleCommand struct{}

func (readerCycleCommand) Run() error          { return nil }
func (readerCycleCommand) SetStdin(io.Reader)  {}
func (readerCycleCommand) SetStdout(io.Writer) {}
func (readerCycleCommand) SetStderr(io.Writer) {}

type readerCycleModel struct {
	writer io.Writer
	cycles int
	keys   int
	limit  int
	err    error
}

func (m *readerCycleModel) Init() Cmd {
	return Exec(readerCycleCommand{}, func(err error) Msg { return readerCycleDone{err} })
}

func (m *readerCycleModel) View() string { return "" }

func (m *readerCycleModel) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case readerCycleDone:
		if msg.err != nil {
			m.err = msg.err
			return m, Quit
		}
		m.cycles++
		return m, func() Msg {
			if _, err := m.writer.Write([]byte("x")); err != nil {
				return readerCycleDone{err}
			}
			return nil
		}
	case KeyMsg:
		if msg.String() != "x" {
			m.err = io.ErrUnexpectedEOF
			return m, Quit
		}
		m.keys++
		if m.keys == m.limit {
			return m, Quit
		}
		return m, m.Init()
	}
	return m, nil
}

func TestExecCyclesCloseReplacedReaders(t *testing.T) {
	before := countReaderPollDescriptors(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close() //nolint:errcheck
	defer w.Close() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	model := &readerCycleModel{writer: w, limit: 64}
	p := NewProgram(model, WithInput(r), WithOutput(io.Discard),
		WithoutRenderer(), WithoutSignalHandler(), WithContext(ctx))
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	if model.err != nil || model.cycles != model.limit || model.keys != model.limit {
		t.Fatalf("cycles=%d keys=%d error=%v", model.cycles, model.keys, model.err)
	}
	after := countReaderPollDescriptors(t)
	t.Logf("%d Exec/input cycles: poll descriptors %d -> %d", model.cycles, before, after)
	if after > before {
		t.Fatalf("poll descriptors grew by %d", after-before)
	}
}

type closeTrackingReader struct{ closes int }

func (*closeTrackingReader) Read([]byte) (int, error) { return 0, io.EOF }
func (*closeTrackingReader) Cancel() bool             { return false }
func (r *closeTrackingReader) Close() error {
	r.closes++
	return nil
}

func TestInitCancelReaderClosesOnlyFinishedReader(t *testing.T) {
	for _, finished := range []bool{false, true} {
		t.Run(map[bool]string{false: "active", true: "finished"}[finished], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			old := &closeTrackingReader{}
			done := make(chan struct{})
			if finished {
				close(done)
			}
			p := &Program{ctx: ctx, cancelReader: old, readLoopDone: done,
				input: strings.NewReader(""), msgs: make(chan Msg), errs: make(chan error, 1)}
			if err := p.initCancelReader(false); err != nil {
				t.Fatal(err)
			}
			defer p.cancelReader.Close() //nolint:errcheck
			p.waitForReadLoop()
			want := 0
			if finished {
				want = 1
			}
			if old.closes != want {
				t.Fatalf("old reader closed %d times, want %d", old.closes, want)
			}
		})
	}
}
