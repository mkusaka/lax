package db

import (
	"context"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/mkusaka/lax/utils"

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
		"mongodb://localhost:27017",
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

// TODO: create indices
// DISCUSS: should we
type TimeMeta struct {
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

func NewTimeMeta() *TimeMeta {
	now := time.Now()
	return &TimeMeta{
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type Customer struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	TimeMeta TimeMeta           `bson:"time_meta" json:"time_meta"`

	// to primary server pluggable, this field just declare as string.
	PrimaryCustomerID string `bson:"primary_customer_id" json:"primary_customer_id"`
}

func (c *Client) NewCustomer(primaryCustomerID string) (*mongo.InsertOneResult, error) {
	customer := Customer{
		ID:                primitive.NewObjectID(),
		PrimaryCustomerID: primaryCustomerID,
		TimeMeta:          *NewTimeMeta(),
	}
	return c.database.Collection("customer").InsertOne(c.defaultContext, customer)
}

// TODO: implement upsert
// this method may not needed
func (c *Client) UpdateCustomer(primaryCustomerID string, updatePrimaryCustomerID string) (*mongo.UpdateResult, error) {
	filter := bson.M{"primary_customer_id": primaryCustomerID}
	update := bson.M{
		"$set": bson.M{
			"primary_customer_id": updatePrimaryCustomerID,
			"time_meta":           bson.M{"updated_at": time.Now()},
		},
	}
	return c.database.Collection("customer").UpdateOne(c.defaultContext, filter, update)
}

func (c *Client) DeleteCustomer(primaryCustomerID string) (*mongo.DeleteResult, error) {
	filter := bson.M{"primary_customer_id": primaryCustomerID}
	return c.database.Collection("customer").DeleteOne(c.defaultContext, filter)
}

func (c *Client) GetCustomer(primaryCustomerID string) Customer {
	var customer Customer
	filter := bson.M{"primary_customer_id": primaryCustomerID}
	result := c.database.Collection("customer").FindOne(c.defaultContext, filter)
	result.Decode(&customer)
	return customer
}

type CacheKeyConfig struct {
	HeaderKeys []string `bson:"header_keys" json:"header_keys"` // key
	UseURL     bool     `bson:"use_url" json:"use_url"`         // use url or not
}

func NewCacheKeyConfig(headerKeys []string, useURL bool) *CacheKeyConfig {
	return &CacheKeyConfig{
		HeaderKeys: headerKeys,
		UseURL:     useURL,
	}
}

// url proxy rule. if request path matches priority high Matcher, then proxy to Priority.
// Marcher support only all matcher suffix like foo/bar/* matches under foo/bar/ path.
type Rule struct {
	Macher    string `bson:"macher" json:"macher"`
	Matched   string `bson:"mached" json:"matched"`
	Priority  int    `bson:"priority" json:"priority"`
	IsDefault bool   `bson:"is_default" json:"is_default"`
}

func IsGeneralPattern(s string) bool {
	return strings.HasSuffix(s, "*")
}

func (r *Rule) IsGeneralMatcherPattern() bool {
	return IsGeneralPattern(r.Macher)
}

func (r *Rule) IsGeneralMatchedPattern() bool {
	return IsGeneralPattern(r.Matched)
}

// foo/bar/baz/*→ foo/bar/baz/
// is this process called normalize???
func NormalizedPath(s string) string {
	return strings.Replace(s, "*", "", 1)
}

func (r *Rule) Match(path string) bool {
	if r.IsGeneralMatcherPattern() {
		return strings.Contains(path, strings.Replace(r.Macher, "*", "", 1))
	} else {
		return strings.Contains(path, r.Macher)
	}
}

type ProxyURL = string

// convert path to proxy url as defined rule
// @example
//  rule: foo/bar/* → yo/*
//  path: foo/bar/123/456
//  returns: yo/123/456
//  rule: foo/bar/* → yo
//  path: foo/bar/123/345
//  returns: yo
//  rule: foo/bar → yo/* // this is invalid rule. rule can't detect right path.
func (r *Rule) RuledPath(path string) (ProxyURL, error) {
	if r.Macher == "" || r.Matched == "" {
		return "", errors.New("invalid rule pattern")
	}

	if !r.IsGeneralMatchedPattern() {
		return r.Matched, nil
	}

	if r.IsGeneralMatcherPattern() {
		// path: foo/bar/baz
		// 1. foo/bar/* → foo/bar/
		nMatcher := NormalizedPath(r.Macher)
		// 2. foo/bar/baz → baz
		nPath := strings.Replace(path, nMatcher, "", 1)
		// 3. yo/* → yo/
		nMatched := NormalizedPath(r.Matched)
		// 4. yo/baz
		return nMatched + nPath, nil
	}
	// in this line, rule has like foo/bar → yo/*. this is invalid pattern. (rule can't detect right path.)
	return "", errors.New("invalid rule pattern")
}

func NewRule(macher string, matched string) (*Rule, error) {
	if macher == "" || matched == "" {
		return nil, errors.New("invalid rule")
	}
	return &Rule{
		Macher:  macher,
		Matched: matched,
	}, nil
}

type Rules []Rule

// returns priority order desc
func (r *Rules) HighPriority() Rules {
	rules := *r
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
	return rules
}

// returns matched rule. if there is no match rule, returns default rule or empty rule struct.
func (r *Rules) MatchRule(path string) *Rule {
	sortedRules := r.HighPriority()
	defaultRule := Rule{}
	for _, rule := range sortedRules {
		if rule.Match(path) {
			return &rule
		} else if rule.IsDefault {
			defaultRule = rule
		}
	}

	return &defaultRule
}

func (r *Rules) ProxyPath(path string) (string, error) {
	return r.MatchRule(path).RuledPath(path)
}

// create each domain. manage proxy rule via Rule struct.
type Config struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	TimeMeta       TimeMeta           `bson:"time_meta" json:"time_meta"`
	CustomerID     primitive.ObjectID `bson:"customer_id" json:"customer_id"` // TODO: make this field's index
	Domain         string             `bson:"domain" json:"domain"`           // TODO: make this field's uniq index
	ProxyDomain    string             `bson:"proxy_domain" json:"proxy_domain"`
	CacheKeyConfig `bson:"cache_key_config" json:"cache_key_config"`
	Rules          `bson:"rules" json:"rules"`
}

func (c *Config) RequestURL(schema, path string) string {
	return schema + "://" + c.Domain + path
}

func (c *Config) Key(schema, path string) string {
	keys := c.HeaderKeys
	sort.Slice(keys, func(i, j int) bool {
		return true
	})
	if c.CacheKeyConfig.UseURL {
		keys = append(keys, c.RequestURL(schema, path))
	}
	return utils.Key(keys)
}

func (c *Client) NewConfig(customer *Customer, domain string, proxyDomain string, cacheKeyConfig *CacheKeyConfig, rules *Rules) *Config {
	config := Config{
		ID:             primitive.NewObjectID(),
		TimeMeta:       *NewTimeMeta(),
		CustomerID:     customer.ID,
		Domain:         domain,
		ProxyDomain:    proxyDomain,
		CacheKeyConfig: *cacheKeyConfig,
		Rules:          *rules,
	}
	return &config
}

func (c *Client) SaveConfig(config *Config) (*mongo.InsertOneResult, error) {
	return c.database.Collection("config").InsertOne(c.defaultContext, *config)
}

func (c *Client) GetConfig(configID primitive.ObjectID) *Config {
	filter := bson.M{"_id": configID}
	var config Config
	c.database.Collection("config").FindOne(c.defaultContext, filter).Decode(&config)
	return &config
}

func (c *Client) GetConfigFromDomain(domain string) (*Config, error) {
	filter := bson.M{"domain": domain}
	var config Config
	c.database.Collection("config").FindOne(c.defaultContext, filter).Decode(&config)
	if config.ID == primitive.NilObjectID {
		return nil, errors.New("not found")
	}
	return &config, nil
}

type CacheMeta struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	TimeMeta TimeMeta           `bson:"time_meta" json:"time_meta"`
	EntityID primitive.ObjectID `bson:"entity_id" json:"entity_id"`
	ConfigID primitive.ObjectID `bson:"config_id" json:"config_id"`
	CacheKey string             `bson:"cache_key" json:"cache_key"`
	Expire   time.Time          `bson:"expire" json:"expire"` // for stale-if-error control, hold expire time
}

func (c *Client) GetCacheMeta(configID primitive.ObjectID, cacheKey string) *CacheMeta {
	filter := bson.M{
		"config_id": configID,
		"cache_key": cacheKey,
	}
	var cacheMeta CacheMeta
	c.database.Collection("cache_meta").FindOne(c.defaultContext, filter).Decode(&cacheMeta)
	return &cacheMeta
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

func (c *Client) GetCacheEntity(metaID primitive.ObjectID) *CacheEntity {
	filter := bson.M{"meta_id": metaID}
	var entity CacheEntity
	c.database.Collection("cache_entity").FindOne(c.defaultContext, filter).Decode(&entity)
	return &entity
}

func GenerateResponseFromCache(cache *CacheEntity) (*http.Response, error) {
	return &http.Response{
		Header: cache.Headers,
		Body:   ioutil.NopCloser(bytes.NewReader(cache.Body)),
	}, nil
}

// you should check expire before cache
func (c *Client) StoreCache(meta CacheMeta, r *http.Response, expireAt time.Time) {
	// cacheMeta := c.database.Collection("cache").FindOne(c.defaultContext, filter)
	// cacheMeta.Decode(&meta)
	entityID := meta.EntityID
	entityFilter := bson.M{"_id": entityID}
	entity := EntityFromResponse(r, entityID)
	// upsert entity
	result, err := c.database.Collection("cache_entity").UpdateOne(c.defaultContext, entityFilter, entity)
	if err != nil {
		log.Fatal(err)
	}

	if meta.EntityID != result.UpsertedID {
		// update meta entity id if created new cache entity
		// FIXME: move to worker?
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
