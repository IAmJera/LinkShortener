// Package geturl contains functions that check if the database contains the value from the argument.
package geturl

import (
	"database/sql"
	"fmt"
	"short_url/initial"
	"short_url/sup"
)

// GetURL accepts a basic structure and URLs. Returns a longURL value taken from DB/Cache
func GetURL(base *initial.General, shortURL string) (string, error) {
	urls := initial.URLs{Short: shortURL}
	longURL, err := base.Redis.Get(base.Context, shortURL).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			if err = getFromDB(base, &urls); err == nil {
				base.Log.Infof("get url from db")
				return urls.Long, nil
			}
		}
		return "", err
	}

	base.Log.Infof("get url from cache")
	return longURL, nil
}

func getFromDB(base *initial.General, urls *initial.URLs) error {
	if err := dbQuery(base.MySQL, urls); err != nil {
		return err
	}
	err := sup.Cache(base, urls)
	return err
}

func dbQuery(db *sql.DB, urls *initial.URLs) error {
	rows, err := db.Query("SELECT * FROM url WHERE shortURL = ?", &urls.Short)
	if err != nil {
		return err
	}

	defer func(rows *sql.Rows) {
		err = rows.Close()
	}(rows)
	if err != nil {
		return err
	}

	for rows.Next() {
		err := rows.Scan(&urls.Short, &urls.Long)
		return err

	}

	if err = rows.Err(); err != nil {
		return err
	}
	return fmt.Errorf("record not exist")
}
