package main

import (
	"fmt"
	"net"

	"github.com/gox/frm/log"
)

func main() {
	c, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		log.Error(err)
		return
	}
	defer c.Close()

	buf := make([]byte, 1400)
	for i := 0; i < 10000; i++ {
		_, err = c.Write([]byte(fmt.Sprintf("Hello world: %d", i)))
		if err != nil {
			log.Error("write failed: %v", err)
			break
		}

		n, err := c.Read(buf)
		if err != nil {
			log.Error("read failed: %v", err)
			break
		}

		log.Debug(string(buf[:n]))
	}
}
