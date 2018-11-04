// Author bytejedi
// 限制单IP最大连接数的中间件；计数器协程

package ws

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

var counter = &connCounter{
	add:       make(chan *MyRequest),
	sub:       make(chan *MyRequest),
	container: make(map[string]uint64),
}

// 计数器类型
type connCounter struct {
	add       chan *MyRequest
	sub       chan *MyRequest
	container map[string]uint64
}

// 为中间件定制的Request类型
type MyRequest struct {
	r     *http.Request
	allow chan bool
}

// 实现Handler接口
type MyHandlerFunc func(http.ResponseWriter, *MyRequest) error

// 重写ServeHTTP
func (f MyHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := f(w, &MyRequest{r: r, allow: make(chan bool, 1)})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// 检查单ip总的连接数的中间件
func checkMaxConnMiddleWare(next http.Handler) http.Handler {
	return MyHandlerFunc(func(w http.ResponseWriter, myR *MyRequest) error {
		// add
		counter.add <- myR
		select {
		case allow := <-myR.allow:
			if allow {
				next.ServeHTTP(w, myR.r)
				// sub
				counter.sub <- myR
				return nil
			} else {
				return errors.New("Servers are too busy, please try again later\n")
			}
		case <-time.After(time.Second * 10):
			return errors.New("Servers are too busy, please try again later\n")
		}
	})
}

// 计数器协程
func runCounter() {
	for {
		select {
		case myR := <-counter.add: // 计数器 ++
			ip := strings.Split(myR.r.RemoteAddr, ":")[0]
			// 检查ip是否存在
			if _, ok := counter.container[ip]; !ok { // 此ip不存在
				counter.container[ip] = 0
			}
			if counter.container[ip] < WsConfig.MaxConnPerIp { // 此ip的总连接数<限制最大单ip连接数
				counter.container[ip]++
				myR.allow <- true
			} else {
				myR.allow <- false
			}
		case myR := <-counter.sub: // 计数器 --
			ip := strings.Split(myR.r.RemoteAddr, ":")[0]
			// 检查ip是否存在
			if _, ok := counter.container[ip]; ok {
				// 如果此ip的连接数为0了，则删掉此ip
				if counter.container[ip] > 0 {
					counter.container[ip]--
					if counter.container[ip] == 0 {
						delete(counter.container, ip)
					}
				} else { // 保险起见
					delete(counter.container, ip)
				}
			}
		}
	}
}
