package supervisord

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (s *Supervisor) ServeHTTP(l net.Listener) error {
	mu := httprouter.New()
	mu.Handle(http.MethodPost, "/:name/restart", s.restartProgram)

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

func (s *Supervisor) restartProgram(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
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
