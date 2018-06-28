package supervisord

import "testing"

func Test_DiffConfig(t *testing.T) {
	oldCfgs := map[string]*ProgramConfig{
		"t1": &ProgramConfig{
			Directory:   "/home/www/t1",
			Command:     "",
			Args:        nil,
			AutoRestart: false,
		},
		"t2": &ProgramConfig{
			Directory:   "/home/www/t2",
			Command:     "",
			Args:        nil,
			AutoRestart: false,
		},
		"t3": &ProgramConfig{
			Directory:   "/home/www/t3",
			Command:     "",
			Args:        nil,
			AutoRestart: false,
		},
	}

	newCfgs := map[string]*ProgramConfig{
		"t4": &ProgramConfig{
			Directory:   "/home/www/t4",
			Command:     "",
			Args:        nil,
			AutoRestart: false,
		},
		"t2": &ProgramConfig{
			Directory:   "/home/www/t2t2",
			Command:     "",
			Args:        nil,
			AutoRestart: false,
		},
		"t3": &ProgramConfig{
			Directory:   "/home/www/t3",
			Command:     "",
			Args:        nil,
			AutoRestart: false,
		},
	}

	ins, dels, ups := diffConfigs(oldCfgs, newCfgs)
	t.Log("inserts: ")
	for name, i := range ins {
		t.Logf("%s => %+v", name, i)
	}
	t.Log("deletes: ")
	for name, i := range dels {
		t.Logf("%s => %+v", name, i)
	}
	t.Log("updates: ")
	for name, i := range ups {
		t.Logf("%s => %+v", name, i)
	}
}
