# Cloud Proxy

Simple cloud proxy for docker

## Usage

# Run proxy itself
```
docker run  --name proxy --restart=always \
            -v /var/run/docker.sock:/var/run/docker.sock \
            -v $(pwd)/certs:/usr/app/certs \
            -p 80:80 -p 443:443 -d \
            neonxp/proxy
```

# Add service to proxy

```
docker run -l "cp.host=HOST" -l "cp.port=PORT" -l "cp.tls=true" -d service
```

Here:
* `cp.host` - label sets hostname of service
* `cp.port` - label sets port that service binds (inside container)
* `cp.tls` - if this label presents, service will work over auto TLS (let's encrypt)