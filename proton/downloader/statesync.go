package downloader

import "github.com/czh0526/perception/common"

type stateReq struct {
}

type stateTask struct {
	attemps map[string]struct{}
}

type stateSync struct {
	d     *Downloader
	tasks map[common.Hash]*stateTask
}

func newStateSync(d *Downloader, root common.Hash) *stateSync {
	return &stateSync{
		d:     d,
		tasks: make(map[common.Hash]*stateTask),
	}
}
