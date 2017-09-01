package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"

	"github.com/binkynet/BinkyNet/model"
	"github.com/julienschmidt/httprouter"
	restkit "github.com/pulcy/rest-kit"
	"github.com/rs/zerolog"
)

type Server interface {
	// Run the HTTP server until the given context is cancelled.
	Run(ctx context.Context) error
}

type API interface {
	// Get the configuration for a specific local worker
	GetWorkerConfig(ctx context.Context, workerID string) (model.LocalConfiguration, error)
}

type Config struct {
	Host string
	Port int
}

func (c Config) createTLSConfig() (*tls.Config, error) {
	return nil, nil
}

// NewServer creates a new server
func NewServer(conf Config, api API, log zerolog.Logger) (Server, error) {
	return &server{
		Config:     conf,
		log:        log.With().Str("component", "server").Logger(),
		requestLog: log.With().Str("component", "server.requests").Logger(),
		api:        api,
	}, nil
}

type server struct {
	Config
	log        zerolog.Logger
	requestLog zerolog.Logger
	api        API
}

// Run the HTTP server until the given context is cancelled.
func (s *server) Run(ctx context.Context) error {
	mux := httprouter.New()
	mux.NotFound = http.HandlerFunc(s.notFound)
	mux.GET("/worker/:id/config", s.handleGetWorkerConfig)

	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	tlsConfig, err := s.Config.createTLSConfig()
	if err != nil {
		return maskAny(err)
	}

	serverErrors := make(chan error)
	go func() {
		defer close(serverErrors)
		if tlsConfig != nil {
			s.log.Info().Msgf("Listening on %s using TLS", addr)
			httpServer.TLSConfig = tlsConfig
			tlsConfig.BuildNameToCertificate()
			if err := httpServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				serverErrors <- maskAny(err)
			}
		} else {
			s.log.Info().Msgf("Listening on %s", addr)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverErrors <- maskAny(err)
			}
		}
	}()

	select {
	case err := <-serverErrors:
		return maskAny(err)
	case <-ctx.Done():
		// Close server
		s.log.Debug().Msg("Closing server...")
		httpServer.Close()
		return nil
	}
}

func handleError(w http.ResponseWriter, err error) {
	errResp := restkit.NewErrorResponseFromError(err)
	sendJSON(w, errResp.HTTPStatusCode(), errResp)
}

func parseBody(r *http.Request, data interface{}) error {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return maskAny(err)
	}
	if err := json.Unmarshal(body, data); err != nil {
		return maskAny(restkit.BadRequestError(err.Error(), 0))
	}
	return nil
}

// sendJSON encodes given body as JSON and sends it to the given writer with given HTTP status.
func sendJSON(w http.ResponseWriter, status int, body interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		w.Write([]byte("{}"))
	} else {
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(body); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
