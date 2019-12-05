package chaindb

const IdealBatchSize = 100 * 1024

type Batch interface {
	KeyValueWriter

	ValueSize() int
	// 将写操作批量刷新进 db
	Write() error
	// 重置缓存
	Reset()
	// 将写操作在 w 上重放
	Replay(w KeyValueWriter) error
}

type Batcher interface {
	NewBatch() Batch
}
