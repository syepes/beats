package influxdb

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

type influxdbConfig struct {
	Timeout       time.Duration     `config:"timeout"`
	BulkMaxSize   int               `config:"bulk_max_size"`
	MaxRetries    int               `config:"max_retries"       validate:"min=-1"`
	TLS           *tlscommon.Config `config:"ssl"`
	Backoff       backoff           `config:"backoff"`
	Username      string            `config:"username"`
	Password      string            `config:"password"`
	Addr          string            `config:"addr"`
	Db            string            `config:"db"`
	Measurement   string            `config:"measurement"`
	TimePrecision string            `config:"time_precision"`
	SendAsTags    []string          `config:"send_as_tags"`
	SendAsTime    string            `config:"send_as_time"`
}

type backoff struct {
	Init time.Duration
	Max  time.Duration
}

var (
	defaultConfig = influxdbConfig{
		Timeout:     5 * time.Second,
		BulkMaxSize: 2048,
		MaxRetries:  3,
		TLS:         nil,
		Backoff: backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		Addr:          "http://localhost:8086",
		Db:            "db",
		Measurement:   "metric",
		TimePrecision: "s",
	}
)

func (c *influxdbConfig) Validate() error {
	return nil
}
