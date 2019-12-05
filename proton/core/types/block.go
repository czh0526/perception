package types

import (
	"io"
	"math/big"
	"sync/atomic"

	"github.com/czh0526/perception/common"
	"github.com/czh0526/perception/rlp"
	"golang.org/x/crypto/sha3"
)

var (
	EmptyRootHash = DeriveSha(Transactions{})
)

type Header struct {
	ParentHash common.Hash `json:"parentHash" gencodec:"required"`
	Number     *big.Int    `json:"number" gencodec:"required"`
	Root       common.Hash `json:"stateRoot" gencodec:"required"`
	TxHash     common.Hash `json:"transactionsRoot" gencodec:"required"`
	Time       uint64      `json:"timestamp" gencodec:"required"`
}

func (h *Header) Hash() common.Hash {
	return rlpHash(h)
}

type Body struct {
	Transactions []*Transaction
}

type Block struct {
	header       *Header
	transactions Transactions

	// caches
	hash atomic.Value
}

type extblock struct {
	Header *Header
	Txs    []*Transaction
}

type Transactions []*Transaction

func (s Transactions) Len() int { return len(s) }

func (s Transactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

func NewBlock(header *Header, txs []*Transaction, uncles []*Header) *Block {
	b := &Block{header: CopyHeader(header)}
	if len(txs) == 0 {
		b.header.TxHash = EmptyRootHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs))
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	return b
}

func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: CopyHeader(header)}
}

// 深拷贝
func CopyHeader(h *Header) *Header {
	cpy := *h
	return &cpy
}

func (b *Block) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := b.header.Hash()
	b.hash.Store(v)
	return v
}

func (b *Block) WithBody(transactions []*Transaction) *Block {
	b.transactions = make([]*Transaction, len(transactions))
	copy(b.transactions, transactions)
	return b
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewLegacyKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func (b *Block) Number() *big.Int        { return new(big.Int).Set(b.header.Number) }
func (b *Block) Time() uint64            { return b.header.Time }
func (b *Block) NumberU64() uint64       { return b.header.Number.Uint64() }
func (b *Block) Header() *Header         { return CopyHeader(b.header) }
func (b *Block) Body() *Body             { return &Body{b.transactions} }
func (b *Block) Root() common.Hash       { return b.header.Root }
func (b *Block) ParentHash() common.Hash { return b.header.ParentHash }

func (b *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extblock{
		Header: b.header,
		Txs:    b.transactions,
	})
}

func (b *Block) DecodeRLP(s *rlp.Stream) error {
	var eb extblock
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.header, b.transactions = eb.Header, eb.Txs
	return nil
}
