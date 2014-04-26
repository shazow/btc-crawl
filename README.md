# btc-crawl

Bitcoin node network crawler (written in golang).

This is a for-fun project to explore the Bitcoin protocol and network.

Current status: 
* It crawls with all kinds of nice parameters but stores everything in memory
  until dumping a giant JSON blob at the end.
* --It crawls from hard-coded values and spits a bunch of stuff to
stdout.~~


## Usage

```
$ go get github.com/shazow/btc-crawl
$ btc-crawl
...
```


## Todo

(In approximate order of priority)

* Apply peer-age filter to results
* Stream JSON rather than accumulate into a giant array.
* Add timeout option.
* Graceful cleanup on Ctrl+C
* Namespace packages properly (outside of `main`)
