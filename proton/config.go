package proton

type Config struct {
	NetworkId uint64

	DatabaseHandles int
	DatabaseCache   int
}

var DefaultConfig = Config{
	NetworkId:       1,
	DatabaseCache:   512,
	DatabaseHandles: 256,
}
