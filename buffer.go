package appcache

import (
	"bytes"
)

const bufCapacity = 1024 * 1024

// BufPool - потокобезопасный пул буферов (не используется sync.Pool
// по причине того что он напрягает GC)
type BufPool struct {
	ch chan *bytes.Buffer
}

// NewBuffPool - конструктор пула bytes.Buffer с capacity 1Мб,
// аллоцирует заранее память под количество буферов, которое разумно
// указывать, как max <= runtime.NumCPU()
func NewBuffPool(max int) *BufPool {
	c := make(chan *bytes.Buffer, max)

	for i := 0; i < max; i++ {
		c <- bytes.NewBuffer(make([]byte, 0, bufCapacity))
	}

	return &BufPool{ch: c}
}

// Get - возвращает первый свободный буфер из пула
func (p *BufPool) Get() *bytes.Buffer {
	select {
	case b := <-p.ch:
		return b
	default:
		return bytes.NewBuffer(make([]byte, 0, bufCapacity))
	}
}

// Put - помещает буфер в конец пула, либо дропает,
// если количество буферов в пуле равно емкости пула,
func (p *BufPool) Put(b *bytes.Buffer) {
	select {
	case p.ch <- b: // ok
	default: // drop
	}
}
