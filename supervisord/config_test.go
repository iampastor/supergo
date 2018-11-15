package supervisord

import "testing"

func Test_GetConfigFile(t *testing.T) {
	t.Log(getConfigFiles("../*.toml"))
}
