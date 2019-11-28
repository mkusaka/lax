package db

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	client *mongo.Client

	database *mongo.Database

	Logger *log.Logger

	defaultContext context.Context
}

func NewClient(timeout time.Duration) *Client {
	contextTimeout := timeout * time.Second
	ctx, _ := context.WithTimeout(context.Background(), contextTimeout)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(
		"mongodb://localhost:27017?w=majority",
	))

	if err != nil {
		log.Fatal(err)
	}

	// operation database
	return &Client{
		client:         client,
		database:       client.Database("lax_cache"),
		defaultContext: ctx,
	}
}

type Customer struct {
	ID primitive.ObjectID `json:"_id,omitempty"`
	// make this field configable for make another manage server from official.
	PrimaryCustomerID string `json:"primary_customer_id"`
}

type Config struct {
	ID         primitive.ObjectID `json:"_id,omitempty"`
	CustomerID primitive.ObjectID `json:"customer_id"`
	ProxyURL   string             `json:"proxy_url"`
	OriginURL  string             `json:"origin_url"`
}

type CacheMeta struct {
	ID       primitive.ObjectID `json:"_id,omitempty"`
	EntityID primitive.ObjectID `json:"entity_id"`
	CacheKey string             `json:"cache_key"`
	Expire   time.Time          `json:"expire"`
}

/**
 * @return current time is equal or after expire date, returns true.
 */
func (c *CacheMeta) IsExpired() bool {
	currentTime := time.Now()
	return currentTime.After(c.Expire) || currentTime.Equal(c.Expire)
}

func (c *CacheMeta) IsExpiredAt(at time.Time) bool {
	return at.After(c.Expire) || at.Equal(c.Expire)
}

type CacheEntity struct {
	ID     primitive.ObjectID `json:"_id,omitempty"`
	MetaID primitive.ObjectID `json:"meta_id"`
	// TODO: delete expire cache via mongodb ttl https://docs.mongodb.com/manual/core/index-ttl/
	//   - idea:
	//       - set ttl to cacheentity collection(with expire after second 0 & index: ttl)
	//       - set expire time calculated from its ttl
	// expireAt     time.Time           `json:"expire_at"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
}

func (c *Client) FetchCache(key string) {
	filter := bson.M{"key": key}
	// TODO: make model package and use it.
	c.database.Collection("cacheEntity").FindOne(c.defaultContext, filter)
}

// you should check expire before cache
func (c *Client) StoreCache(meta CacheMeta, r *http.Response, expire_at time.Time) {
	// cacheMeta := c.database.Collection("cache").FindOne(c.defaultContext, filter)
	// cacheMeta.Decode(&meta)
	entityID := meta.EntityID
	entityFilter := bson.M{"_id": entityID}
	entity := EntityFromResponse(r, entityID)
	// upsert entity
	result, err := c.database.Collection("cacheEntity").UpdateOne(c.defaultContext, entityFilter, entity)
	if err != nil {
		log.Fatal(err)
	}

	if meta.EntityID != result.UpsertedID {
		// update meta entity id if created new cache entity
		// FIXME: move to worker?
		// ここでmetaのidを持ってきたいが、取れないのだろうか？
		// filter := bson.M{"_id": meta.ID}
		// c.database.Collection("cache").UpdateOne(c.defaultContext, filter, update)
	}
}

func EntityFromResponse(r *http.Response, entityID primitive.ObjectID) *CacheEntity {
	entity := CacheEntity{
		Headers: r.Header,
	}
	if r.Body != nil {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		entity.Body = body
	}

	// TODO: Store MetaID
	return &entity
}
