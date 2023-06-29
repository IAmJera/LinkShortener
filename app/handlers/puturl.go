// Package handlers contains functions that check if the database contains the value from the argument
package handlers

import (
	"LinkShortener/app/general"
	"LinkShortener/app/initial"
	"fmt"
	"log"
	"os"
)

// WriteToDB accepts a basic structure and URLs. Writes a record to DB/Cache
func WriteToDB(base *initial.General, urls *initial.URLs) error {
	exist, err := isExist(base, urls)
	if err != nil {
		log.Printf("WriteToDB:isExist: %s", err)
		return err
	}

	if exist {
		return fmt.Errorf("record already exist")
	}
	err = writeRecord(base, urls)
	return err
}

func isExist(base *initial.General, urls *initial.URLs) (bool, error) {
	longURL, err := base.Redis.Get(base.Context, urls.Short).Result()
	if err != nil {
		if err.Error() != "redis: nil" {
			log.Printf("isExist:Get: %s", err)
			return false, err
		}
		return isExistInDB(base, urls)
	}

	if isRecurrent(urls, longURL) {
		if err = recurrentCache(base, urls); err != nil {
			log.Printf("isExist:recurrentCache: %s", err)
			return false, err
		}
		return false, nil
	}

	urls.Long = longURL
	return true, nil
}

func isExistInDB(base *initial.General, urls *initial.URLs) (bool, error) {
	longURL := urls.Long
	err := GetFromDB(base, urls)
	if err != nil {
		if err.Error() == "sql: no rows in result set" { //"not exist" {
			return false, nil
		}
		return false, err
	}

	if isRecurrent(urls, longURL) {
		if err = recurrentDB(base, urls); err != nil {
			return false, err
		}
		return false, nil
	}

	if err = general.Cache(base, urls); err != nil {
		return false, err
	}
	return true, nil
}

func isRecurrent(urls *initial.URLs, longURL string) bool {
	if longURL != urls.Long {
		return true
	}
	return false
}

func recurrentDB(base *initial.General, urls *initial.URLs) error {
	for {
		urls.Short = general.Hash(urls.Short, os.Getenv("SALT"))
		var short string
		err := base.MySQL.QueryRow("SELECT * FROM url WHERE shortURL = ?", urls.Short).Scan(&short)
		if err != nil {
			log.Printf("recurrentDB:Query: %s", err)
			return err
		}

		if short != urls.Short {
			return nil
		}
	}
}

func recurrentCache(base *initial.General, urls *initial.URLs) error {
	for {
		urls.Short = general.Hash(urls.Short, os.Getenv("SALT"))
		if _, err := base.Redis.Get(base.Context, urls.Short).Result(); err != nil {
			if err.Error() == "redis: nil" {
				return nil
			}
			log.Printf("recurrentCache:Get: %s", err)
			return err
		}
	}
}

func writeRecord(base *initial.General, urls *initial.URLs) error {
	query := "INSERT INTO `url` (`shorturl`, `longurl`) VALUES (?, ?)"
	if _, err := base.MySQL.Exec(query, urls.Short, urls.Long); err != nil {
		log.Printf("writeRecord:Query: %s", err)
		return fmt.Errorf("impossible insert record: %s", err)
	}

	err := general.Cache(base, urls)
	return err
}
