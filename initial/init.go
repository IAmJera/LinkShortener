// Package initial contains the functions that initiate all services and add them to the structure
package initial

import (
	"context"
	"database/sql"
	"github.com/go-redis/redis/v8"
	"github.com/go-sql-driver/mysql"
	"log"
	"net/url"
	"os"
	"time"
)

// General contains the most commonly used service instances
type General struct {
	Context context.Context
	MySQL   *sql.DB
	Redis   *redis.Client
}

// URLs contain long and short URLs
type URLs struct {
	Short string `json:"shorturl"`
	Long  string `json:"longurl"`
}

// InitAll init all services and fill base struct
func InitAll(base *General) error {
	var err error
	if base.Context, base.Redis, err = initRedis(); err != nil {
		log.Printf("initRedis:Ping: %s", err)
		return err
	}
	if base.MySQL, err = initMySQL(); err != nil {
		return err
	}
	if err = prepareDB(base); err != nil {
		log.Printf("prepareDB: %s", err)
		return err
	}

	return nil
}

func initRedis() (context.Context, *redis.Client, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, nil, err
	}
	return ctx, rdb, nil
}

func initMySQL() (*sql.DB, error) {
	auth := mysql.Config{
		User:                 os.Getenv("MYSQL_USER"),
		Passwd:               os.Getenv("MYSQL_PASSWORD"),
		Net:                  "tcp",
		Addr:                 os.Getenv("MYSQL_ADDRESS"),
		DBName:               os.Getenv("MYSQL_DB"),
		AllowNativePasswords: true,
	}
	db, err := sql.Open("mysql", auth.FormatDSN())
	if err != nil {
		log.Printf("initMySQL:Open: %s", err)
		return nil, err
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err = db.Ping(); err != nil {
		log.Printf("initMySQL:Ping: %s", err)
		return nil, err
	}
	return db, err
}

func prepareDB(base *General) error {
	if exist, err := tableExist(base); err != nil {
		return err
	} else if exist {
		return nil
	}

	query := "CREATE TABLE url ( shorturl varchar(10), longurl text);"
	if _, err := base.MySQL.ExecContext(context.Background(), query); err != nil {
		if err.Error() != "Error 1050 (42S01): Table 'url' already exists" {
			return err
		}
	}
	return nil
}

func tableExist(base *General) (bool, error) {
	if _, err := base.MySQL.Query("SELECT * FROM url;"); err != nil {
		if err.Error() == "Error 1146 (42S02): Table 'urls.url' doesn't exist" {
			return false, nil
		}
		log.Printf("tableExist: %s", err)
		return false, err
	}
	return true, nil
}

// NotValid validate URL if enabled
func (urls *URLs) NotValid() bool {
	if os.Getenv("VALIDATE_URL") == "true" {
		if _, err := url.ParseRequestURI(urls.Long); err != nil {
			return true
		}
	}
	return false
}
