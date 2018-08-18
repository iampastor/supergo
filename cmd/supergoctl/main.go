package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/iampastor/supervisor/supervisord"
)

var (
	urlAddr string

	usage = `Usage of supergoctl:

-url 127.0.0.1:22106 // supergo的地址

Commands:

supergoctl status
supergoctl reread
supergoctl update
supergoctl start <prog>
supergoctl stop <prog>
supergoctl restart <prog>
`
)

func init() {
	v := flag.Bool("version", false, "print version info & exit")
	flag.StringVar(&urlAddr, "url", "http://127.0.0.1:22106", "supergo server url address")

	flag.Parse()

	if *v {
		PrintVersion()
		os.Exit(0)
	}
}

var client = &http.Client{
	Timeout: time.Second * 30,
}

func main() {
	if flag.NArg() == 1 {
		cmd := flag.Arg(0)
		switch cmd {
		case "status":
			status()
		case "reread":
			reread()
		case "update":
			update()
		default:
			fmt.Fprintf(os.Stderr, usage)
		}
	} else if flag.NArg() == 2 {
		cmd := flag.Arg(0)
		name := flag.Arg(1)
		switch cmd {
		case "start":
			start(name)
		case "stop":
			stop(name)
		case "restart":
			restart(name)
		default:
			fmt.Fprintf(os.Stderr, usage)
		}
	} else {
		fmt.Fprintf(os.Stderr, usage)
	}
}

type ApiResponse struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func get(cmd string) (*ApiResponse, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s", urlAddr, cmd))
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	apiResp := new(ApiResponse)
	json.Unmarshal(data, apiResp)
	if apiResp.Status != 0 {
		return nil, errors.New(apiResp.Message)
	}
	return apiResp, nil
}

func post(cmd string, name string) (*ApiResponse, error) {
	resp, err := client.PostForm(fmt.Sprintf("%s/%s/%s", urlAddr, cmd, name), nil)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	apiResp := new(ApiResponse)
	json.Unmarshal(data, apiResp)
	if apiResp.Status != 0 {
		return nil, errors.New(apiResp.Message)
	}
	return apiResp, nil
}

func status() {
	resp, err := get("status")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	var progStatus []*supervisord.ProgramStatus
	if err := json.Unmarshal(resp.Data, &progStatus); err != nil {
		fmt.Fprintf(os.Stderr, err.Error()+string(resp.Data))
	}
	for _, ps := range progStatus {
		if ps.State == supervisord.ProcessStateRunning {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%-10s\t%-8s\tpid %-5d\tstart at %s", ps.Name, ps.State, ps.Pid,
				time.Unix(ps.StartTime, 0).Format("2006-01-02 15:04:05")))
		} else if ps.State == supervisord.ProcessStateStopped {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%-10s\t%-8s\tpid %-5d\tstop at  %s", ps.Name, ps.State, ps.Pid,
				time.Unix(ps.StopTime, 0).Format("2006-01-02 15:04:05")))
		} else {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%-10s\t%-8s\tpid %-5d\t%s", ps.Name, ps.State, ps.Pid,
				time.Unix(ps.StopTime, 0).Format("2006-01-02 15:04:05")))
		}
	}
}

func reread() {
	resp, err := get("reread")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	changes := make(map[string]map[string]*supervisord.ProgramConfig)
	json.Unmarshal(resp.Data, &changes)
	ins := changes["inserts"]
	dels := changes["deletes"]
	ups := changes["updates"]
	for name, _ := range ins {
		fmt.Fprintln(os.Stderr, name, "add")
	}
	for name, _ := range dels {
		fmt.Fprintln(os.Stderr, name, "delete")
	}
	for name, _ := range ups {
		fmt.Fprintln(os.Stderr, name, "update")
	}
}

func update() {
	resp, err := post("update", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	fmt.Fprintln(os.Stderr, string(resp.Message))
}

func start(name string) {
	resp, err := post("start", name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	fmt.Fprintln(os.Stderr, string(resp.Message))
}

func stop(name string) {
	resp, err := post("stop", name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	fmt.Fprintln(os.Stderr, string(resp.Message))
}

func restart(name string) {
	resp, err := post("restart", name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	fmt.Fprintln(os.Stderr, string(resp.Message))
}
