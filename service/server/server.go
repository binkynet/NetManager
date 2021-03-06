package server

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"strconv"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
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
	api.NetworkControlServiceServer
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
	api.RegisterNetworkControlServiceServer(grpcSrv, s.api)
	// Register reflection service on gRPC server.
	reflection.Register(grpcSrv)

	nctx, cancel := context.WithCancel(ctx)
	defer cancel()
	g, nctx := errgroup.WithContext(nctx)
	g.Go(func() error {
		if err := grpcSrv.Serve(grpcLis); err != nil {
			s.log.Warn().Err(err).Msg("failed to serve GRPC")
			return err
		}
		return nil
	})
	g.Go(func() error {
		return api.RegisterServiceEntry(nctx, api.ServiceTypeLocalWorkerConfig, api.ServiceInfo{
			ApiVersion: "v1",
			ApiPort:    int32(s.GRPCPort),
			Secure:     false,
		})
	})
	g.Go(func() error {
		return api.RegisterServiceEntry(nctx, api.ServiceTypeLocalWorkerControl, api.ServiceInfo{
			ApiVersion: "v1",
			ApiPort:    int32(s.GRPCPort),
			Secure:     false,
		})
	})
	g.Go(func() error {
		return api.RegisterServiceEntry(nctx, api.ServiceTypeNetworkControl, api.ServiceInfo{
			ApiVersion: "v1",
			ApiPort:    int32(s.GRPCPort),
			Secure:     false,
		})
	})
	g.Go(func() error {
		// Wait for content cancellation
		select {
		case <-ctx.Done():
			// Stop
		case <-nctx.Done():
			// Stop
		}
		// Close server
		s.log.Debug().Msg("Closing server...")
		grpcSrv.GracefulStop()
		cancel()
		return nil
	})
	if err := g.Wait(); err != nil {
		s.log.Debug().Err(err).Msg("Wait failed")
		return err
	}
	return nil
}
