package main

import (
	"flag"
	"io"
	"net/http"
	"time"

	"github.com/gox/frm/log"
)

var URL string

func test() {
	c := http.Client{
		Timeout: time.Second * 15,
	}

	rsp, err := c.Post(URL, "application/json", nil)
	if err != nil {
		log.Error("http post_form failed: %v", err)
		return
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		log.Error("http post failed code: %v => %v", rsp.StatusCode, rsp.Status)
		return
	}

	body, _ := io.ReadAll(rsp.Body)
	log.Debug("response: %v", string(body))
}

func main() {
	flag.StringVar(&URL, "h", "", "host usage")
	flag.Parse()

	log.Debug(URL)

	for {
		test()
		time.Sleep(3 * time.Second)
	}
}
