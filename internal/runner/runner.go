// Package runner запускает внешний процесс, читает его stdout/stderr
// построчно и уведомляет вызывающий код через callback.
package runner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"crunch/internal/logtheme"
)

// scannerBuf — стартовый размер буфера для bufio.Scanner.
// 256 КБ покрывает самые длинные строки dnf с запасом.
const scannerBuf = 256 * 1024

type Callbacks struct {
	OnLine func(line string, tag logtheme.Tag)
	OnStep func(index int, marker string)
	OnDone func(rc int)
}

type Runner struct {
	cmd   *exec.Cmd
	steps []string
	cb    Callbacks

	mu        sync.Mutex
	cancelled bool
	proc      *os.Process
	pipeW     *io.PipeWriter

	doneCh    chan struct{}
	closeOnce sync.Once
}

func New(cmd *exec.Cmd, steps []string, cb Callbacks) *Runner {
	return &Runner{
		cmd:    cmd,
		steps:  steps,
		cb:     cb,
		doneCh: make(chan struct{}),
	}
}

func (r *Runner) Run() {
	defer r.closeOnce.Do(func() { close(r.doneCh) })

	stdout, err := r.startProcess()
	if err != nil {
		r.cb.OnLine(fmt.Sprintf("Ошибка запуска: %v", err), logtheme.Error)
		r.cb.OnDone(-1)
		return
	}

	r.mu.Lock()
	shouldKill := r.cancelled
	r.mu.Unlock()
	if shouldKill {
		r.signal(syscall.SIGTERM)
	}

	waitDone := r.waitAsync()
	r.readOutput(stdout)
	<-waitDone

	r.cb.OnDone(r.finalExitCode())
}

func (r *Runner) startProcess() (io.Reader, error) {
	stdin, err := r.cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin: %w", err)
	}

	pr, pw := io.Pipe()
	r.cmd.Stdout = pw
	r.cmd.Stderr = pw

	if err := r.cmd.Start(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.proc = r.cmd.Process
	r.pipeW = pw
	r.mu.Unlock()

	// Шлём "y" на каждый интерактивный запрос dnf copr enable.
	// 15 попыток — с запасом; обычно dnf спрашивает около 2 раз.
	go func() {
		defer stdin.Close()
		for i := 0; i < 15; i++ {
			if _, err := stdin.Write([]byte("y\n")); err != nil {
				return
			}
		}
	}()

	return pr, nil
}

func (r *Runner) waitAsync() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		_ = r.cmd.Wait()
		r.mu.Lock()
		pw := r.pipeW
		r.mu.Unlock()
		if pw != nil {
			_ = pw.Close()
		}
		close(done)
	}()
	return done
}

func (r *Runner) readOutput(stdout io.Reader) {
	stepIdx := 0
	total := len(r.steps)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, scannerBuf), scannerBuf)

	for scanner.Scan() {
		line := logtheme.StripANSI(scanner.Text())
		if strings.TrimSpace(line) == "" {
			r.cb.OnLine("", logtheme.Default)
			continue
		}
		r.cb.OnLine(line, logtheme.ParseTag(line))

		if stepIdx < total && strings.Contains(line, r.steps[stepIdx]) {
			stepIdx++
			r.cb.OnStep(stepIdx, r.steps[stepIdx-1])
		}
	}

	// Если поток оборвался до EOF — сообщаем об этом в GUI.
	if err := scanner.Err(); err != nil {
		r.cb.OnLine(fmt.Sprintf("Поток вывода прервался: %v", err), logtheme.Error)
	}
}

func (r *Runner) finalExitCode() int {
	rc := 0
	if r.cmd.ProcessState != nil {
		rc = r.cmd.ProcessState.ExitCode()
	}
	r.mu.Lock()
	if r.cancelled {
		rc = -1
	}
	r.mu.Unlock()
	return rc
}

func (r *Runner) Cancel() {
	r.mu.Lock()
	if r.cancelled {
		r.mu.Unlock()
		return
	}
	r.cancelled = true
	r.mu.Unlock()
	r.signal(syscall.SIGTERM)
}

func (r *Runner) ForceKill() {
	r.signal(syscall.SIGKILL)
}

func (r *Runner) Wait(timeout time.Duration) bool {
	select {
	case <-r.doneCh:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (r *Runner) signal(sig syscall.Signal) {
	r.mu.Lock()
	proc := r.proc
	r.mu.Unlock()
	if proc == nil {
		return
	}
	if pgid, err := syscall.Getpgid(proc.Pid); err == nil && pgid > 0 {
		_ = syscall.Kill(-pgid, sig)
		return
	}
	_ = proc.Signal(sig)
}