# lax
The proxy server, which written in go. (lax settings, work powerful.)

# component
- core: core proxy only pluggable server. core has lifesycle for plugin.
  - plugins: plugin which add useful function to core. such as config save location, cache, waf, sandbox execution code env or some else. may will be written in golang.
  - request lifecycle: able to add additional process for each request timing.
    - on proxy viewer request from viewer
    - on proxy request to origin
    - on proxy received request from origin
    - on proxy response request to viewer
  - core server lifecycle:
    - on launch: eg, load settings infomation as configured.
- edge: edge server, distributed around the world. based on core & some plugin, such as cache
  - manage data:
    - customers configuration (proxy settings)
    - cache entity
    - auth info for primary & edge server communication
- controller: manager of edge server. manage data for customer
  - manage data:
    - customers information
      - auth setting
      - cash setting etc...
    - auth info for primary & edge server communication
    - customers configuration
    - edge server's working information at that time only (all log information store to another db, like big query)
      - cache purge processing status
      - edge server deployment processing etc..

edge server and primary server communication will implemented by [grpc](https://github.com/grpc/grpc-go) (not yet).

- optimized dns: optimized dns server. this server returns nearest client server ip address
  - manage data:
    - ip address and location data

## edge
### server
- proxy server
  - handle request from client & proxy to origin or response request
- operation server
  - handle request from service provider(cdn itself or content owner) & purge or update settings
- manage server
  - may use k8s for deployment, watch server & container metrics.

### worker
- operation worker
  - execute heavy operation request, like purge cache

### data store
- mongodb

### execute codes
- (want to use) [lucet](https://github.com/bytecodealliance/lucet)

## controller
simple server which management edge server.

this server provide below
- manage customer which use cache proxy.
- manage setting request from customer, cache settings, purge request. And proxy it to edge server.
- manage edge server for update, stop, start or some.
  - want to provide like blue green deployment.
- log aggregation hub and store and proxy.

### data store
- mongodb

## optimized dns server
dns server. returns nearlest(by geolocation which infer by ip address) server ip.

# operation
## cache store
- eager cache store will support

## cache purge
- light purge
  - add delete flag to server
  - execute without worker
- hard purge
  - delete entire data
  - execute from worker
- expire from each customer cache
- expire from each path (from rule?)

## configuration
- whitelist ip & blacklist ip
- whitelist header & blacklist header

# database collections
## Customer
- user used this proxy cache
- entity exit on primary database
- this provide primary id and current data id correspondence

## Config
- create each url or its pattern
- manage hit url to origin url correspondence rule

### Rule
- like AWS cloudfront's behavior
- manage path pattern & proxy to specific path
- embedded to config document

## Cache
- belongs to one Config
- manage Cache expire time, CacheKey
- manage expire time via mongodb ttl or use gridfs for support large file.
  - ttl → cache size will limit by 16MB of document limitation of mongodb
  - gridfs → no cache limit. but ttl cannot work well, so we self implement cache deletion algorithem

# image
![lax](https://user-images.githubusercontent.com/24956031/71309612-4a8c1e80-244d-11ea-84a0-ca31f48dcb35.png)

# ref
https://blog.cloudflare.com/cloudflare-architecture-and-how-bpf-eats-the-world/
