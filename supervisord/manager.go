package supervisord

import (
	"errors"
	"log"
	"reflect"
	"sync"
)

type Supervisor struct {
	porgrams map[string]*Program
	lock     sync.RWMutex
	cfg      *SupervisorConfig
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
	supervisor.lock.Lock()
	defer supervisor.lock.Unlock()
	prog, err = NewProgram(name, progCfg)
	supervisor.porgrams[name] = prog
	supervisor.cfg.ProgramConfigs[name] = progCfg
	return
}

func (supervisor *Supervisor) StartProgram(name string) error {
	supervisor.lock.Lock()
	defer supervisor.lock.Unlock()
	prog, ok := supervisor.porgrams[name]
	if !ok {
		return ErrProgramNotFound
	}
	prog.StartProcess()
	return nil
}

func (supervisor *Supervisor) StopProgram(name string) error {
	supervisor.lock.Lock()
	defer supervisor.lock.Unlock()
	prog, ok := supervisor.porgrams[name]
	if !ok {
		return ErrProgramNotFound
	}
	prog.StopProcess()
	return nil
}

func (supervisor *Supervisor) RestartProgram(name string) error {
	supervisor.lock.Lock()
	defer supervisor.lock.Unlock()
	prog, ok := supervisor.porgrams[name]
	if !ok {
		return ErrProgramNotFound
	}
	prog.RestartProess()
	return nil
}

func (supervisor *Supervisor) DeleteProgram(name string) error {
	supervisor.lock.Lock()
	defer supervisor.lock.Unlock()
	prog, ok := supervisor.porgrams[name]
	delete(supervisor.porgrams, name)
	delete(supervisor.cfg.ProgramConfigs, name)
	if !ok {
		return ErrProgramNotFound
	}
	prog.StopProcess()
	prog.Destory()
	return nil
}

func (supervisor *Supervisor) UpdateProgram(name string, progCfg *ProgramConfig) (prog *Program, err error) {
	supervisor.lock.Lock()
	defer supervisor.lock.Unlock()
	prog, ok := supervisor.porgrams[name]
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
	supervisor.porgrams[name] = newProg
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
	supervisor.lock.Lock()
	defer supervisor.lock.Unlock()
	for _, program := range supervisor.porgrams {
		program.StopProcess()
		program.Destory()
	}
}

func (supervisor *Supervisor) GetStatus() []*ProgramStatus {
	supervisor.lock.RLock()
	defer supervisor.lock.RUnlock()
	status := make([]*ProgramStatus, 0, len(supervisor.porgrams))
	for _, prog := range supervisor.porgrams {
		s := prog.Status()
		status = append(status, s)
	}
	return status
}

func (supervisor *Supervisor) Reload(cfgs map[string]*ProgramConfig) error {
	inserts, deletes, updates := supervisor.Diff(cfgs)
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
	supervisor.cfg.ProgramConfigs = cfgs
	return nil
}

// 对比新旧配置，返回新增，删除和更新了的项目
func (supervisor *Supervisor) Diff(newCfgs map[string]*ProgramConfig) (
	inserts map[string]*ProgramConfig,
	deletes map[string]*ProgramConfig,
	updates map[string]*ProgramConfig) {
	oldCfgs := supervisor.cfg.ProgramConfigs
	return diffConfigs(oldCfgs, newCfgs)
}

func diffConfigs(oldCfgs map[string]*ProgramConfig, newCfgs map[string]*ProgramConfig) (
	inserts map[string]*ProgramConfig,
	deletes map[string]*ProgramConfig,
	updates map[string]*ProgramConfig) {

	inserts = make(map[string]*ProgramConfig)
	deletes = make(map[string]*ProgramConfig)
	updates = make(map[string]*ProgramConfig)

	for name, ne := range newCfgs {
		if _, ok := oldCfgs[name]; !ok {
			inserts[name] = ne
		}
	}

	for name, ne := range oldCfgs {
		if _, ok := newCfgs[name]; !ok {
			deletes[name] = ne
		}
	}

	for name, old := range oldCfgs {
		if ne, ok := newCfgs[name]; ok {
			if !reflect.DeepEqual(old, ne) {
				updates[name] = ne
			}
		}
	}
	return
}
