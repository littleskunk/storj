// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package boltdb

import (
	"time"

	"github.com/boltdb/bolt"
	"go.uber.org/zap"

	"storj.io/storj/storage"
)

type boltClient struct {
	logger *zap.Logger
	db     *bolt.DB
	Path   string
	Bucket []byte
}

const (
	// fileMode sets permissions so owner can read and write
	fileMode = 0600
	// PointerBucket is the string representing the bucket used for `PointerEntries`
	PointerBucket = "pointers"
	// OverlayBucket is the string representing the bucket used for a bolt-backed overlay dht cache
	OverlayBucket = "overlay"
)

var (
	defaultTimeout = 1 * time.Second
)

// NewClient instantiates a new BoltDB client given a zap logger, db file path, and a bucket name
func NewClient(logger *zap.Logger, path, bucket string) (storage.KeyValueStore, error) {
	db, err := bolt.Open(path, fileMode, &bolt.Options{Timeout: defaultTimeout})
	if err != nil {
		return nil, err
	}

	return &boltClient{
		logger: logger,
		db:     db,
		Path:   path,
		Bucket: []byte(bucket),
	}, nil
}

// Put adds a value to the provided key in boltdb, returning an error on failure.
func (c *boltClient) Put(key storage.Key, value storage.Value) error {
	c.logger.Debug("entering bolt put")
	return c.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(c.Bucket)
		if err != nil {
			return err
		}

		return b.Put(key, value)
	})
}

// Get looks up the provided key from boltdb returning either an error or the result.
func (c *boltClient) Get(pathKey storage.Key) (storage.Value, error) {
	c.logger.Debug("entering bolt get: " + string(pathKey))
	var pointerBytes []byte
	err := c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(c.Bucket)
		v := b.Get(pathKey)
		pointerBytes = v
		return nil
	})

	if err != nil {
		// TODO: log
		return nil, err
	}

	return pointerBytes, nil
}

// List returns either a list of keys for which boltdb has values or an error.
func (c *boltClient) List(startingKey storage.Key, limit storage.Limit) (storage.Keys, error) {
	c.logger.Debug("entering bolt list")
	var paths storage.Keys
	err := c.db.Update(func(tx *bolt.Tx) error {
		cur := tx.Bucket(c.Bucket).Cursor()
		var k []byte
		if startingKey == nil {
			k, _ = cur.First()
		} else {
			k, _ = cur.Seek(startingKey)
		}
		for ; k != nil; k, _ = cur.Next() {
			paths = append(paths, k)
			if limit > 0 && int(limit) == int(len(paths)) {
				break
			}
		}
		return nil
	})
	return paths, err
}

// Delete deletes a key/value pair from boltdb, for a given the key
func (c *boltClient) Delete(pathKey storage.Key) error {
	c.logger.Debug("entering bolt delete: " + string(pathKey))
	return c.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(c.Bucket).Delete(pathKey)
	})
}

// Close closes a BoltDB client
func (c *boltClient) Close() error {
	return c.db.Close()
}
