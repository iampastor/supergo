package supervisord

import (
	"errors"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type Program struct {
	Name string
	cfg  *ProgramConfig

	process  *Process
	files    []*os.File
	maxRetry int
	logger   *log.Logger

	status *ProgramStatus
}

// 完善信息
type ProgramStatus struct {
	Name      string `json:"name,omitempty"`
	Pid       int    `json:"pid,omitempty"`
	StartTime int64  `json:"start_time,omitempty"`
	StopTime  int64  `json:"stop_time,omitempty"`
	State     string `json:"state,omitempty"`
}

type ProgramState string

const (
	ProcessStateStarting = "Starting"
	ProcessStateRunning  = "Running"
	// ProcessStateStopping = "Stopping"
	ProcessStateStopped = "Stopped"
	ProcessStateExited  = "Exited"
)

func NewProgram(name string, cfg *ProgramConfig) (p *Program, err error) {
	p = &Program{
		cfg:    cfg,
		Name:   name,
		logger: log.New(os.Stderr, "["+name+"] ", log.LstdFlags|log.Lshortfile),
		status: &ProgramStatus{
			Name:      name,
			Pid:       0,
			StartTime: 0,
			StopTime:  0,
			State:     ProcessStateStopped,
		},
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

func (program *Program) Status() *ProgramStatus {
	return program.status
}

type Process struct {
	cmd      *exec.Cmd
	stopChan chan struct{}
	spawn    bool
}

func (program *Program) StartProcess() {
	if program.status.State != ProcessStateStopped && program.status.State != ProcessStateExited {
		return
	}
	program.logger.Printf("start")
	program.status.State = ProcessStateStarting
	go program.startNewProcess()
	return
}

func (program *Program) startNewProcess() {
	var stderr, stdout *os.File
	var err error
	if program.cfg.StderrFile != "" {
		stderr, err = os.OpenFile(program.cfg.StderrFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			program.logger.Printf("open file %s: %s", program.cfg.StderrFile, err.Error())
		}
	}
	if program.cfg.StderrFile != "" {
		stdout, err = os.OpenFile(program.cfg.StdoutFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			program.logger.Printf("open file %s: %s", program.cfg.StderrFile, err.Error())
		}
	}
	progCmds := strings.Split(strings.TrimSpace(program.cfg.Command), " ")
	cmd := &exec.Cmd{
		Dir:    program.cfg.Directory,
		Path:   progCmds[0],
		Args:   append(progCmds, program.cfg.Args...),
		Stderr: stderr,
		Stdout: stdout,
		SysProcAttr: &syscall.SysProcAttr{
			Setpgid: true,
		},
	}
	cmd.ExtraFiles = program.files

	process := &Process{
		cmd:      cmd,
		stopChan: make(chan struct{}, 1),
	}
	program.process = process
	err = process.run()
	if stderr != nil {
		stderr.Close()
	}
	if stdout != nil {
		stdout.Close()
	}
	if err == nil {
		program.status.StartTime = time.Now().Unix()
		program.status.Pid = process.cmd.Process.Pid

		program.maxRetry = 0
		program.status.State = ProcessStateRunning
		// TODO: exit code expected
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
	if program.status.State == ProcessStateStopped {
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
			program.status.State = ProcessStateExited
		}
	} else {
		program.logger.Printf("exited")
		program.status.State = ProcessStateExited
	}
}

func (program *Program) RestartProess() (process *Process) {
	if program.status.State != ProcessStateRunning {
		return
	}
	program.logger.Printf("restart")
	oldProc := program.process
	program.status.State = ProcessStateStarting
	go program.startNewProcess()
	if oldProc != nil {
		oldProc.spawn = true
		program.stopProc(oldProc)
	}

	return
}

func (program *Program) StopProcess() (exitCode int) {
	if program.status.State != ProcessStateRunning {
		return
	}
	program.logger.Printf("stop")
	proc := program.process
	program.status.State = ProcessStateStopped
	program.stopProc(proc)
	program.status.StopTime = time.Now().Unix()
	program.status.Pid = 0
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
