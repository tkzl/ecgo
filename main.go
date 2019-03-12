package ecgo

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/tim1020/ecgo/util"
	//"log"
)

var container *Container

// 框架的结构体对象及接口定义
type (
	// 应用全局对象
	App struct {
		Utime       int64
		middlewares []Middleware
	}
	//单例容器
	Container struct {
		controllers map[string]Controller
		services    map[string]Servicer
		models      map[string]*Model
	}
	//service
	Service struct {
		Config *util.IniConfig
		Logger *util.Logger
		*Request
	}

	// 单次会话上下文对象(应用于middleware\controller)
	Context struct {
		Config *util.IniConfig
		Logger *util.Logger
		Response
		*Request
		Id    string
		Stime int64
		data  map[string]interface{} //保存自定义数据
	}
	// 请求参数对象
	Request struct {
		*http.Request
		Length    int64               //请求大小
		UpFile    map[string][]UpFile //存放上传的文件信息
		QueryData map[string]string   //存放Get参数
		PostData  map[string]string   //存放Post/put参数
		Cookie    map[string]string   //存放cookie
		Header    map[string]string   //存放header
		Method    string              //请求的方法 GET/POST...
		Path      string              //请求的Path
	}
	// 响应对象
	Response struct {
		http.ResponseWriter
		Length int //响应大小
		Code   int //响应码
	}
	//上传文件信息结构
	UpFile struct {
		Error int    //错误码，没有错误时为0
		Name  string //上传原始的文件名
		Size  int64  //文件大小
		Type  string //文件content-type
		Temp  string //上传后保存在服务器的临时文件路径
	}
	// Request reader
	ReqReader interface {
		Get(k ...string) (string, bool)
		Gets(keys ...string) map[string]string
		Post(k ...string) (string, bool)
		Posts(keys ...string) map[string]string
		GetCookie(key string) string
		GetHeader(key string) string
	}
	// controller接口，确保传入的Context对象继承了框架的Context
	Controller interface {
		Execute(c, act string)
		Store(k string, v interface{})
		Load(k string) interface{}
		Out(content string)
		JsonOk(content interface{})
		JsonErr(code int, err string)
		NotFound()
		SetHeader(k, v string)
		SetCookie(c ...interface{})
		ReqReader
	}
	//中间件接口
	Middleware interface {
		Handler(next http.Handler) http.Handler
	}
	Servicer interface {
		ReqReader
	}
)

//框架对象
func New(c ...Controller) *App {
	Logger.Debug("App start")
	container = &Container{}
	container.addController(c...)
	e := &App{Utime: time.Now().UnixNano()}

	// 绑定中间件入口
	e.Use(&entryMiddleware{})
	return e
}

func newRequest(r *http.Request) (req *Request) {
	req = &Request{Request: r}
	req.parse()
	return
}

// ---- ecgo 框架全局方法 ----

//启动服务
func (this *App) Run() {
	this.Use(&routerMiddleware{})
	ctx := reflect.ValueOf(&Context{Config: Config, Logger: Logger})
	mux := http.Handler(nil)
	for _, v := range this.middlewares {
		elem := reflect.ValueOf(v).Elem()
		elem.FieldByName("Context").Set(ctx)
		//为middleware注入service
		for i := 0; i < elem.NumField(); i++ {
			serviceName := elem.Type().Field(i).Name
			if service, ok := container.services[serviceName]; ok {
				Logger.Debug("inject %s to %s", serviceName, elem.Type().Name())
				s := reflect.ValueOf(service)
				elem.FieldByName(serviceName).Set(s)
			}
		}
		mux = v.Handler(mux)
	}
	port := Config.Get("port", ":8081")
	Logger.Debug("listen %s", port)
	http.Handle("/", mux)
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	go http.ListenAndServe(port, nil)

	//ssl
	if sslOn := Config.Get("ssl.on", "false"); sslOn == "true" {
		sslPort := Config.Get("ssl.port", ":443")
		if !strings.HasPrefix(sslPort, ":") {
			sslPort = ":" + sslPort
		}
		ca := Config.Get("ssl.ca", "")
		caKey := Config.Get("ssl.ca_key", "")
		Logger.Debug("ssl listen %s", sslPort)
		if err := http.ListenAndServeTLS(sslPort, ca, caKey, nil); err != nil {
			panic(fmt.Sprintf("ssl error: %v", err))
		}
	}

	select {}
}

// 加载中间件(栈)
func (this *App) Use(m Middleware) {
	this.middlewares = append([]Middleware{m}, this.middlewares...)
}

// 添加service进容器
func (this *App) AddService(s ...Servicer) {
	container.addService(s...)
}

// 设置自定义LogWriter
func (this *App) SetLogWriter(kind string, w util.LogWriter) {
	Logger.SetWriter(kind, w)
}
