go-metrics-influxdb
===================

This is a reporter for the [go-metrics](https://github.com/rcrowley/go-metrics) library which will post the metrics to [InfluxDB](https://influxdb.com/). go-metrics-influxdb uses latest influxDB client via HTTP/UDP protocol.

Note
----

This is only compatible with InfluxDB 0.9+.

Usage
-----

```go
import "github.com/vrischmann/go-metrics-influxdb"

// send metrics via HTTP protocol
go influxdb.InfluxDB(
    metrics.DefaultRegistry, // metrics registry
    time.Second * 10,        // interval
    "http://localhost:8086", // the InfluxDB address
    "mydb",                  // your InfluxDB database
    "myuser",                // your InfluxDB user
    "mypassword",            // your InfluxDB password
)

// send metrics via UDP protocol
go influxdb.InfluxDB(
    metrics.DefaultRegistry, // metrics registry
    time.Second * 10,        // interval
    "udp://localhost:8125",  // the InfluxDB address
    "mydb",                  // your InfluxDB database
    "",                      // your InfluxDB user
    "",                      // your InfluxDB password
)
```

License
-------

go-metrics-influxdb is licensed under the MIT license. See the LICENSE file for details.
