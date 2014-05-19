# btc-crawl

Bitcoin node network crawler (written in golang).

This is a for-fun project to explore the Bitcoin protocol and network.

Current status: 
* JSON streaming is in place, and graceful shutdown.
* ~~It crawls with all kinds of nice parameters but stores everything in memory
  until dumping a giant JSON blob at the end.~~
* ~~It crawls from hard-coded values and spits a bunch of stuff to
stdout.~~


## Usage

```
$ go get github.com/shazow/btc-crawl
$ btc-crawl --help
...
$ btc-crawl \
  --concurrency=100 \
  --output="btc-crawl.json" \
  --peer-age="24h" \
  --user-agent="/batman:1.0/" \
  --verbose
...
```

**Estimated crawl time:** Unknown.

There should be under 10,000 active network nodes at any given time, according
to [bitnodes.io](https://getaddr.bitnodes.io/). Each node returns around ~2,500
known nodes, but usually only several have timestamps within the last hour.


## Todo

(In approximate order of priority)

* Namespace useful sub-packages properly (outside of `main`)
* Fix `go build -race` race condition warnings. (Not sure if this is feasible. `golog` and other
  fundamental pieces seem to trigger warnings possibly erroneously.)
* Tests would be nice.


## License

MIT (see LICENSE file).
