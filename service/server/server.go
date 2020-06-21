package server

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"strconv"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	api "github.com/binkynet/BinkyNet/apis/v1"
)

type Server interface {
	// Run the HTTP server until the given context is cancelled.
	Run(ctx context.Context) error
}

// Service ('s) that we offer
type Service interface {
	api.LocalWorkerConfigServiceServer
	api.LocalWorkerControlServiceServer
}

type Config struct {
	Host     string
	GRPCPort int
}

func (c Config) createTLSConfig() (*tls.Config, error) {
	return nil, nil
}

// NewServer creates a new server
func NewServer(conf Config, api Service, log zerolog.Logger) (Server, error) {
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
	api        Service
}

// Run the HTTP server until the given context is cancelled.
func (s *server) Run(ctx context.Context) error {
	// Create TLS config
	/*tlsConfig, err := s.Config.createTLSConfig()
	if err != nil {
		return err
	}*/

	// Prepare GRPC listener
	grpcAddr := net.JoinHostPort(s.Host, strconv.Itoa(s.GRPCPort))
	grpcLis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on address %s: %v", grpcAddr, err)
	}

	// Prepare GRPC server
	grpcSrv := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	api.RegisterLocalWorkerConfigServiceServer(grpcSrv, s.api)
	api.RegisterLocalWorkerControlServiceServer(grpcSrv, s.api)
	// Register reflection service on gRPC server.
	reflection.Register(grpcSrv)

	go func() {
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Fatalf("failed to serve GRPC: %v", err)
		}
	}()
	select {
	case <-ctx.Done():
		// Close server
		s.log.Debug().Msg("Closing server...")
		grpcSrv.GracefulStop()
		return nil
	}
}
