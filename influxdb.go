package influxdb

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
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
	WithTags(r, d, addr, database, username, password, nil)
}

// WithTags starts a InfluxDB reporter which will post the metrics from the given registry at each d interval with the specified tags
func WithTags(r metrics.Registry, d time.Duration, addr, database, username, password string, tags map[string]string) {
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

func (r *reporter) makeClient() (err error) {
	url, err := url.Parse(r.addr)
	if err != nil {
		log.Printf("unable to parse InfluxDB address %s. err=%v", r.addr, err)
		return err
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

//ReporterItem 发送一条数据的内容
type ReporterItem struct {
	Reg  metrics.Registry
	Tags map[string]string
}

//Reporter 发送给数据库的client对象
type Reporter struct {
	database string
	client   client.Client
}

//NewReporter 发送批量的metrics数据(当前为gauge)到influxDB, 非定时发送
func NewReporter(address, database, username, password string) *Reporter {
	//client 只创建一次
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     address,
		Username: username,
		Password: password,
	})
	if err != nil {
		log.Printf("fail to create NewHTTPClient, err=%v", err)
		return nil
	}

	rep := &Reporter{
		database: database,
		client:   c,
	}
	return rep
}

// Send 发送一条logItem类型的数据(一个registry, 多个tag)
func (r *Reporter) Send(item *ReporterItem, measureName string) error {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: r.database,
	})

	if err != nil {
		log.Printf("fail to create batch points, err=%v", err)
		return err
	}

	st, err := strconv.Atoi(item.Tags["start_time"])
	if err != nil {
		log.Printf("fail to convert start time, err=%v", err)
		return err
	}
	delete(item.Tags, "start_time")

	item.Reg.Each(func(name string, i interface{}) {
		now := time.Now()

		switch metric := i.(type) {
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			fields := map[string]interface{}{
				"value":      ms.Value(),
				"start_time": st,
			}
			pt, _ := client.NewPoint(fmt.Sprintf("%s.gauge", measureName), item.Tags, fields, now)
			bp.AddPoint(pt)
		}
	})

	if len(bp.Points()) > 0 {
		err = r.client.Write(bp)
	} else {
		err = errors.New("no point to send")
	}
	return err
}
