# netperf — a minimalistic tool for measuring network data transfer performance in Go

## Overview
`netperf` is an experimental, minimalistic, command line tool to measure performance of transferring data over the network.

It is intended to understand the penalty (if any) of developing data transfer tools in Go, as compared to tools developed in lower level languages, such as [bbcp](https://www.slac.stanford.edu/~abh/bbcp/) or [iperf](http://software.es.net/iperf/). It may also be useful for comparing the performance of exchanging data using different network protocols, such as raw TCP, TLS, HTTP(S), WebSockets, etc. under the same network conditions (e.g. bandwidth, latency, packet loss, etc.).

It consists of a client and a server. The server listens for network connections from clients (current only TCP and TLS are implemented). The client connects to the server and sends data during a specified period of time, using one or more network streams. After the data exchange period is finished, both the client and the server report on the observed throughput.

## How to use
First, start a server for listening for TCP connections:

```bash
$ netperf server -addr :5000
```

Then, start a client for sending data to the server launched in the previous step, during one minute, using 10 TCP streams:

```bash
$ netperf client -server host.example.org:5000 -duration 1m -parallel 10
netperf: duration:               1m0.000766133s
netperf: streams:                10
netperf: data volume:            277127 MB
netperf: aggregated throughput:  4619 MB/sec
netperf: per stream throughput:  462 ± 0 MB/sec
```

This is the synopsis of the command:

```
$ netperf
USAGE:
    netperf server [options]
    netperf client [options]

    netperf -help
    netperf -version

Use 'netperf -help' to get detailed information about options and examples
of usage.
```

For getting details on available options do `netperf server -help` or `netperf client -help`.

## Installation
To **build from sources**, you need the [Go programming environment](https://golang.org). Do:

```
go get -u github.com/airnandez/netperf
```

## Feedback

Your feedback is welcome. Please feel free to provide it by [opening an issue](https://github.com/airnandez/netperf/issues).

## Credits

This tool is being developed and maintained by Fabio Hernandez at [IN2P3 / CNRS computing center](http://cc.in2p3.fr) (Lyon, France).

## License
Copyright 2016 Fabio Hernandez

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
