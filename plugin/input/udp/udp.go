package udp

import (
	"net"
	"sync"

	"github.com/ozontech/file.d/fd"
	"github.com/ozontech/file.d/metric"
	"github.com/ozontech/file.d/pipeline"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Plugin struct {
	config                  *Config
	params                  *pipeline.InputPluginParams
	readBuffs               *sync.Pool
	eventBuffs              *sync.Pool
	controller              pipeline.InputPluginController
	totalMessagesCounter    prometheus.Counter
	parsedMessagesCounter   prometheus.Counter
	unparsedMessagesCounter prometheus.Counter
	stopCh                  chan struct{}
}

func init() {
	fd.DefaultPluginRegistry.RegisterInput(&pipeline.PluginStaticInfo{
		Type:    "udp",
		Factory: Factory,
	})
}

func Factory() (pipeline.AnyPlugin, pipeline.AnyConfig) {
	return &Plugin{}, &Config{}
}

// ! config-params
// ^ config-params
type Config struct {
	Port            string `json:"port" default:":514"`              // *
	InputBufferSize int    `json:"input_buffer_size" default:"1024"` // *
	Workers         int    `json:"workers" default:"10"`
}

func (p *Plugin) Start(config pipeline.AnyConfig, params *pipeline.InputPluginParams) {

	p.controller = params.Controller
	p.config = config.(*Config)
	p.params = params
	p.registerMetrics(params.MetricCtl)

	processor := NewProcessor(p.config.Workers, func(bytes []byte, offset int64) {
		p.controller.In(pipeline.SourceID(0), "udp", pipeline.NewOffsets(offset, nil), bytes, true, nil)
	}, params.Logger)
	p.startUDPHandler(processor)

}

func (p *Plugin) Stop() {
	p.stopCh <- struct{}{}
}

func (p *Plugin) Commit(event *pipeline.Event) {
	// We are UDP inputter - we don't need this
}

func (p *Plugin) PassEvent(event *pipeline.Event) bool {
	return true
}

func (p *Plugin) registerMetrics(ctl *metric.Ctl) {
	p.parsedMessagesCounter = ctl.RegisterCounter("parsed_messages", "Messages, that were processed by UDP INPUT plugin")
	p.unparsedMessagesCounter = ctl.RegisterCounter("unparsed_messages", "Messages, that were not parsed by UDP INPUT plugin")
	p.totalMessagesCounter = ctl.RegisterCounter("total_messages", "Messages, that processed by UDP plugin")

}

func (p *Plugin) startUDPHandler(processor *Processor) {
	// Определяем порт для прослушивания

	go func() {
		addr, err := net.ResolveUDPAddr("udp", p.config.Port)
		if err != nil {
			p.params.Logger.Error("Error during resolve address port", zap.String("address", p.config.Port), zap.Error(err))
			p.stopCh <- struct{}{}
			return
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			p.params.Logger.Error("Error during opening port", zap.String("address", p.config.Port), zap.Error(err))
			p.stopCh <- struct{}{}
			return
		}
		defer conn.Close()
		p.params.Logger.Info("Opening UDP port", zap.String("address", p.config.Port))
		buffer := make([]byte, 65535)
		offset := int64(0)
		processor.Start()
		for {

			select {
			case <-p.stopCh:
				p.params.Logger.Info("Stop message received", zap.Error(err))
				return
			default:
			}

			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				p.params.Logger.Info("Error reading Packet", zap.Error(err))
				continue
			}
			offset++
			processor.PutEvent(buffer[:n])
		}
	}()

}

type Offset int64

func (o Offset) Current() int64 {
	return int64(o)
}

func (o Offset) ByStream(_ string) int64 {
	panic("unimplemented")
}
