package ecgo

import (
	"net/http"
	"reflect"
	"strings"
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

//缺省路由处理，根据请求Path来分派
func (this *routerMiddleware) Handler(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if this.Path == "/favicon.ico" {
			return
		}
		path := this.Path
		routeTable := this.Config.GetSection("router")
		if p, exists := routeTable[path]; exists {
			path = p
		}
		c := "Index"
		act := "Default"
		if path != "/" {
			p := strings.SplitN(path, "/", 3) //=> /Controller/Action
			switch len(p) {
			case 3:
				c = p[1]
				act = p[2]
			case 2:
				c = p[1]
			}
		}
		this.Execute(c, act)
	})
}
