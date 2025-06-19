package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gox/frm/log"
	"github.com/gox/frm/nw"
	"github.com/gox/frm/utils"
)

const (
	N     = 10000
	NCONN = 10
	// HOST  = "18.166.30.234:9090"
	HOST = "127.0.0.1:9090"
)

var (
	ssize = int64(0)
	rsize = int64(0)
	ntime = int64(0)
)

func testClient(wg *sync.WaitGroup) {
	defer wg.Done()

	c, err := nw.NewTcpClient(HOST, 0, 0x12345678, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	for i := 0; i < N; i++ {
		str := fmt.Sprintf("Hello world: %v", i)
		wdata := *utils.Str2Bytes(str)

		wlen, err := c.Write(wdata)
		if err != nil {
			log.Error(err)
			break
		}

		atomic.AddInt64(&ssize, int64(wlen))

		rdata, err := c.Read()
		if err != nil {
			log.Error(err)
			break
		}

		atomic.AddInt64(&rsize, int64(len(rdata)))
		atomic.AddInt64(&ntime, 1)
	}
}

func main() {
	tnow := time.Now()

	wg := sync.WaitGroup{}

	wg.Add(NCONN)
	for i := 0; i < NCONN; i++ {
		go testClient(&wg)
	}

	wg.Wait()

	spend := time.Since(tnow).Seconds()

	log.Info("请求总数: %d, 发送数据: %d Bytes, 接收数据: %d Bytes, 耗时: %v",
		ntime, ssize, rsize, spend)
	log.Info("QPS: %.2f, 吞吐量: %.2f Bytes/s", float64(N)/spend, float64(ssize)/spend)
}
