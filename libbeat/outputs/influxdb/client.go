package influxdb

import (
	"fmt"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

type client struct {
	log           *logp.Logger
	conn          influxdb.Client
	observer      outputs.Observer
	addr          string
	username      string
	password      string
	db            string
	measurement   string
	timePrecision string
	tagFields     []string
	tagFieldsHash map[string]int
	timeField     string
}

func newClient(observer outputs.Observer, addr string, user string, pass string, db string, measurement string, timePrecision string, tagFields []string, timeField string) *client {
	hash := make(map[string]int)
	for _, f := range tagFields {
		if f != "" {
			hash[f] = 1
		}
	}

	c := &client{
		log:           logp.NewLogger("InfluxDB"),
		observer:      observer,
		addr:          addr,
		username:      user,
		password:      pass,
		db:            db,
		measurement:   measurement,
		timePrecision: timePrecision,
		tagFields:     tagFields,
		tagFieldsHash: hash,
		timeField:     timeField,
	}
	return c
}

func (c *client) Connect() (err error) {
	logp.Debug("influxdb", "Connect")

	c.conn, err = influxdb.NewHTTPClient(influxdb.HTTPConfig{Addr: c.addr, Username: c.username, Password: c.password})
	if err != nil {
		logp.Err("Failed to create HTTP conn to influxdb: %v", err)
		return err
	}

	logp.Info("Client to influxdb has created: %v", c.addr)
	return err
}

func (c *client) Close() error {
	logp.Debug("influxdb", "Close connection")
	return c.conn.Close()
}

func (c *client) Publish(batch publisher.Batch) error {
	if c == nil {
		panic("No client")
	}
	if batch == nil {
		panic("No batch")
	}

	events := batch.Events()
	c.observer.NewBatch(len(events))
	failed, err := c.publish(events)
	if err != nil {
		batch.RetryEvents(failed)
		c.observer.Failed(len(failed))
		return err
	}
	batch.ACK()
	return nil
}

func (c *client) publish(data []publisher.Event) ([]publisher.Event, error) {
	var err error

	serialized := c.serializeEvents(data)
	dropped := len(data) - len(serialized)
	c.observer.Dropped(dropped)
	if dropped > 0 {
		logp.Info("Number of dropped points: %v/%v", dropped, len(data))
	}

	if (len(serialized)) == 0 {
		return nil, nil
	}

	bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  c.db,
		Precision: c.timePrecision,
	})

	for i := 0; i < len(serialized); i++ {
		pt := serialized[i]
		bp.AddPoint(pt)
	}

	err = c.conn.Write(bp)

	if err != nil {
		logp.Err("Failed to write %d records to influxdb: %v", len(data[:len(serialized)]), err)
		for _, event := range data[:len(serialized)] {
			logp.Debug("influxdb", "Failed record: %v", event.Content)
		}
		return data[:len(serialized)], err
	}

	c.observer.Acked(len(serialized))
	return nil, nil
}

func (c *client) scanFields(originFields map[string]interface{}) (*string, map[string]string, map[string]interface{}) {
	tags := make(map[string]string)
	fields := make(map[string]interface{})
	measurement := new(string)

	for k := range originFields {
		// skip time field
		if c.timeField == k {
			continue
		}

		// Find measurement name
		if c.measurement == k {
			switch v := originFields[k].(type) {
			case string:
				measurement = &v
				continue
			default:
				logp.Warn("Unsupported metric name type: %v(%T)", v, v)
			}
		}

		_, ok := c.tagFieldsHash[k]
		if !ok {
			fields[k] = originFields[k]
			continue
		}

		// This field is a tag, need to check wether is a string
		switch v := originFields[k].(type) {
		case string:
			tags[k] = v
		case int, int8, int16, int32, int64:
			tags[k] = fmt.Sprintf("%d", v)
		default:
			logp.Warn("Unsupported tag type: %v(%T)", v, v)
		}
	}

	return measurement, tags, fields
}

func (c *client) serializeEvents(data []publisher.Event) []*influxdb.Point {
	to := make([]*influxdb.Point, 0, len(data))

	for _, d := range data {
		t := d.Content.Timestamp
		if timestamp, ok := d.Content.Fields[c.timeField]; ok {
			if v, ok := timestamp.(int64); ok {
				if c.timePrecision == "s" {
					t = time.Unix(v, 0)
				} else if c.timePrecision == "ms" {
					t = time.Unix(v/1000, (v%1000)*1000*1000)
				} else {
					logp.Warn("Unsupported time precision: %v", c.timePrecision)
				}
			}
		}

		measurement, tags, fields := c.scanFields(d.Content.Fields)
		logp.Debug("influxdb", "measurement: %s, tags: %v, fields: %v, ts:", *measurement, tags, fields, t)

		if measurement != nil {
			point, err := influxdb.NewPoint(*measurement, tags, fields, t)
			if err != nil {
				logp.Err("Encoding event failed with error: %v", err)
				return to
			}
			to = append(to, point)

		} else {
			point, err := influxdb.NewPoint(c.measurement, tags, fields, t)
			if err != nil {
				logp.Err("Encoding event failed with error: %v", err)
				return to
			}
			to = append(to, point)
		}
	}

	return to
}

func (c *client) String() string {
	return "influxdb(" + c.addr + ")"
}
