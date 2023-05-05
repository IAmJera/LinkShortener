// Package puturl contains functions that write records to the database
package puturl

import (
	"context"
	"database/sql"
	"fmt"
	"go.uber.org/zap"
	"short_url/initial"
	"short_url/sup"
)

// WriteToDB writes the entry to the database if it doesn't already exist. Otherwise, return nil
func WriteToDB(base *initial.General, urls *initial.URLs) error {
	if exist, err := isExist(base, urls); err != nil {
		return err
	} else if exist {
		base.Log.Infof("record already exist")
		return nil
	}
	err := writeRecord(base, urls)
	return err
}

func isExist(base *initial.General, urls *initial.URLs) (bool, error) {
	longURL, err := base.Redis.Get(base.Context, urls.Short).Result()
	if err != nil {
		if err.Error() != "redis: nil" {
			return false, err
		}
		return isExistInDB(base, urls)

	} else if isRecurrent(urls, &longURL) {
		base.Log.Infof("recurrent record")
		if err = recurrentCache(base, urls); err != nil {
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
		base.Log.Infof("recurrent record")
		if err = recurrentDB(base, urls); err != nil {
			return false, err
		}
		return false, nil
	}

	urls.Long = longURL
	if err = sup.Cache(base, urls); err != nil {
		return false, err
	}
	return true, nil
}

func getFromDB(base *initial.General, urls *initial.URLs) (string, error) {
	rows, err := base.MySQL.Query("SELECT * FROM url WHERE shortURL = ?", urls.Short)
	if err != nil {
		return "", err
	}
	defer closeRow(rows, base.Log)

	var short, long string
	if rows.Next() {
		if err := rows.Scan(&short, &long); err != nil {
			return "", err
		}
		return long, nil
	}

	if err = rows.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("not exist")
}

func recurrentDB(base *initial.General, urls *initial.URLs) error {
	for {
		base.Log.Debugf("salting hash")
		urls.Short = sup.Hash(urls.Short, "salt")
		rows, err := base.MySQL.Query("SELECT * FROM url WHERE shortURL = ?", urls.Short)
		if err != nil {
			return err
		}

		var short, long string
		if rows.Next() {
			if err := rows.Scan(&short, &long); err != nil {
				closeRow(rows, base.Log)
				return err
			}
		}

		if short != urls.Short {
			closeRow(rows, base.Log)
			return nil
		}
		closeRow(rows, base.Log)
	}
}

func recurrentCache(base *initial.General, urls *initial.URLs) error {
	for {
		base.Log.Debugf("salting hash")
		urls.Short = sup.Hash(urls.Short, "salt")
		if _, err := base.Redis.Get(base.Context, urls.Short).Result(); err != nil {
			if err.Error() == "redis: nil" {
				return nil
			}
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
		return fmt.Errorf("impossible insert record: %s", err)
	}

	base.Log.Infof("record added to database")
	err := sup.Cache(base, urls)
	return err
}

func closeRow(rows *sql.Rows, l *zap.SugaredLogger) {
	err := rows.Close()
	if err != nil {
		l.Errorf("error occurred: %s", err)
		return
	}
}
