// Package handlers contains functions that check if the database contains the value from the argument
package handlers

import (
	"LinkShortener/app/general"
	"LinkShortener/app/initial"
	"fmt"
	"log"
)

// GetURL gets a long URL from a short URL
func GetURL(base *initial.General, shortURL string) (string, error) {
	urls := initial.URLs{Short: shortURL}
	longURL, err := base.Redis.Get(base.Context, shortURL).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			if err = GetFromDB(base, &urls); err != nil {
				if err.Error() == "sql: no rows in result set" {
					return urls.Long, fmt.Errorf("record not exist")
				}
				log.Printf("dbQuery:Query: %s", err)
				return "", err
			}
		}
		log.Printf("GetURL:Get: %s", err)
		return "", err
	}

	return longURL, nil
}

func GetFromDB(base *initial.General, urls *initial.URLs) error {
	err := base.MySQL.QueryRow("SELECT * FROM url WHERE shortURL = ?", urls.Short).Scan(urls.Long)
	if err != nil {
		return err
	}
	err = general.Cache(base, urls)
	return err
}
