module github.com/binkynet/NetManager

go 1.13

replace github.com/coreos/go-systemd => github.com/coreos/go-systemd v0.0.0-20190620071333-e64a0ec8b42a

require (
	github.com/binkynet/BinkyNet v0.0.0-20200621102753-38dfa13f5789
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/juju/errgo v0.0.0-20140925100237-08cceb5d0b53 // indirect
	github.com/mattn/go-pubsub v0.0.0-20160821075316-7a151c7747cd
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.0 // indirect
	github.com/pulcy/go-terminate v0.0.0-20160630075856-d486fe7ee814
	github.com/rs/zerolog v1.18.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	google.golang.org/grpc v1.27.1
	gopkg.in/yaml.v2 v2.3.0
)
