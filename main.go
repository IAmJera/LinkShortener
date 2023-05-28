// main starts the web server and handles the POST/GET methods
package main

import (
	"LinkShortener/general"
	"LinkShortener/geturl"
	"LinkShortener/initial"
	"LinkShortener/puturl"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strings"
)

func main() {
	base := initial.General{}

	if err := initial.InitAll(&base); err != nil {
		log.Panicf("InitAll: %s", err)
	}
	defer general.CloseFile(base.Redis)
	defer general.CloseFile(base.MySQL)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/geturl/:short", getRecord(&base))
	r.POST("/seturl", writeRecord(&base))

	if err := r.Run(":8080"); err != nil {
		log.Panicf("gin:Run: %s", err)
	}
}

func writeRecord(base *initial.General) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var urls initial.URLs
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		if err := c.Request.ParseForm(); err != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}

		for key, val := range c.Request.PostForm {
			if key == "longurl" {
				urls.Short = general.Hash(strings.Join(val, ""), "")
				urls.Long = strings.Join(val, "")
			}
		}
		if urls.NotValid() {
			log.Printf("writeRecoed:NotValid: %s", urls.Long)
			c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "Not valid URL"})
			return
		}

		if err := puturl.WriteToDB(base, &urls); err != nil {
			if err.Error() != "record already exist" {
				c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err})
				return
			}
		}
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
				c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "record not exist"})
				return
			}
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
		c.IndentedJSON(http.StatusOK, initial.URLs{Short: shortURL, Long: longURL})
	}
	return fn
}
