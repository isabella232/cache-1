package cache

import (
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type KeyValue struct {
	ObjectId  bson.ObjectId `bson:"_id" json:"_id"`
	Key       string        `bson:"key" json:"key"`
	Value     interface{}   `bson:"value" json:"value"`
	CreatedAt time.Time     `bson:"createdAt" json:"createdAt"`
	ExpireAt  time.Time     `bson:"expireAt" json:"expireAt"`
}

var (
	defaultExpireDuration = time.Second * 60
)

// keyValueColl default collection name for mongoDB
const defaultKeyValueColl = "jKeyValue"

// CreateKeyValueWithExpiration creates the key-value pair with default time constants
func (m *MongoCache) createKeyValueWithExpiration(k *KeyValue) error {
	return m.createKeyValue(setDefaultDataTimes(k))
}

func (m *MongoCache) getKeyWithExpireCheck(k string) (*KeyValue, error) {
	key, err := m.getKey(k)
	if err != nil {
		return nil, err
	}

	if key.ExpireAt.Before(time.Now().UTC()) {
		if err := m.deleteKey(k); err != nil {
			return nil, err
		}
		return nil, mgo.ErrNotFound
	}

	return key, nil
}

// getKey fetches the key with its key
func (m *MongoCache) getKey(key string) (*KeyValue, error) {
	keyValue := new(KeyValue)

	query := func(c *mgo.Collection) error {
		return c.Find(bson.M{"key": key}).One(&keyValue)
	}

	err := m.run(m.CollectionName, query)
	if err != nil {
		return nil, err
	}

	return keyValue, nil
}

// UpdateKey updates the key-value in mongoDB
func (m *MongoCache) UpdateKey(selector, update bson.M) error {
	query := func(c *mgo.Collection) error {
		return c.Update(selector, bson.M{"$set": update})
	}

	return m.run(m.CollectionName, query)
}

// deleteKey removes the key-value from mongoDB
func (m *MongoCache) deleteKey(key string) error {
	selector := bson.M{"key": key}

	query := func(c *mgo.Collection) error {
		err := c.Remove(selector)
		return err
	}

	return m.run(m.CollectionName, query)
}

func (m *MongoCache) deleteExpiredKeys() error {
	var selector = bson.M{"expireAt": bson.M{
		"$lte": time.Now().UTC(),
	}}

	query := func(c *mgo.Collection) error {
		_, err := c.RemoveAll(selector)
		return err
	}

	return m.run(m.CollectionName, query)
}

func (m *MongoCache) createKeyValue(k *KeyValue) error {
	k.CreatedAt = time.Now().UTC()
	query := insertQuery(k)
	return m.run(m.CollectionName, query)
}

func setDefaultDataTimes(k *KeyValue) *KeyValue {
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now().UTC()
	}

	// ExpireAt should be in the future as time
	if k.ExpireAt.Before(time.Now().UTC()) || k.ExpireAt.IsZero() {
		k.ExpireAt = k.CreatedAt.Add(defaultExpireDuration)
	}

	return k
}

func insertQuery(data interface{}) func(*mgo.Collection) error {
	return func(c *mgo.Collection) error {
		return c.Insert(data)
	}
}

//
// MongoDB helper functions
// no need to be exported functions
//

func (m *MongoCache) close() {
	m.mongeSession.Close()
}

func (m *MongoCache) refresh() {
	m.mongeSession.Refresh()
}

func (m *MongoCache) copy() *mgo.Session {
	return m.mongeSession.Copy()
}

func (m *MongoCache) run(collection string, s func(*mgo.Collection) error) error {
	session := m.copy()
	defer session.Close()
	c := session.DB("").C(collection)
	return s(c)
}
