package supervisord

import (
	"errors"
	"log"
	"sync"
)

type Supervisor struct {
	porgrams map[string]*Program
	lock     sync.RWMutex
	cfg      *SupervisorConfig
}

type SupervisorConfig struct {
	ProgramConfigs map[string]*ProgramConfig `toml:"program"`
}

var (
	ErrProgramNotFound = errors.New("program not found")
)

func NewSupervisor(cfg *SupervisorConfig) *Supervisor {
	s := &Supervisor{
		porgrams: make(map[string]*Program),
		cfg:      cfg,
	}

	return s
}

func (supervisor *Supervisor) AddProgram(name string, progCfg *ProgramConfig) (prog *Program, err error) {
	prog, err = NewProgram(name, progCfg)
	supervisor.lock.Lock()
	supervisor.porgrams[name] = prog
	supervisor.lock.Unlock()
	return
}

func (supervisor *Supervisor) StartProgram(name string) error {
	supervisor.lock.RLock()
	prog, ok := supervisor.porgrams[name]
	supervisor.lock.RUnlock()
	if !ok {
		return ErrProgramNotFound
	}
	prog.StartProcess()
	return nil
}

func (supervisor *Supervisor) StopProgram(name string) error {
	supervisor.lock.RLock()
	prog, ok := supervisor.porgrams[name]
	supervisor.lock.RUnlock()
	if !ok {
		return ErrProgramNotFound
	}
	prog.StopProcess()
	return nil
}

func (supervisor *Supervisor) RestartProgram(name string) error {
	supervisor.lock.RLock()
	prog, ok := supervisor.porgrams[name]
	supervisor.lock.RUnlock()
	if !ok {
		return ErrProgramNotFound
	}
	prog.RestartProess()
	return nil
}

func (supervisor *Supervisor) DeleteProgram(name string) error {
	supervisor.lock.RLock()
	prog, ok := supervisor.porgrams[name]
	supervisor.lock.Unlock()
	if !ok {
		return ErrProgramNotFound
	}
	prog.StopProcess()
	prog.Destory()
	return nil
}

func (supervisor *Supervisor) UpdateProgram(name string, progCfg *ProgramConfig) (prog *Program, err error) {
	supervisor.lock.RLock()
	prog, ok := supervisor.porgrams[name]
	supervisor.lock.RUnlock()
	if !ok {
		return nil, ErrProgramNotFound
	}
	prog.StopProcess()
	prog.Destory()
	newProg, err := NewProgram(name, progCfg)
	if err != nil {
		return
	}
	newProg.StartProcess()
	supervisor.lock.Lock()
	supervisor.porgrams[name] = newProg
	supervisor.lock.Unlock()
	return
}

func (supervisor *Supervisor) GetProgram(name string) *Program {
	supervisor.lock.RLock()
	defer supervisor.lock.RUnlock()
	return supervisor.porgrams[name]
}

func (supervisor *Supervisor) ListPrograms() []*Program {
	supervisor.lock.RLock()
	defer supervisor.lock.RUnlock()
	progs := make([]*Program, 0, len(supervisor.porgrams))
	for _, prog := range supervisor.porgrams {
		progs = append(progs, prog)
	}
	return progs
}

func (supervisor *Supervisor) Exit() {
	for _, program := range supervisor.porgrams {
		program.StopProcess()
		program.Destory()
	}
}

func (supervisor *Supervisor) GetStatus() []*ProgramStatus {
	status := make([]*ProgramStatus, 0, len(supervisor.porgrams))
	for name, prog := range supervisor.porgrams {
		s := &ProgramStatus{
			Name:  name,
			State: string(prog.GetState()),
		}
		status = append(status, s)
	}
	return status
}

func (supervisor *Supervisor) Reload(cfgs map[string]*ProgramConfig) error {
	inserts, deletes, updates := supervisor.diff(supervisor.cfg.ProgramConfigs, cfgs)
	for name, _ := range deletes {
		err := supervisor.DeleteProgram(name)
		if err != nil {
			log.Printf("delete program %s errors: %s", name, err.Error())
		}
	}

	for name, cfg := range inserts {
		p, err := supervisor.AddProgram(name, cfg)
		if err != nil {
			log.Printf("add program %s error %s", name, err.Error())
		}
		p.StartProcess()
	}

	for name, cfg := range updates {
		_, err := supervisor.UpdateProgram(name, cfg)
		if err != nil {
			log.Printf("update program %s error: %s", name, err.Error())
		}
	}

	return nil
}

func (supervisor *Supervisor) diff(oldCfgs map[string]*ProgramConfig, newCfgs map[string]*ProgramConfig) (
	inserts map[string]*ProgramConfig,
	deletes map[string]*ProgramConfig,
	updates map[string]*ProgramConfig) {

	return
}
