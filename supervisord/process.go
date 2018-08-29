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

	process        *Process
	files          []*os.File
	maxRetry       int
	logger         *log.Logger
	startChan      chan bool // 进程是否成功启动
	listenerInited bool      // listener是否已经初始化

	status *ProgramStatus
}

// 完善信息
type ProgramStatus struct {
	Name      string   `json:"name,omitempty"`
	Pid       int      `json:"pid,omitempty"`
	StartTime int64    `json:"start_time,omitempty"`
	StopTime  int64    `json:"stop_time,omitempty"`
	State     string   `json:"state,omitempty"`
	Listeners []string `json:"listeners,omitempty"`
}

type ProgramState string

const (
	ProcessStateStarting = "Starting"
	ProcessStateRunning  = "Running"
	// ProcessStateStopping = "Stopping"
	ProcessStateStopped = "Stopped"
	ProcessStateExited  = "Exited"
	ProcessStateFatal   = "Fatal"
	ProcessStateUnknown = "Unknown"
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
			Listeners: cfg.ListenAddrs,
		},
		startChan: make(chan bool, 1),
	}

	err = p.initListener()
	return
}

func (program *Program) initListener() error {
	if !program.listenerInited {
		var files []*os.File
		for _, addr := range program.cfg.ListenAddrs {
			var l net.Listener
			var f *os.File
			l, err := net.Listen("tcp", addr)
			if err != nil {
				program.status.State = ProcessStateFatal
				return err
			}
			f, err = l.(*net.TCPListener).File()
			if err != nil {
				program.status.State = ProcessStateFatal
				l.Close()
				return err
			}
			l.Close()
			files = append(files, f)
		}
		program.files = files
		program.listenerInited = true
	}
	return nil
}

func (program *Program) closeListener() {
	for _, l := range program.files {
		l.Close()
	}
	program.listenerInited = false
}

func (program *Program) Destory() {
	program.closeListener()
}

func (program *Program) Status() *ProgramStatus {
	return program.status
}

type Process struct {
	cmd      *exec.Cmd
	stopChan chan struct{}
	spawn    bool // 标识是否是手动restart
}

func (program *Program) StartProcess() {
	if program.status.State != ProcessStateStopped && program.status.State != ProcessStateExited && program.status.State != ProcessStateFatal {
		return
	}
	program.logger.Printf("start")
	program.status.State = ProcessStateStarting
	program.initListener()
	go program.startNewProcess()
	<-program.startChan
	return
}

func (program *Program) startNewProcess() {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			program.status.State = ProcessStateUnknown
		}
	}()
	var stderr, stdout *os.File
	var err error
	if program.cfg.StderrLogFile != "" {
		stderr, err = os.OpenFile(program.cfg.StderrLogFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			program.logger.Printf("open file %s: %s", program.cfg.StderrLogFile, err.Error())
		}
	}
	if program.cfg.StdoutLogFile != "" {
		stdout, err = os.OpenFile(program.cfg.StdoutLogFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			program.logger.Printf("open file %s: %s", program.cfg.StdoutLogFile, err.Error())
		}
	}
	progCmds := strings.Split(strings.TrimSpace(program.cfg.Command), " ")
	cmd := &exec.Cmd{
		Dir:        program.cfg.Directory,
		Path:       progCmds[0],
		Args:       append(progCmds, program.cfg.Args...),
		Stderr:     stderr,
		Stdout:     stdout,
		ExtraFiles: program.files, // 传递文件描述符
		SysProcAttr: &syscall.SysProcAttr{
			Setpgid: true, // 设置进程组ID为自己
		},
	}

	process := &Process{
		cmd:      cmd,
		stopChan: make(chan struct{}, 1),
	}

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

		type processResult struct {
			exitCode int
			err      error
		}
		var result *processResult
		resultChan := make(chan *processResult, 1)
		go func() {
			exitCode, err := process.wait()
			resultChan <- &processResult{exitCode, err}
		}()
		select {
		// 如果进程启动之后迅速的退出，说明进程本身有问题，需要在进程启动一定的时间之后，才将重试的次数设置为0，
		// 防止进程因为异常一直重启而不会被发现
		case <-time.After(time.Second):
			// 进程运行一段时间后，才能设置为Running
			// TODO: 该时间可配置
			program.status.State = ProcessStateRunning
			program.maxRetry = 0
			program.process = process
			program.startChan <- true

			result = <-resultChan

		case result = <-resultChan:
		}

		// 进程执行完毕，可能是程序自动退出，也可能是通过stop退出
		close(process.stopChan)
		if err != nil {
			program.logger.Printf("wait error: %s", result.err.Error())
		}
		program.logger.Printf("exit with code %d", result.exitCode)
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
			// 进程异常重启的次数超过最大值，进程的状态将设置为Fatal
			program.status.State = ProcessStateFatal
			program.status.StopTime = time.Now().Unix()
			program.process = nil
			program.closeListener()
			program.startChan <- false
		}
	} else {
		program.logger.Printf("exited")
		// 进程正常的结束，状态为Exited
		program.status.State = ProcessStateExited
		program.status.StopTime = time.Now().Unix()
		program.closeListener()
		program.process = nil
	}
}

func (program *Program) RestartProess() (process *Process) {
	if program.status.State == ProcessStateStarting {
		return
	}
	program.logger.Printf("restart")
	oldProc := program.process
	program.status.State = ProcessStateStarting
	program.maxRetry = 0
	// 重启时也需要检查listener是否已经初始化
	program.initListener()
	go program.startNewProcess()
	// 保证第二个进程已经启动
	// TODO: 第二个进程启动失败时，第一个进程可以不停止，此时进程应该处于另一种状态
	<-program.startChan
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
	// program.status.Pid = 0
	program.closeListener()
	program.process = nil
	return
}

func (program *Program) stopProc(proc *Process) error {
	// TODO: 允许配置进程退出的时发送的信号量
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
