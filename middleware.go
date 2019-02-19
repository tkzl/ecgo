package ecgo

import (
	"net/http"
	"reflect"
	"time"

	"github.com/tim1020/ecgo/util"
)

// 入口中间件
type entryMiddleware struct {
	*Context
}

// 内置默认路由中间件(最后一个中间件)
type routerMiddleware struct {
	*Context
}

func (this *entryMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//设置默认content-type
		w.Header().Set("Content-Type", "text/html;charset=UTF-8")
		//绑定Context相关字段
		elem := reflect.ValueOf(this).Elem()
		elem.FieldByName("Response").Set(reflect.ValueOf(Response{w}))
		elem.FieldByName("Request").Set(reflect.ValueOf(newRequest(r)))
		ts := time.Now().UnixNano()
		elem.FieldByName("Id").Set(reflect.ValueOf(util.Md5(ts, 8)))
		elem.FieldByName("Stime").Set(reflect.ValueOf(ts))

		next.ServeHTTP(w, r)
	})
}

func (this *routerMiddleware) Handler(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 根据请求的Method,Path来获得要执行控制器及方法
		c, _ := this.Get("c")
		act, _ := this.Get("act")
		this.Execute(c, act)
	})
}
