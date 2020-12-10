package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	admission "k8s.io/api/admission/v1"

	"github.com/supriya-premkumar/gandalf/types"
)

// Server implements REST Server interface
type Server struct {
	ctx context.Context
	log *logrus.Entry
	srv *http.Server
	rtr *mux.Router

	adm               types.AdmissionReviewer
	certPath, keyPath string
}

// NewRESTServer returns a new instance of the REST API
func NewRESTServer(ctx context.Context, logger *logrus.Logger, adm types.AdmissionReviewer,
	port int, certPath, keyPath string) *Server {
	rtr := mux.NewRouter()
	return &Server{
		ctx: ctx,
		log: logger.WithField("component", types.FixedWidthFormatter("api")),
		srv: &http.Server{
			Addr:         fmt.Sprintf("0.0.0.0:%d", port),
			WriteTimeout: types.DefaultHTTPWriteTimeout,
			ReadTimeout:  types.DefaultHTTPReadTimeout,
			IdleTimeout:  types.DefaultHTTPIdleTimeout,
			Handler:      rtr,
		},
		rtr: rtr,

		adm:      adm,
		certPath: certPath,
		keyPath:  keyPath,
	}
}

// Start registers API Routes and starts Server on the specified port
func (s *Server) Start() error {
	s.log.Infof("Listening for admission reviews on %s", s.srv.Addr)
	s.rtr.HandleFunc("/v1/ping", s.debugHandler).Methods("GET")
	s.rtr.HandleFunc("/v1/api/admission/review", s.admissionHandler).Methods("POST")
	go func() {
		if err := s.srv.ListenAndServeTLS(s.certPath, s.keyPath); err != nil {
			s.log.Errorf("Failed to start REST Server. Err: %v", err)
		} else {
			s.log.Info("REST Server running!")
		}
	}()

	return nil
}

func (s *Server) debugHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(types.APIResponse{
		Status:  http.StatusText(http.StatusOK),
		Message: "PONG",
	})
}

func (s *Server) admissionHandler(w http.ResponseWriter, r *http.Request) {
	var req admission.AdmissionReview
	dat, err := ioutil.ReadAll(r.Body)

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		// TODO Increment Err count here
		s.log.Errorf("Failed to read admission review request. Err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(types.APIResponse{
			Status:  http.StatusText(http.StatusInternalServerError),
			Message: fmt.Sprintf("failed to read admission review request. Err: %v", err),
		})
		return
	}

	if err := json.Unmarshal(dat, &req); err != nil {
		// TODO Increment Err count here
		s.log.Errorf("Failed to unmarshal admission review request. Err: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.APIResponse{
			Status:  http.StatusText(http.StatusBadRequest),
			Message: fmt.Sprintf("failed to unmarshal admission review request. Err: %v", err),
		})
		return
	}

	resp, err := s.adm.Review(&req)
	if err != nil && resp == nil {
		// TODO Increment Err count here
		s.log.Errorf("gandalf failed to review request. Err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(types.APIResponse{
			Status:  http.StatusText(http.StatusInternalServerError),
			Message: fmt.Sprintf("gandalf failed to review request. Err: %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	if resp != nil && resp.Allowed == true {
		// TODO Increment Admit count here
		s.log.Infof("gandalf admitted %v", req.Request.Kind)
	} else {
		// TODO Increment Reject count here
		s.log.Infof("gandalf rejected %v", req.Request.Kind)
	}

	json.NewEncoder(w).Encode(&admission.AdmissionReview{
		TypeMeta: metav1.TypeMeta{},
		Request:  nil,
		Response: resp,
	})

	return
}

// Stop gracefully stops the REST Server
func (s *Server) Stop() {
	if err := s.srv.Shutdown(s.ctx); err != nil {
		s.log.Errorf("Failed to gracefully shutdown server. Err: %v", err)
	}
	s.log.Info("Successfully stopped REST Server")
}
