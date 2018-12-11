package discovery

import (
	"context"
	"encoding/json"
	"net"
	"sync"

	api "github.com/binkynet/BinkyNet/discovery"
	"github.com/rs/zerolog"
)

type Config struct {
	Port int
}

type Dependencies struct {
	Log      zerolog.Logger
	Messages chan RegisterWorkerMessage
}

type Service interface {
	// Run the discovery service until the given context is cancelled.
	Run(ctx context.Context) error
}

// NewService instantiates a new Service.
func NewService(config Config, deps Dependencies) (Service, error) {
	return &service{
		Config:       config,
		Dependencies: deps,
	}, nil
}

type RegisterWorkerMessage struct {
	api.RegisterWorkerMessage
	RemoteHost string
}

type service struct {
	Config
	Dependencies
}

// Run the discovery service until the given context is cancelled.
func (s *service) Run(ctx context.Context) error {
	socket, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: s.Config.Port,
	})
	var mutex sync.Mutex

	closeSocket := func() {
		mutex.Lock()
		tmp := socket
		socket = nil
		mutex.Unlock()
		if tmp != nil {
			socket = nil
			s.Log.Debug().Msg("Closing discovery socket")
			if err := tmp.Close(); err != nil {
				s.Log.Error().Err(err).Msg("Failed to close discovery socket")
			} else {
				s.Log.Info().Msg("Closed discovery socket")
			}
		}
	}
	defer closeSocket()
	if err != nil {
		return maskAny(err)
	}
	returned := make(chan struct{})
	defer close(returned)
	go func() {
		select {
		case <-ctx.Done():
			closeSocket()
		case <-returned:
			// Done
		}
	}()
	for {
		data := make([]byte, 4096)
		n, remoteAddr, err := socket.ReadFromUDP(data)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			s.Log.Error().Err(err).Msg("Failed to read")
		} else {
			var msg api.RegisterWorkerMessage
			if err := json.Unmarshal(data[0:n], &msg); err != nil {
				s.Log.Error().Err(err).Msg("Invalid message: JSON decode failed")
			} else {
				s.Log.Debug().Str("remote", remoteAddr.String()).Msg("Received request")
				s.Messages <- RegisterWorkerMessage{
					RegisterWorkerMessage: msg,
					RemoteHost:            remoteAddr.IP.String(),
				}
			}
		}
	}
}
