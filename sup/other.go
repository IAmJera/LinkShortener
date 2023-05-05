// Package sup contains simple but commonly used functions
package sup

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"short_url/initial"
	"strconv"
	"time"
)

// Hash concatenates the arguments, creates a md5 hash of them, and returns the first 5 characters
func Hash(url string, suf string) string {
	hash := md5.Sum([]byte(url + suf))
	return hex.EncodeToString(hash[:4])
}

// Cache writes long and short urls to the DB
func Cache(base *initial.General, urls *initial.URLs) error {
	dur, err := strconv.Atoi(os.Getenv("CACHE_EXPIRATION"))
	if err != nil {
		return fmt.Errorf("error occurred: .env: \"CACHE_EXPIRATION\" must be an integer")
	}
	if err := base.Redis.Set(base.Context, urls.Short, urls.Long, time.Duration(dur)*time.Minute).Err(); err != nil {
		return err
	}
	base.Log.Infof("record added to cache")
	return nil
}
