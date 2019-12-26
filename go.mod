module gitlab.com/akita/mgpusim

require (
	github.com/DataDog/zstd v1.4.4 // indirect
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.8.1 // indirect
	github.com/rs/xid v1.2.1
	github.com/tebeka/atexit v0.1.0
	github.com/vbauerster/mpb/v4 v4.11.2
	gitlab.com/akita/akita v1.10.0
	gitlab.com/akita/gcn3 v1.6.1 // indirect
	gitlab.com/akita/mem v1.8.0
	gitlab.com/akita/noc v1.3.3
	gitlab.com/akita/util v0.3.0
	go.mongodb.org/mongo-driver v1.2.0 // indirect
	golang.org/x/crypto v0.0.0-20191219195013-becbf705a915 // indirect
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/sys v0.0.0-20191224085550-c709ea063b76 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
	gopkg.in/yaml.v2 v2.2.7 // indirect
)

// replace gitlab.com/akita/akita => ../akita

// replace gitlab.com/akita/noc => ../noc

// replace gitlab.com/akita/mem => ../mem
//
// replace gitlab.com/akita/util => ../util

go 1.13
