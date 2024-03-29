package state

import (
	"math/big"

	"github.com/czh0526/perception/common"
)

type journalEntry interface {
	revert(*StateDB)
	dirtied() *common.Address
}

type journal struct {
	entries []journalEntry
	dirties map[common.Address]int
}

func newJournal() *journal {
	return &journal{
		dirties: make(map[common.Address]int),
	}
}

func (j *journal) append(entry journalEntry) {
	j.entries = append(j.entries, entry)
	if addr := entry.dirtied(); addr != nil {
		j.dirties[*addr]++
	}
}

// 回退到 snapshot 之前的 entries
func (j *journal) revert(statedb *StateDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		j.entries[i].revert(statedb)

		if addr := j.entries[i].dirtied(); addr != nil {
			if j.dirties[*addr]--; j.dirties[*addr] == 0 {
				delete(j.dirties, *addr)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}

type (
	createObjectChange struct {
		account *common.Address
	}
	resetObjectChange struct {
		prev *stateObject
	}
	balanceChange struct {
		account *common.Address
		prev    *big.Int
	}
	nonceChange struct {
		account *common.Address
		prev    uint64
	}
	codeChange struct {
		account            *common.Address
		prevcode, prevhash []byte
	}
	storageChange struct {
		account       *common.Address
		key, prevalue common.Hash
	}
)

func (ch createObjectChange) dirtied() *common.Address {
	return ch.account
}

func (ch createObjectChange) revert(s *StateDB) {
	delete(s.stateObjects, *ch.account)
	delete(s.stateObjectsDirty, *ch.account)
}

func (ch resetObjectChange) dirtied() *common.Address {
	return nil
}

func (ch resetObjectChange) revert(s *StateDB) {
	s.setStateObject(ch.prev)
}

func (ch nonceChange) dirtied() *common.Address {
	return ch.account
}

func (ch nonceChange) revert(s *StateDB) {
	s.getStateObject(*ch.account).setNonce(ch.prev)
}

func (ch balanceChange) dirtied() *common.Address {
	return ch.account
}

func (ch balanceChange) revert(s *StateDB) {
	s.getStateObject(*ch.account).setBalance(ch.prev)
}

func (ch codeChange) dirtied() *common.Address {
	return ch.account
}

func (ch codeChange) revert(s *StateDB) {
	s.getStateObject(*ch.account).setCode(common.BytesToHash(ch.prevhash), ch.prevcode)
}

func (ch storageChange) dirtied() *common.Address {
	return ch.account
}

func (ch storageChange) revert(s *StateDB) {
	s.getStateObject(*ch.account).setState(ch.key, ch.prevalue)
}
