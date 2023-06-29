// Package general defines general functions
package general

import (
	"LinkShortener/app/initial"
	"crypto/md5"
	"encoding/hex"
	"log"
	"os"
	"strconv"
	"time"
)

// Closer defines the interface to which all objects with the Close method correspond
type Closer interface {
	Close() error
}

// CloseFile closes the object that satisfies the Closer interface
func CloseFile(c Closer) {
	if err := c.Close; err != nil {
		log.Panicf("Error closing file")
	}
}

// Hash hashes a given string with the addition of salt
func Hash(url string, suf string) string {
	hash := md5.Sum([]byte(url + suf))
	return hex.EncodeToString(hash[:4])
}

// Cache caches the short URL and the long URL in Redis
func Cache(base *initial.General, urls *initial.URLs) error {
	dur, err := strconv.Atoi(os.Getenv("CACHE_EXPIRATION"))
	if err != nil {
		log.Panicf("config.env: \"CACHE_EXPIRATION\" must be an integer")
		return err
	}
	if err = base.Redis.Set(base.Context, urls.Short, urls.Long, time.Duration(dur)*time.Minute).Err(); err != nil {
		log.Printf("Cache:Set: %s", err)
		return err
	}
	return nil
}
