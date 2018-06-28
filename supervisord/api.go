package supervisord

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type APIServer struct {
	*Supervisor
	CfgFilepath string
}

func (s *APIServer) ServeHTTP(l net.Listener) error {
	mu := httprouter.New()
	mu.Handle(http.MethodGet, "/status", s.getStatus)
	mu.Handle(http.MethodGet, "/reread", s.reReadConfig)
	mu.Handle(http.MethodPost, "/update", s.updatePrograms)
	mu.Handle(http.MethodPost, "/start/:name", s.startProgram)
	mu.Handle(http.MethodPost, "/stop/:name", s.stopProgram)
	mu.Handle(http.MethodPost, "/restart/:name", s.restartProgram)

	serv := http.Server{
		Handler: mu,
	}

	return serv.Serve(l)
}

type HttpResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (r *HttpResponse) ToJson() []byte {
	data, _ := json.Marshal(r)
	return data
}

func (s *APIServer) restartProgram(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp := new(HttpResponse)
	name := params.ByName("name")
	err := s.RestartProgram(name)
	if err != nil {
		resp.Status = 1
		resp.Message = err.Error()
		w.Write(resp.ToJson())
		return
	}

	resp.Message = "success"
	w.Write(resp.ToJson())
}

func (s *APIServer) getStatus(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp := new(HttpResponse)
	status := s.GetStatus()
	resp.Message = "success"
	resp.Data = status
	w.Write(resp.ToJson())
}

func (s *APIServer) startProgram(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp := new(HttpResponse)
	name := params.ByName("name")
	if err := s.StartProgram(name); err != nil {
		resp.Status = 1
		resp.Message = err.Error()
		w.Write(resp.ToJson())
		return
	}
	resp.Message = "success"
	w.Write(resp.ToJson())
}

func (s *APIServer) stopProgram(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp := new(HttpResponse)
	name := params.ByName("name")
	if err := s.StopProgram(name); err != nil {
		resp.Status = 1
		resp.Message = err.Error()
		w.Write(resp.ToJson())
		return
	}
	resp.Message = "success"
	w.Write(resp.ToJson())
}

func (s *APIServer) updatePrograms(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp := new(HttpResponse)
	cfg, err := ParseConfigFile(s.CfgFilepath)
	if err != nil {
		resp.Status = 1
		resp.Message = err.Error()
		w.Write(resp.ToJson())
		return
	}
	err = s.Reload(cfg.ProgramConfigs)
	if err != nil {
		resp.Status = 1
		resp.Message = err.Error()
		resp.Data = err
		w.Write(resp.ToJson())
		return
	}

	resp.Message = "success"
	w.Write(resp.ToJson())
}

func (s *APIServer) reReadConfig(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp := new(HttpResponse)
	cfg, err := ParseConfigFile(s.CfgFilepath)
	if err != nil {
		resp.Status = 1
		resp.Message = err.Error()
		w.Write(resp.ToJson())
		return
	}

	ins, dels, ups := s.Supervisor.Diff(cfg.ProgramConfigs)

	data := make(map[string]interface{})
	if len(ins) != 0 {
		data["inserts"] = ins
	}
	if len(dels) != 0 {
		data["deletes"] = dels
	}
	if len(ups) != 0 {
		data["updates"] = ups
	}
	resp.Data = data
	resp.Message = "success"
	w.Write(resp.ToJson())
}
