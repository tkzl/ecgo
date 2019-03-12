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

// 入口中间件
type entryMiddleware struct {
	*Context
}

func (this *entryMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//设置默认content-type
		w.Header().Set("Content-Type", "text/html;charset=UTF-8")
		//绑定Context相关字段
		elem := reflect.ValueOf(this).Elem()
		elem.FieldByName("Response").Set(reflect.ValueOf(Response{w, 0, 200}))
		elem.FieldByName("Request").Set(reflect.ValueOf(newRequest(r)))
		ts := time.Now().UnixNano()
		elem.FieldByName("Id").Set(reflect.ValueOf(util.Md5(ts, 8)))
		elem.FieldByName("Stime").Set(reflect.ValueOf(ts))

		next.ServeHTTP(w, r)
	})
}

// 内置默认路由中间件(最后一个中间件)
type routerMiddleware struct {
	*Context
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

// 访问日志
type AccessLogMiddleware struct {
	*Context
}

func (this *AccessLogMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts := time.Now().UnixNano()
		next.ServeHTTP(w, r)

		if this.Config.Get("logger.access") != "none" {
			elapse, _ := strconv.ParseFloat(strconv.FormatInt(time.Now().UnixNano()-ts, 10), 64)
			format := this.Config.Get("logger.access_format", "method query post status length execute_time ua")
			fields := strings.Split(format, " ")
			var logs []string
			for _, field := range fields {
				switch field {
				case "method":
					logs = append(logs, r.Method)
				case "path":
					logs = append(logs, r.URL.Path)
				case "status":
					logs = append(logs, strconv.Itoa(this.Response.Code))
				case "query":
					logs = append(logs, r.URL.RequestURI())
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
					logs = append(logs, r.UserAgent())
				case "ip":
					logs = append(logs, r.RemoteAddr)
				case "referer":
					logs = append(logs, r.Referer())
				default:
					logs = append(logs, "-")
				}
			}
			msg := strings.Join(logs, " ")
			this.Logger.Access(msg)
		}
	})
}

// 状态统计
type StatsMiddleware struct {
	*Context
}
type statsCounter struct {
	ts    int64 //开始时间
	es    int64 //结束时间
	pv    int64
	bytes int64
}

func (this *statsCounter) inc(bytes int) {
	this.pv++
	this.bytes += int64(bytes)
}

// 记录一段时间的流量和访问量，1分，5分, 1小时，1天
func (this *StatsMiddleware) Handler(next http.Handler) http.Handler {
	ts := time.Now().Unix()
	total := &statsCounter{ts: ts}

	var last1m, last5m, last1h, last1d statsCounter

	current1m := &statsCounter{ts: ts}
	current5m := &statsCounter{ts: ts}
	current1h := &statsCounter{ts: ts}
	current1d := &statsCounter{ts: ts}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if this.Path == "/status" {
			html := "<table border=\"1\" cellspacing=\"0\"><tr><th>-</th><th>流量(K)</th><th>pv</th></tr>"
			html += fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%d</td></tr>", "Total", float32(total.bytes)/1024, total.pv)
			html += fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%d</td></tr>", "last 1 min", float32(last1m.bytes)/1024, last1m.pv)
			html += fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%d</td></tr>", "last 5 min", float32(last5m.bytes)/1024, last5m.pv)
			html += fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%d</td></tr>", "last 1 hour", float32(last1h.bytes)/1024, last1h.pv)
			html += fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%d</td></tr>", "yesterday", float32(last1d.bytes)/1024, last1d.pv)
			html += "</table>"
			this.Out(html)
			return
		}
		next.ServeHTTP(w, r)
		total.inc(this.Response.Length)
		current1m.inc(this.Response.Length)
		current5m.inc(this.Response.Length)
		current1h.inc(this.Response.Length)
		current1d.inc(this.Response.Length)
		tn := time.Now().Unix()
		if tn-current1m.ts >= 60 {
			last1m = statsCounter{current1m.ts, tn, current1m.pv, current1m.bytes}
			current1m = &statsCounter{ts: tn}
		}
		if tn-current5m.ts >= 5*60 {
			last5m = statsCounter{current5m.ts, tn, current5m.pv, current5m.bytes}
			current5m = &statsCounter{ts: tn}
		}
		//上一小时，整点切换
		if time.Unix(tn, 0).Format("2006010215") != time.Unix(current1d.ts, 0).Format("2006010215") {
			last1h = statsCounter{current1h.ts, tn, current1h.pv, current1h.bytes}
			current1h = &statsCounter{ts: tn}
		}
		// 上一天用自然天
		if time.Unix(tn, 0).Format("20060102") != time.Unix(current1d.ts, 0).Format("20060102") {
			last1d = statsCounter{current1d.ts, tn, current5m.pv, current5m.bytes}
			current1d = &statsCounter{ts: tn}
		}
	})
}
