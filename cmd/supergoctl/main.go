package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var (
	urlAddr string

	usage = `Usage of supergoctl:

-url 127.0.0.1:22106

Commands:

	supergoctl status
	supergoctl reread
	supergoctl update
	supergoctl start prog
	supergoctl stop prog
	supergoctl restart prog
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
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func status() {
	resp, err := client.Get(urlAddr + "/status")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	apiResp := new(ApiResponse)
	json.Unmarshal(data, apiResp)
	fmt.Fprintln(os.Stderr, apiResp)
}

func reread() {
	resp, err := client.Get(urlAddr + "/reread")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	apiResp := new(ApiResponse)
	json.Unmarshal(data, apiResp)
	fmt.Fprintln(os.Stderr, apiResp)
}

func update() {
	resp, err := client.PostForm(urlAddr+"/update", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	apiResp := new(ApiResponse)
	json.Unmarshal(data, apiResp)
	fmt.Fprintln(os.Stderr, apiResp)
}

func start(name string) {
	resp, err := client.PostForm(urlAddr+"/start/"+name, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	apiResp := new(ApiResponse)
	json.Unmarshal(data, apiResp)
	fmt.Fprintln(os.Stderr, apiResp)
}

func stop(name string) {
	resp, err := client.PostForm(urlAddr+"/stop/"+name, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	apiResp := new(ApiResponse)
	json.Unmarshal(data, apiResp)
	fmt.Fprintln(os.Stderr, apiResp)
}

func restart(name string) {
	resp, err := client.PostForm(urlAddr+"/restart/"+name, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	apiResp := new(ApiResponse)
	json.Unmarshal(data, apiResp)
	fmt.Fprintln(os.Stderr, apiResp)
}
