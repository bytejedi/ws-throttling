package ws

import (
	"net/http"
)

func ServeWs(addr string) {
	//go initData()
	// 是否允许跨域请求
	if !WsConfig.CheckOrigin {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
	}

	serveWs := http.HandlerFunc(serveWs)
	if WsConfig.MaxConnPerIp == 0 {
		http.Handle("/", serveWs)
	} else {
		go runCounter()
		http.Handle("/", checkMaxConnMiddleWare(serveWs))
	}
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		zlog.Error("ListenAndServe: ", err)
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	// 升级http协议为websocket协议
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		zlog.Error(err)
		return
	}
}
