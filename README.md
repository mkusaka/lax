# lax
The proxy server, which written in go. (lax settings, work powerful.)

# component
- edge: edge server, distributed around the world
  - manage data:
    - customers configuration (proxy settings)
    - cache entity
- primary: manager of edge server. manage data for customer
  - manage data:
    - customers information
      - auth setting
      - cash setting etc...
    - customers configuration
    - edge server's working information at that time only (all log information store to another db, like big query)
      - cache purge processing
      - edge server deployment processing etc..

edge server and primary server communication want to implement by [grpc](https://github.com/grpc/grpc-go)

## edge
### server
- proxy server
  - handle request from client & proxy to origin or response request
- operation server
  - handle request from service provider(cdn itself or content owner) & purge or update settings

### worker
- operation worker
  - execute heavy operation request, like purge cache

### data store
- mongodb

### execute codes
- (want to use) [lucet](https://github.com/bytecodealliance/lucet)

## primary
simple setting manage server.
this server provide below
- manage customer which use cache proxy.
- manage setting request from customer, cache settings, purge request. And proxy it to edge server.
- manage edge server for update, stop, start or some.
  - want to provide like blue green deployment.
- log aggregation hub and store and proxy.

# operation
## cache store

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

## CacheMeta
- create each Cache
- belongs to one Config
- has one CacheEntity
- manage Cache expire time, CacheKey, or some.

## CacheEntity
- create each Cache
- belongs to CacheMeta
- CacheEntity
- (will) manage expire time via mongodb ttl

# ref
https://blog.cloudflare.com/cloudflare-architecture-and-how-bpf-eats-the-world/
