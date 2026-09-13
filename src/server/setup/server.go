package setup

import (
	"net/http"
	"server/tools"
	"time"

	u "github.com/quollix/common/utils"
)

type Server struct {
	Mux *http.ServeMux
}

func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         ":" + tools.Port,
		Handler:      u.RecoverHttpPanics(s.Mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	u.Logger.Info("server starting", tools.PortField, tools.Port)
	err := srv.ListenAndServe()
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}
