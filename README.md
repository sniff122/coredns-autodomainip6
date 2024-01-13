# coredns-autodomainip6

A fork of [vlcty/coredns-auto-ipv6-ptr](https://github.com/vlcty/coredns-auto-ipv6-ptr)

Goal: Generate IPv6 AAAA records on the fly

Additional benefit: Works with known hosts.

## Examples

### Generate PTR records if not found in a zonefile

```
rdns.example.com. {
    autodomainip6 {
        suffix rdns.example.com
        ttl 60
        allowed 2001:0db8:1234:1::/64 2001:0db8:1234:2::/64
    }
    file /path/to/zone.file
    log
    errors
}

```

### Same as above but with a transferred zone
(Untested, not used in my setup - if used please report success/failure)

```
rdns.example.com. {
    autodomainip6 {
        suffix rdns.example.com
        ttl 60
        allowed 2001:0db8:1234:1::/64 2001:0db8:1234:2::/64
    }
    secondary {
        transfer from your.master.dns
    }
    log
    errors
}
```

## Order is everything!

It's necessary that `file` or `secondary` comes right after `autodomainip6`! This plugin always calls the next plugin and checks its return. It will only generate an AAAA if a negative result comes back.

## Building a ready-to-use coredns binary using Docker

Using the docker infrastructure it's easy for you to build a working binary with the plugin:

> docker build --pull --no-cache --output type=local,dest=result -f Dockerfile.build .

If everything checks out you'll find an x86_64 binary locally under `result/coredns`.



## Note

My Go skills are basically non-existent, first time dabbling into Go so expect best practices and stuff to not be followed, feel free to submit PRs!
