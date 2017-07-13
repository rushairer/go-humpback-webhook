# go-humpback-webhook

#### 1. 安装etcd
假设安装etcd的服务器内网IP为 172.16.1.171

``` shell
yum install etcd
```
#### 2. 修改etcd配置

```
vim /etc/etcd/etcd.conf
```

``` shell
#修改以下内容
ETCD_LISTEN_CLIENT_URLS="http://172.16.1.171:2379"
ETCD_ADVERTISE_CLIENT_URLS="http://172.16.1.171:2379"
ETCD_CORS="*"
```

``` shell
#启动etcd
systemctl daemon-reload
systemctl enable etcd
systemctl start etcd

#检查是否启动成功
etcdctl cluster-health

```

#### 2. 初始化haproxy-discover配置键值

``` shell

curl -XPUT http://172.16.1.171:2379/v2/keys/haproxy-discover/services\?dir\=true
curl -XPUT http://172.16.1.171:2379/v2/keys/haproxy-discover/tcp-services\?dir\=true

```

#### 3. 启动haproxy服务

``` shell
docker run -d -it \
-p 80:8080 \
--restart=always \
-e ETCD_NODE=172.16.1.171:2379 \
--name gateway \
yaronr/haproxy-confd:latest
```

#### 4. 启动humpback-webhook服务

假如运行webhook的服务器内网IP为172.16.1.180

``` shell
docker run -d -it \
-p 8080:8080 \
--restart=always \
-e HUMPBACKWEBHOOK_TOKEN=yourtokenstring \
-e HUMPBACKWEBHOOK_ETCD=http://172.16.1.171:2379 \
-e HUMPBACKWEBHOOK_CENTER_PORT=8500 \
--name humpback-webhook \
io84/go-humpback-webhook:latest
```

webhook地址为 http://172.16.1.180:8080/webhook.php
