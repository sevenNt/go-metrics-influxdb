package influxdb

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/rcrowley/go-metrics"
)

type reporter struct {
	reg      metrics.Registry
	interval time.Duration

	addr     string
	database string
	username string
	password string
	tags     map[string]string

	client client.Client
}

// InfluxDB starts a InfluxDB reporter which will post the metrics from the given registry at each d interval.
func InfluxDB(r metrics.Registry, d time.Duration, addr, database, username, password string) {
	InfluxDBWithTags(r, d, addr, database, username, password, nil)
}

// InfluxDBWithTags starts a InfluxDB reporter which will post the metrics from the given registry at each d interval with the specified tags
func InfluxDBWithTags(r metrics.Registry, d time.Duration, addr, database, username, password string, tags map[string]string) {
	rep := &reporter{
		reg:      r,
		interval: d,
		addr:     addr,
		database: database,
		username: username,
		password: password,
		tags:     tags,
	}
	if err := rep.makeClient(); err != nil {
		log.Printf("unable to make InfluxDB client. err=%v", err)
		return
	}

	rep.run()
}

//发送批量的metrics数据(当前为gauge)到influxDB, 非定时发送
func InfluxDBWithTagsV2(r metrics.Registry, addr, database, username, password, tableName string, tags map[string]string) {
	rep := &reporter{
		reg:      r,
		addr:     addr,
		database: database,
		username: username,
		password: password,
		tags:     tags,
	}

	if err := rep.makeClient(); err != nil {
		log.Printf("unable to make InfluxDB client. err=%v", err)
		return
	}
	rep.runV2(tableName)
}

func (r *reporter) makeClient() (err error) {
	url, err := url.Parse(r.addr)
	if err != nil {
		log.Printf("unable to parse InfluxDB address %s. err=%v", r.addr, err)
		return
	}

	switch url.Scheme {
	case "http":
		r.client, err = client.NewHTTPClient(client.HTTPConfig{
			Addr:     r.addr,
			Username: r.username,
			Password: r.password,
		})
	case "udp":
		r.client, err = client.NewUDPClient(client.UDPConfig{
			Addr:        url.Host,
			PayloadSize: 1024,
		})
	default:
		err = errors.New("invalid scheme")
	}

	return
}

func (r *reporter) runV2(tableName string) error {
	_, _, err := r.client.Ping(1 * time.Second)
	if err != nil {
		log.Printf("got error while sending a ping to InfluxDB, err=%v", err)
		return err
	}

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: r.database,
	})

	if err != nil {
		return err
	}

	r.reg.Each(func(name string, i interface{}) {
		now := time.Now()

		switch metric := i.(type) {
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			fields := map[string]interface{}{
				"value": ms.Value(),
			}
			pt, _ := client.NewPoint(fmt.Sprintf("%s.gauge", tableName), r.tags, fields, now)
			bp.AddPoint(pt)
		}
	})

	if len(bp.Points()) > 0 {
		err = r.client.Write(bp)
	} else {
		err = errors.New("no point in client")
	}
	return err
}

func (r *reporter) run() {
	intervalTicker := time.Tick(r.interval)
	pingTicker := time.Tick(time.Second * 5)

	for {
		select {
		case <-intervalTicker:
			if err := r.send(); err != nil {
				log.Printf("unable to send metrics to InfluxDB. err=%v", err)
			}
		case <-pingTicker:
			_, _, err := r.client.Ping(1 * time.Second)
			if err != nil {
				log.Printf("got error while sending a ping to InfluxDB, trying to recreate client. err=%v", err)

				if err = r.makeClient(); err != nil {
					log.Printf("unable to make InfluxDB client. err=%v", err)
				}
			}
		}
	}
}

func (r *reporter) send() error {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: r.database,
	})
	if err != nil {
		return err
	}

	r.reg.Each(func(name string, i interface{}) {
		now := time.Now()

		switch metric := i.(type) {
		case metrics.Counter:
			ms := metric.Snapshot()
			fields := map[string]interface{}{
				"value": ms.Count(),
			}
			pt, _ := client.NewPoint(fmt.Sprintf("%s.count", name), r.tags, fields, now)
			bp.AddPoint(pt)
		case metrics.Gauge:
			ms := metric.Snapshot()
			fields := map[string]interface{}{
				"value": ms.Value(),
			}
			pt, _ := client.NewPoint(fmt.Sprintf("%s.gauge", name), r.tags, fields, now)
			bp.AddPoint(pt)
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			fields := map[string]interface{}{
				"value": ms.Value(),
			}
			pt, _ := client.NewPoint(fmt.Sprintf("%s.gauge", name), r.tags, fields, now)
			bp.AddPoint(pt)
		case metrics.Histogram:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			fields := map[string]interface{}{
				"count":    ms.Count(),
				"max":      ms.Max(),
				"mean":     ms.Mean(),
				"min":      ms.Min(),
				"stddev":   ms.StdDev(),
				"variance": ms.Variance(),
				"p50":      ps[0],
				"p75":      ps[1],
				"p95":      ps[2],
				"p99":      ps[3],
				"p999":     ps[4],
				"p9999":    ps[5],
			}
			pt, _ := client.NewPoint(fmt.Sprintf("%s.histogram", name), r.tags, fields, now)
			bp.AddPoint(pt)
		case metrics.Meter:
			ms := metric.Snapshot()
			fields := map[string]interface{}{
				"count": ms.Count(),
				"m1":    ms.Rate1(),
				"m5":    ms.Rate5(),
				"m15":   ms.Rate15(),
				"mean":  ms.RateMean(),
			}
			pt, _ := client.NewPoint(fmt.Sprintf("%s.meter", name), r.tags, fields, now)
			bp.AddPoint(pt)
		case metrics.Timer:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			fields := map[string]interface{}{
				"count":    ms.Count(),
				"max":      ms.Max(),
				"mean":     ms.Mean(),
				"min":      ms.Min(),
				"stddev":   ms.StdDev(),
				"variance": ms.Variance(),
				"p50":      ps[0],
				"p75":      ps[1],
				"p95":      ps[2],
				"p99":      ps[3],
				"p999":     ps[4],
				"p9999":    ps[5],
				"m1":       ms.Rate1(),
				"m5":       ms.Rate5(),
				"m15":      ms.Rate15(),
				"meanrate": ms.RateMean(),
			}
			pt, _ := client.NewPoint(fmt.Sprintf("%s.timer", name), r.tags, fields, now)
			bp.AddPoint(pt)
		}
	})

	return r.client.Write(bp)
}
