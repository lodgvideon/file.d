package udp

import (
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type Processor struct {
	workers  int
	input    chan []byte
	outPutFn func(bytes []byte, offset int64)
	offset   *atomic.Int64

	logger *zap.SugaredLogger
}

func (p *Processor) Start() {
	//
	//var bufPool = sync.Pool{
	//	New: func() interface{} {
	//		return &bytes.Buffer{}
	//	},
	//}

	for i := 0; i < p.workers; i++ {
		go func() {
			for {
				select {
				case data := <-p.input:
					b := data[4:]

					clonedMessageBytes := make([]byte, len(b))
					copy(clonedMessageBytes, b)
					p.outPutFn(clonedMessageBytes, p.offset.Inc())
				}
			}
		}()
	}
}

func (p *Processor) PutEvent(data []byte) {
	p.input <- data
}

func NewProcessor(workers int, outPutFn func(
	bytes []byte,
	offset int64,
), logger *zap.SugaredLogger) *Processor {
	return &Processor{

		workers:  workers,
		input:    make(chan []byte, 1024*1024),
		outPutFn: outPutFn,
		offset:   atomic.NewInt64(0),
		logger:   logger,
	}
}
