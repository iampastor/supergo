package supervisord

import "testing"

func Test_ParseConfig(t *testing.T) {
	data := `
	[program.test]
	directory = "/home/www"
	command = "/home/www/hello"
	args = ["world"]
	auto_restart = true
	stdout_file = "/tmp/hello.log"
	stderr_file = "/tmp/hello.err"
	max_retry = 3
	listen_addrs = [":1084"]
	stop_timeout = 10
	`
	cfg, err := ParseConfigString(data)
	if err != nil {
		t.Error(err)
	}
	for name, c := range cfg.ProgramConfigs {
		t.Logf("%s: %+v", name, *c)
	}
}
