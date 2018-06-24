package supervisord

import (
	"errors"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Program struct {
	Name string
	cfg  *ProgramConfig

	process  *Process
	files    []*os.File
	maxRetry int
	state    ProgramState

	logger *log.Logger
}

type ProgramStatus struct {
	Name      string
	StartTime int64
	StopTime  int64
	State     string
}

type ProgramState string

const (
	ProcessStateStarting = "Starting"
	ProcessStateRunning  = "Running"
	// ProcessStateStopping = "Stopping"
	ProcessStateStopped = "Stopped"
	ProcessStateExited  = "Exited"
)

type ProgramConfig struct {
	Directory   string   `toml:"directory"`
	Command     string   `toml:"command"`
	Args        []string `toml:"args"`
	AutoRestart bool     `toml:"auto_restart"`
	StdoutFile  string   `toml:"stdout_file"`
	StderrFile  string   `toml:"stderr_file"`
	MaxRetry    int      `toml:"max_retry"`
	ListenAddrs []string `toml:"listen_addrs"`
	StopTimeout int      `toml:"stop_timeout"`
}

func NewProgram(name string, cfg *ProgramConfig) (p *Program, err error) {
	p = &Program{
		cfg:    cfg,
		Name:   name,
		state:  ProcessStateStopped,
		logger: log.New(os.Stderr, "["+name+"] ", log.LstdFlags|log.Lshortfile),
	}

	var files []*os.File
	for _, addr := range cfg.ListenAddrs {
		var l net.Listener
		var f *os.File
		l, err = net.Listen("tcp", addr)
		if err != nil {
			return
		}
		f, err = l.(*net.TCPListener).File()
		if err != nil {
			l.Close()
			return
		}
		l.Close()
		files = append(files, f)
	}
	p.files = files
	return
}

func (program *Program) Destory() {
	for _, f := range program.files {
		f.Close()
	}
	return
}

type Process struct {
	cmd      *exec.Cmd
	stopChan chan struct{}
	spawn    bool
}

func (program *Program) StartProcess() {
	if program.state != ProcessStateStopped && program.state != ProcessStateExited {
		return
	}
	program.logger.Printf("start")
	program.state = ProcessStateStarting
	go program.startNewProcess()
	return
}

func (program *Program) startNewProcess() {
	cmd := &exec.Cmd{
		Dir:    program.cfg.Directory,
		Path:   program.cfg.Command,
		Args:   append([]string{program.cfg.Command}, program.cfg.Args...),
		Stderr: os.Stderr,
		Stdout: os.Stdout,
	}
	cmd.ExtraFiles = program.files

	process := &Process{
		cmd:      cmd,
		stopChan: make(chan struct{}, 1),
	}
	program.process = process
	// TODO: exit code expected
	err := process.run()
	if err == nil {
		program.maxRetry = 0
		program.state = ProcessStateRunning
		exitCode, err := process.wait()
		// 进程执行完毕，可能是程序自动退出，也可能是通过stop退出
		close(process.stopChan)
		if err != nil {
			program.logger.Printf("wait error: %s", err.Error())
		}
		program.logger.Printf("exit with code %d", exitCode)
		// 如果是restart，该进程则不需要自动重启
		if process.spawn {
			return
		}
	} else {
		program.logger.Printf("start error: %s", err.Error())
	}
	program.shouldRetry()
}

func (program *Program) shouldRetry() {
	// 如果是被手动停止的，则不需要重启
	if program.state == ProcessStateStopped {
		return
	}
	if program.cfg.AutoRestart {
		program.maxRetry++
		if program.maxRetry <= program.cfg.MaxRetry {
			time.Sleep(time.Second * 1)
			program.logger.Printf("retry %d", program.maxRetry)
			program.startNewProcess()
		} else {
			program.logger.Printf("max retry excessed")
			program.state = ProcessStateExited
		}
	} else {
		program.logger.Printf("exited")
		program.state = ProcessStateExited
	}
}

func (program *Program) RestartProess() (process *Process) {
	if program.state != ProcessStateRunning {
		return
	}
	program.logger.Printf("restart")
	oldProc := program.process
	program.state = ProcessStateStarting
	go program.startNewProcess()
	if oldProc != nil {
		oldProc.spawn = true
		program.stopProc(oldProc)
	}

	return
}

func (program *Program) StopProcess() (exitCode int) {
	if program.state != ProcessStateRunning {
		return
	}
	program.logger.Printf("stop")
	proc := program.process
	program.state = ProcessStateStopped
	program.stopProc(proc)
	program.process = nil
	return
}

func (program *Program) stopProc(proc *Process) error {
	if err := proc.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		program.logger.Printf("stop process %s", err.Error())
	}
	select {
	case <-proc.stopChan:
	case <-time.After(time.Second * time.Duration(program.cfg.StopTimeout)):
		if err := proc.cmd.Process.Signal(syscall.SIGKILL); err != nil {
			program.logger.Printf("kill process %s", err.Error())
		}
	}

	return nil
}

func (program *Program) GetState() ProgramState {
	return program.state
}

func (process *Process) run() error {
	return process.cmd.Start()
}

func (process *Process) wait() (exitCode int, err error) {
	err = process.cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode, _ = getExitCode(exitErr.Sys())
			return
		} else {
			return 0, err
		}
	} else {
		return getExitCode(process.cmd.ProcessState.Sys())
	}
}

func getExitCode(v interface{}) (code int, err error) {
	if status, ok := v.(syscall.WaitStatus); ok {
		return status.ExitStatus(), nil
	} else {
		return 0, errors.New("can not get exit code")
	}
}
