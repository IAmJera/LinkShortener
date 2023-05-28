// Package puturl contains functions that write records to the database
package puturl

import (
	"LinkShortener/general"
	"LinkShortener/initial"
	"context"
	"database/sql"
	"fmt"
	"log"
)

// WriteToDB writes the entry to the database if it doesn't already exist. Otherwise, return nil
func WriteToDB(base *initial.General, urls *initial.URLs) error {
	if exist, err := isExist(base, urls); err != nil {
		return err
	} else if exist {
		return nil
	}
	err := writeRecord(base, urls)
	return err
}

func isExist(base *initial.General, urls *initial.URLs) (bool, error) {
	longURL, err := base.Redis.Get(base.Context, urls.Short).Result()
	if err != nil {
		if err.Error() != "redis: nil" {
			log.Printf("isExist: %s", err)
			return false, err
		}
		return isExistInDB(base, urls)

	} else if isRecurrent(urls, &longURL) {
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
	longURL, err := getFromDB(base, urls)
	if err != nil {
		if err.Error() == "not exist" {
			return false, nil
		}
		return false, err
	}

	if isRecurrent(urls, &longURL) {
		if err = recurrentDB(base, urls); err != nil {
			return false, err
		}
		return false, nil
	}

	urls.Long = longURL
	if err = general.Cache(base, urls); err != nil {
		return false, err
	}
	return true, nil
}

func getFromDB(base *initial.General, urls *initial.URLs) (string, error) {
	rows, err := base.MySQL.Query("SELECT * FROM url WHERE shortURL = ?", urls.Short)
	if err != nil {
		log.Printf("getFromDB:Query: %s", err)
		return "", err
	}
	defer func(rows *sql.Rows) {
		if err = rows.Close(); err != nil {
			log.Printf("error closing file")
		}
	}(rows)

	var short, long string
	if rows.Next() {
		if err = rows.Scan(&short, &long); err != nil {
			log.Printf("getFromDB:Scan: %s", err)
			return "", err
		}
		return long, nil
	}

	if err = rows.Err(); err != nil {
		log.Printf("getFromDB:Err: %s", err)
		return "", err
	}
	return "", fmt.Errorf("not exist")
}

func recurrentDB(base *initial.General, urls *initial.URLs) error {
	for {
		urls.Short = general.Hash(urls.Short, "salt")
		rows, err := base.MySQL.Query("SELECT * FROM url WHERE shortURL = ?", urls.Short)
		if err != nil {
			log.Printf("recurrentDB:Query: %s", err)
			//general.CloseFile(rows)
			return err
		}

		var short, long string
		if rows.Next() {
			if err = rows.Scan(&short, &long); err != nil {
				log.Printf("recurrentDB:Next: %s", err)
				general.CloseFile(rows)
				return err
			}
		}

		if short != urls.Short {
			general.CloseFile(rows)
			return nil
		}
		general.CloseFile(rows)
	}
}

func recurrentCache(base *initial.General, urls *initial.URLs) error {
	for {
		urls.Short = general.Hash(urls.Short, "salt")
		if _, err := base.Redis.Get(base.Context, urls.Short).Result(); err != nil {
			if err.Error() == "redis: nil" {
				return nil
			}
			log.Printf("recurrentCache:Get: %s", err)
			return err
		}
	}
}

func isRecurrent(urls *initial.URLs, longURL *string) bool {
	if *longURL != urls.Long {
		return true
	}
	return false
}

func writeRecord(base *initial.General, urls *initial.URLs) error {
	query := "INSERT INTO `url` (`shorturl`, `longurl`) VALUES (?, ?)"
	if _, err := base.MySQL.ExecContext(context.Background(), query, urls.Short, urls.Long); err != nil {
		log.Printf("writeRecord:Query: %s", err)
		return fmt.Errorf("impossible insert record: %s", err)
	}

	err := general.Cache(base, urls)
	return err
}
