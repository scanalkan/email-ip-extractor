package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"regexp"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/gin-gonic/gin"
)

var ipAddresses []string

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			log.Fatal(err)
		}

		data := struct {
			IPAddresses []string
		}{
			IPAddresses: ipAddresses,
		}

		c.Header("Content-Type", "text/html")
		tmpl.Execute(c.Writer, data)
	})

	go func() {
		log.Fatal(r.Run(":8080"))
	}()

	checkEmails()
	select {}
}

func checkEmails() {
	//write host for your mail adress
	c, err := client.DialTLS("mail host", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Logout()

	if err := c.Login("mailadress", "password"); err != nil {
		log.Fatal(err)
	}
	//where is the mails which ones you reach
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}
	if mbox == nil {
		log.Fatal("No messages found")
	}

	since := time.Now().Add(-time.Hour)
	criteria := imap.NewSearchCriteria()
	criteria.Since = since

	msgSeqNums, err := c.Search(criteria)
	if err != nil {
		log.Fatal(err)
	}

	seqSet := new(imap.SeqSet)
	for _, seqNum := range msgSeqNums {
		seqSet.AddNum(seqNum)
	}

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, "BODY[]"}, messages)
	}()

	for msg := range messages {
		bodySection := msg.GetBody(&imap.BodySectionName{})
		if bodySection == nil {
			continue
		}

		body, err := ioutil.ReadAll(bodySection)
		if err != nil {
			log.Println("Error reading body:", err)
			continue
		}

		ipRegex := `(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`
		matches := regexp.MustCompile(ipRegex).FindAllString(string(body), -1)

		for _, match := range matches {
			if !containsIP(ipAddresses, match) {
				ipAddresses = append(ipAddresses, match)
			}
		}
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}
}

func containsIP(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
