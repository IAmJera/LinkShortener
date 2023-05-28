// Package geturl contains functions that check if the database contains the value from the argument.
package geturl

import (
	"LinkShortener/general"
	"LinkShortener/initial"
	"database/sql"
	"fmt"
	"log"
)

// GetURL accepts a basic structure and URLs. Returns a longURL value taken from DB/Cache
func GetURL(base *initial.General, shortURL string) (string, error) {
	urls := initial.URLs{Short: shortURL}
	longURL, err := base.Redis.Get(base.Context, shortURL).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			if err = getFromDB(base, &urls); err == nil {
				return urls.Long, nil
			}
		}
		log.Printf("GetURL:Get: %s", err)
		return "", err
	}

	return longURL, nil
}

func getFromDB(base *initial.General, urls *initial.URLs) error {
	if err := dbQuery(base.MySQL, urls); err != nil {
		return err
	}
	err := general.Cache(base, urls)
	return err
}

func dbQuery(db *sql.DB, urls *initial.URLs) error {
	rows, err := db.Query("SELECT * FROM url WHERE shortURL = ?", &urls.Short)
	if err != nil {
		log.Printf("dbQuery:Query: %s", err)
		return err
	}
	defer general.CloseFile(rows)

	for rows.Next() {
		err = rows.Scan(&urls.Short, &urls.Long)
		log.Printf("dbQuery:Scan: %s", err)
		return err
	}

	if err = rows.Err(); err != nil {
		log.Printf("dbQuery:Err: %s", err)
		return err
	}
	return fmt.Errorf("record not exist")
}
