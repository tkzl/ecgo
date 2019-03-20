package ecgo

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/tim1020/ecgo/util"
)

type (
	// 中间件对象
	Middleware struct {
		*Context
	}
	// 中间件接口
	IMiddleware interface {
		Execute(c, act string)
		Handler(func())
	}
	// mux
	middlewareMux struct {
		*App
		Handler http.Handler
	}
)

//设置中间件入口
func (this *middlewareMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i := 0
	var next func() // next 函数指针
	ts := time.Now().UnixNano()
	response := Response{w, 0, 200}
	ctx := &Context{App: this.App, Request: newRequest(r), Response: response, Id: util.Md5(ts, 8), Stime: ts}
	rvM := reflect.ValueOf(&Middleware{Context: ctx})
	next = func() {
		if i < len(c.middlewares) { //加入到容器中的中间件
			i++
			m := c.middlewares[i-1]
			elem := reflect.ValueOf(m)
			//复制中间件对象
			if !elem.Elem().FieldByName("singleton").IsValid() {
				tye := reflect.TypeOf(m).Elem()
				elem = reflect.New(tye)
			}
			elem.Elem().FieldByName("Middleware").Set(rvM)
			//TODO: 绑定Service
			method := elem.MethodByName("Handler")
			params := make([]reflect.Value, 1)
			params[0] = reflect.ValueOf(next)
			method.Call(params)
		} else if this.Handler != nil {
			this.Handler.ServeHTTP(response, r)
		}
	}
	next()
}

// 执行控制器方法
func (this *Middleware) Execute(control, act string) {
	this.Logger.Debug("excute: control=%s, action=%s", control, act)
	elem := Clone("controller", control)
	if elem.IsValid() {
		this.Logger.Debug("this.Context=%p", this.Context)
		ctl := &Controller{Context: this.Context}
		elem.Elem().FieldByName("Controller").Set(reflect.ValueOf(ctl))
		//TODO: 绑定service

		method := elem.MethodByName(act)
		if method.IsValid() {
			begin := elem.MethodByName("Begin")
			if begin.IsValid() {
				begin.Call(nil)
			}
			method.Call(nil)
			end := elem.MethodByName("End")
			if end.IsValid() {
				end.Call(nil)
			}
		} else {
			this.Logger.Debug("action(%s) not found", act)
			this.NotFound()
		}
	} else {
		this.Logger.Debug("controller(%s) not found", control)
		this.NotFound()
	}
}

// 入口中间件
type entryMiddleware struct {
	*Middleware
}

func (this *entryMiddleware) Handler(next func()) {
	ts := time.Now().UnixNano()

	this.Logger.Debug(">>> request start, entryMiddleware in >>>")
	this.Logger.Debug("method=%s,query=%v, postData=%v", this.Method, this.URL.RequestURI(), this.PostData)
	this.Logger.Debug("entryMiddleware ptr=%p", this)
	// 设置默认content-type
	this.SetHeader("Content-Type", "text/html;charset=UTF-8")

	next()

	if this.Config.Get("logger.access") != "none" {
		elapse, _ := strconv.ParseFloat(strconv.FormatInt(time.Now().UnixNano()-ts, 10), 64)
		format := this.Config.Get("logger.access_format", "method query post status length execute_time ua")
		fields := strings.Split(format, " ")
		var logs []string
		for _, field := range fields {
			switch field {
			case "method":
				logs = append(logs, this.Method)
			case "path":
				logs = append(logs, this.URL.Path)
			case "status":
				logs = append(logs, strconv.Itoa(this.Response.Code))
			case "query":
				logs = append(logs, this.URL.RequestURI())
			case "post":
				post := []string{}
				if this.Request.PostData != nil {
					for k, v := range this.Request.PostData {
						post = append(post, fmt.Sprintf("%s=%s", k, v))
					}
				}
				if len(post) > 0 {
					logs = append(logs, strings.Join(post, "&"))
				} else {
					logs = append(logs, "-")
				}

			case "length":
				logs = append(logs, strconv.Itoa(this.Response.Length))
			case "execute_time":
				logs = append(logs, fmt.Sprintf("%.3f", elapse/1000000))
			case "ua":
				logs = append(logs, this.UserAgent())
			case "ip":
				logs = append(logs, this.RemoteAddr)
			case "referer":
				logs = append(logs, this.Referer())
			default:
				logs = append(logs, "-")
			}
		}
		msg := strings.Join(logs, " ")
		this.Logger.Access(msg)
	}
	this.Logger.Debug(">>> request end, entryMiddleware out >>>")
}

// 内置默认路由中间件(最后一个中间件)
type routerMiddleware struct {
	*Middleware
}

//缺省路由处理，根据请求Path来分派
func (this *routerMiddleware) Handler(next func()) {
	this.Logger.Debug("routerMiddleware ptr=%p", this)
	if this.Path == "/favicon.ico" {
		return
	}
	path := this.Path
	routeTable := this.Config.GetSection("router")
	if p, exists := routeTable[path]; exists {
		path = p
	}
	control := "Index"
	act := "Default"
	if path != "/" {
		p := strings.SplitN(path, "/", 3) //=> /Controller/Action
		switch len(p) {
		case 3:
			control = p[1]
			act = p[2]
		case 2:
			control = p[1]
		}
	}
	this.Execute(control, act)
}

// 状态统计
type StatsMiddleware struct {
	*Middleware
	singleton                                                                         bool
	total, current1m, current5m, current1h, current1d, last1m, last5m, last1h, last1d *statsCounter
}
type statsCounter struct {
	ts    int64 //开始时间
	es    int64 //结束时间
	pv    int64
	bytes int64
	e404  int64 //404
}

func (this *statsCounter) inc(bytes int, code int) {
	this.pv++
	this.bytes += int64(bytes)
	if code == 404 {
		this.e404++
	}
}

// 记录一段时间的流量和访问量，1分，5分, 1小时，1天
func (this *StatsMiddleware) Handler(next func()) {
	this.Logger.Debug("StatsMiddleware ptr=%p", this)
	if this.total == nil {
		ts := time.Now().Unix()
		this.total = &statsCounter{ts: ts}
		this.current1m = &statsCounter{ts: ts}
		this.current5m = &statsCounter{ts: ts}
		this.current1h = &statsCounter{ts: ts}
		this.current1d = &statsCounter{ts: ts}
		this.last1m = &statsCounter{ts: ts}
		this.last5m = &statsCounter{ts: ts}
		this.last1h = &statsCounter{ts: ts}
		this.last1d = &statsCounter{ts: ts}
	}
	if this.Path == "/status" {
		html := "<table border=\"1\" cellspacing=\"0\"><tr><th>-</th><th>流量(K)</th><th>pv</th><th>404</th></tr>"
		html += fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%d</td><td>%d</td></tr>", "Total", float32(this.total.bytes)/1024, this.total.pv, this.total.e404)
		html += fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%d</td><td>%d</td></tr>", "last 1 min", float32(this.last1m.bytes)/1024, this.last1m.pv, this.last1m.e404)
		html += fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%d</td><td>%d</td></tr>", "last 5 min", float32(this.last5m.bytes)/1024, this.last5m.pv, this.last5m.e404)
		html += fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%d</td><td>%d</td></tr>", "last 1 hour", float32(this.last1h.bytes)/1024, this.last1h.pv, this.last1h.e404)
		html += fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%d</td><td>%d</td></tr>", "yesterday", float32(this.last1d.bytes)/1024, this.last1d.pv, this.last1d.e404)
		html += "</table>"
		this.Out(html)
		return
	}

	next()

	this.total.inc(this.Response.Length, this.Response.Code)
	this.current1m.inc(this.Response.Length, this.Response.Code)
	this.current5m.inc(this.Response.Length, this.Response.Code)
	this.current1h.inc(this.Response.Length, this.Response.Code)
	this.current1d.inc(this.Response.Length, this.Response.Code)
	tn := time.Now().Unix()
	if tn-this.current1m.ts >= 60 {
		this.last1m = &statsCounter{this.current1m.ts, tn, this.current1m.pv, this.current1m.bytes, this.current1m.e404}
		this.current1m = &statsCounter{ts: tn}
	}
	if tn-this.current5m.ts >= 5*60 {
		this.last5m = &statsCounter{this.current5m.ts, tn, this.current5m.pv, this.current5m.bytes, this.current5m.e404}
		this.current5m = &statsCounter{ts: tn}
	}
	//上一小时，整点切换
	if time.Unix(tn, 0).Format("2006010215") != time.Unix(this.current1d.ts, 0).Format("2006010215") {
		this.last1h = &statsCounter{this.current1h.ts, tn, this.current1h.pv, this.current1h.bytes, this.current1h.e404}
		this.current1h = &statsCounter{ts: tn}
	}
	// 上一天用自然天
	if time.Unix(tn, 0).Format("20060102") != time.Unix(this.current1d.ts, 0).Format("20060102") {
		this.last1d = &statsCounter{this.current1d.ts, tn, this.current5m.pv, this.current5m.bytes, this.current5m.e404}
		this.current1d = &statsCounter{ts: tn}
	}
}
