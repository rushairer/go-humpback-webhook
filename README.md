# go-humpback-webhook

## How to use?

``` shell
docker run -d -it \
-p 8080:8080 \
--restart=always \
-e HUMPBACKWEBHOOK_TOKEN=1324ea1b59b0d9a3661f96dd7d4aa0a1 \
-e HUMPBACKWEBHOOK_ETCD=http://172.16.1.171:2379 \
-e HUMPBACKWEBHOOK_CENTER_PORT=8500 \
--name humpback-webhook \
io84/go-humpback-webhook:latest
```
