go-metrics-influxdb
===================

This is a reporter for the [go-metrics](https://github.com/rcrowley/go-metrics) library which will post the metrics to [InfluxDB](https://influxdb.com/). go-metrics-influxdb uses latest influxDB client via HTTP/UDP protocol.

Note
----

This is only compatible with InfluxDB 0.9+.

Usage
-----

```go
import "github.com/sevenNt/go-metrics-influxdb"

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
    "udp://localhost:8089",  // the InfluxDB address
    "",                      // UDP database is setted in your configuration file
    "",                      // your InfluxDB user
    "",                      // your InfluxDB password
)

//send guage metries(field) in registry only one time not by time interval
go influxdb.InfluxDBWithTagsV2(
    metrics.DefaultRegistry, // metrics registry
    "http://localhost:8086", // the InfluxDB address
    "mydb",                  // your InfluxDB database
    "myuser",                // your InfluxDB user
    "mypassword",            // your InfluxDB password
    "tableName",// your new table name
    "tags", // tags info,  not field
)

License
-------

go-metrics-influxdb is licensed under the MIT license. See the LICENSE file for details.
