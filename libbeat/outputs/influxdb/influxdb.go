package influxdb

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
)

type influxdbOut struct {
	beat beat.Info
}

var debugf = logp.MakeDebug("influxdb")

func init() {
	outputs.RegisterType("influxdb", makeInfluxdb)
}

func makeInfluxdb(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	var err error
	config := defaultConfig
	if err = cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	_, err = tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return outputs.Fail(err)
	}

	client := newClient(observer, config.Addr, config.Username, config.Password, config.Db, config.Measurement, config.TimePrecision, config.SendAsTags, config.SendAsTime)

	// return outputs.SuccessNet(config.LoadBalance, config.BulkMaxSize, config.MaxRetries, client)
	return outputs.Success(config.BulkMaxSize, config.MaxRetries, client)
}
