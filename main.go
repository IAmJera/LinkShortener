// main starts the web server and handles the POST/GET methods
package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"net/http"
	"os"
	"short_url/geturl"
	"short_url/initial"
	"short_url/puturl"
	"short_url/sup"
	"strings"
)

func main() {
	base := initial.General{}
	if err := godotenv.Load(); err != nil {
		if _, err := fmt.Fprintf(os.Stderr, "error occurred: %s", err); err != nil {
			return
		}
		os.Exit(1)
	}

	logger, _ := zap.NewProduction()
	base.Log = logger.Sugar()
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			os.Exit(1)
		}
	}(logger)

	if err := initial.InitAll(&base); err != nil {
		os.Exit(1)
	}
	defer func(Redis *redis.Client) {
		err := Redis.Close()
		if err != nil {
			base.Log.Errorf("error occurred: %s", err)
			os.Exit(1)
		}
	}(base.Redis)
	defer func(MySQL *sql.DB) {
		if err := MySQL.Close(); err != nil {
			base.Log.Errorf("error occurred: %s", err)
			os.Exit(1)
		}
	}(base.MySQL)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/geturl/:short", getRecord(&base))
	r.POST("/seturl", writeRecord(&base))
	if err := r.Run(os.Getenv("SERVER_ADDRESS") + ":" + os.Getenv("SERVER_PORT")); err != nil {
		base.Log.Errorf("error occurred: %s", err)
		os.Exit(1)
	}
}

func writeRecord(base *initial.General) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var urls initial.URLs
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		if err := c.Request.ParseForm(); err != nil {
			base.Log.Errorf("error occurred: %s", err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "unknown error"})
			return
		}

		for key, val := range c.Request.PostForm {
			if key == "longurl" {
				urls.Short = sup.Hash(strings.Join(val, ""), "")
				urls.Long = strings.Join(val, "")
			}
		}

		if urls.NotValid() {
			base.Log.Errorf("error occurred: error while validate URL")
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Not valid URL"})
		}

		if err := puturl.WriteToDB(base, &urls); err != nil {
			if err.Error() != "record already exist" {
				base.Log.Errorf("error occurred: %s", err)
				c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal error"})
				return
			}
		}

		base.Log.Infof("writeRecord: response sent")
		c.IndentedJSON(http.StatusOK, urls)
	}
	return fn
}

func getRecord(base *initial.General) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		shortURL := c.Param("short")

		longURL, err := geturl.GetURL(base, shortURL)
		if err != nil {
			if err.Error() == "record not exist" {
				base.Log.Errorf("error occurred: %s", err)
				c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "record not exist"})
				return
			}
			base.Log.Errorf("error occurred: %s", err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal error"})
			return
		}
		base.Log.Infof("getRecord: response sent")
		c.IndentedJSON(http.StatusOK, initial.URLs{Short: shortURL, Long: longURL})
	}
	return fn
}
