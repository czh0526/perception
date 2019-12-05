module github.com/czh0526/perception

go 1.12

require (
	github.com/allegro/bigcache v1.0.0
	github.com/aristanetworks/goarista v0.0.0-20191023202215-f096da5361bb
	github.com/btcsuite/btcd v0.20.0-beta
	github.com/coreos/go-semver v0.3.0
	github.com/go-stack/stack v1.8.0
	github.com/gogo/protobuf v1.3.1
	github.com/google/uuid v1.1.1
	github.com/gorilla/websocket v1.4.1
	github.com/hashicorp/golang-lru v0.5.3
	github.com/huin/goupnp v1.0.0
	github.com/ipfs/go-cid v0.0.3
	github.com/ipfs/go-datastore v0.1.1
	github.com/ipfs/go-ipfs-util v0.0.1
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/go-todocounter v0.0.2
	github.com/jackpal/gateway v1.0.5
	github.com/jackpal/go-nat-pmp v1.0.1
	github.com/jbenet/go-temp-err-catcher v0.0.0-20150120210811-aac704a3f4f2
	github.com/jbenet/goprocess v0.1.3
	github.com/koron/go-ssdp v0.0.0-20180514024734-4a0ed625a78b
	github.com/libp2p/go-addr-util v0.0.1
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/libp2p/go-conn-security-multistream v0.1.0
	github.com/libp2p/go-eventbus v0.1.0
	github.com/libp2p/go-flow-metrics v0.0.1
	github.com/libp2p/go-libp2p v0.3.1
	github.com/libp2p/go-libp2p-autonat v0.1.0
	github.com/libp2p/go-libp2p-circuit v0.1.3
	github.com/libp2p/go-libp2p-core v0.2.3
	github.com/libp2p/go-libp2p-discovery v0.1.0
	github.com/libp2p/go-libp2p-kad-dht v0.0.0-20191022103404-9c020873aceb
	github.com/libp2p/go-libp2p-kbucket v0.2.1
	github.com/libp2p/go-libp2p-loggables v0.1.0
	github.com/libp2p/go-libp2p-mplex v0.2.1
	github.com/libp2p/go-libp2p-nat v0.0.4
	github.com/libp2p/go-libp2p-peerstore v0.1.3
	github.com/libp2p/go-libp2p-record v0.1.1
	github.com/libp2p/go-libp2p-routing v0.1.0
	github.com/libp2p/go-libp2p-secio v0.2.0
	github.com/libp2p/go-libp2p-swarm v0.2.2
	github.com/libp2p/go-libp2p-transport-upgrader v0.1.1
	github.com/libp2p/go-libp2p-yamux v0.2.1
	github.com/libp2p/go-maddr-filter v0.0.5
	github.com/libp2p/go-mplex v0.1.0
	github.com/libp2p/go-msgio v0.0.4
	github.com/libp2p/go-nat v0.0.3
	github.com/libp2p/go-openssl v0.0.3
	github.com/libp2p/go-reuseport v0.0.1
	github.com/libp2p/go-reuseport-transport v0.0.2
	github.com/libp2p/go-stream-muxer-multistream v0.2.0
	github.com/libp2p/go-tcp-transport v0.1.1
	github.com/libp2p/go-ws-transport v0.1.2
	github.com/libp2p/go-yamux v1.2.4
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/minio/sha256-simd v0.1.1
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/mr-tron/base58 v1.1.2
	github.com/multiformats/go-base32 v0.0.3
	github.com/multiformats/go-multiaddr v0.1.1
	github.com/multiformats/go-multiaddr-dns v0.2.0
	github.com/multiformats/go-multiaddr-fmt v0.1.0
	github.com/multiformats/go-multiaddr-net v0.1.1
	github.com/multiformats/go-multibase v0.0.1
	github.com/multiformats/go-multihash v0.0.8
	github.com/multiformats/go-multistream v0.1.0
	github.com/naoina/go-stringutil v0.1.0
	github.com/naoina/toml v0.0.0-20170918210437-9fafd6967416
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/spacemonkeygo/spacelog v0.0.0-20180420211403-2296661a0572
	github.com/spaolacci/murmur3 v1.1.0
	github.com/steakknife/bloomfilter v0.0.0-20180922174646-6819c0d2a570
	github.com/steakknife/hamming v0.0.0-20180906055917-c99c65617cd3 // indirect
	github.com/stretchr/testify v1.4.0 // indirect
	github.com/syndtr/goleveldb v1.0.0
	github.com/urfave/cli v1.22.1
	github.com/whyrusleeping/base32 v0.0.0-20170828182744-c30ac30633cc
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1
	github.com/whyrusleeping/go-logging v0.0.1 // indirect
	github.com/whyrusleeping/mafmt v1.2.8
	github.com/whyrusleeping/mdns v0.0.0-20190826153040-b9b60ed33aa9 // indirect
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7
	go.opencensus.io v0.22.1
	go.uber.org/atomic v1.4.0
	go.uber.org/multierr v1.2.0
	go.uber.org/zap v1.11.0
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	google.golang.org/appengine v1.4.0 // indirect
)
