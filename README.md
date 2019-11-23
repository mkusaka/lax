# lax
The proxy server, which written in go. (lax settings, work powerful.)

# component
## server
- proxy server
  - handle request from client & proxy to origin or response request
- operation server
  - handle request from service provider(cdn itself or content owner) & purge or update settings

## worker
- operation worker
  - execute heavy operation request, like purge cache

## data store
- mongodb

## execute codes
- (want to use) [lucet](https://github.com/bytecodealliance/lucet)

# operation
## cache store

## cache purge
- light purge
  - add delete flag to server
  - execute without worker
- hard purge
  - delete entire data
  - execute from worker

## configuration
- whitelist ip & blacklist ip
- whitelist header & blacklist header
