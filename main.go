package ecgo

import (
	"database/sql"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/tim1020/ecgo/util"
	//"log"
)

// 框架的结构体对象及接口定义
type (
	// 应用全局对象
	App struct {
		Utime  int64
		Config *util.IniConfig
		Logger *util.Logger
	}
	// 全局对象容器
	container struct {
		controllers       map[string]IController // 所有的controller
		services          map[string]IService    // 所有的service
		middlewares       []IMiddleware          // 用到的中间件
		controllerDepends map[string][]string    // controller依赖的service
		middlewareDepends map[string][]string    // 中间件依赖的service
		sevicesDepends    map[string]serviceNode //service所依赖的service和model
	}
	// 控制器对象
	Controller struct {
		*Context
	}
	// 单次会话上下文对象(应用于middleware\controller)
	Context struct {
		*App
		*Dao
		Services map[string]Service
		Response
		*Request
		Id    string
		Stime int64
		data  map[string]interface{} //保存自定义数据
	}
	// 请求参数对象
	Request struct {
		*http.Request
		Length    int64             //请求大小
		QueryData map[string]string //存放Get参数
		PostData  map[string]string //存放Post/put参数
		Cookie    map[string]string //存放cookie
		Header    map[string]string //存放header
		Method    string            //请求的方法 GET/POST...
		Path      string            //请求的Path
	}
	// 响应对象
	Response struct {
		http.ResponseWriter
		Length int //响应大小
		Code   int //响应码
	}
	// controller接口，确保传入的Controller对象继承了框架的Controller
	IController interface {
		Store(k string, v interface{})
		Load(k string) interface{}
		Out(content string)
		JsonOk(content interface{})
		JsonErr(code int, err string)
		NotFound()
		SetHeader(k, v string)
		SetCookie(c ...interface{})
		//ReqRead
		Get(k ...string) (string, bool)
		Gets(keys ...string) map[string]string
		Post(k ...string) (string, bool)
		Posts(keys ...string) map[string]string
		GetCookie(key string) string
		GetHeader(key string) string
	}
	Dao struct {
		DB  *sql.DB
		tx  *sql.Tx
		err error
		//sql语句，执行时间
	}
)

var (
	sBase = reflect.TypeOf((*IService)(nil)).Elem()
	c     *container
)

//框架对象
func New() *App {
	Logger.Debug("App start")
	c = newContainer()
	app := &App{Utime: time.Now().UnixNano(), Config: Config, Logger: Logger}
	app.Use(&entryMiddleware{})
	return app
}

func newRequest(r *http.Request) (req *Request) {
	req = &Request{Request: r}
	req.parse()
	return
}

func newContainer() *container {
	controllers := make(map[string]IController)    // 所有的controller
	services := make(map[string]IService)          // 所有的service
	middlewares := []IMiddleware{}                 // 用到的中间件
	controllerDepends := make(map[string][]string) // controller依赖的service
	middlewareDepends := make(map[string][]string) // 中间件依赖的service
	sevicesDepends := make(map[string]serviceNode) // service所依赖的service和model
	return &container{controllers, services, middlewares, controllerDepends, middlewareDepends, sevicesDepends}
}

// ---- ecgo 框架全局方法 ----

//启动服务
func (this *App) Run() {
	this.Use(&routerMiddleware{})
	//先绑定App到所有中间件

	mux := &middlewareMux{App: this}
	mux.Handler = http.Handler(nil)

	port := this.Config.Get("port", ":8081")
	http.Handle("/", mux)
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	this.Logger.Debug("listen %s", port)
	go http.ListenAndServe(port, nil)

	//ssl
	if sslOn := this.Config.Get("ssl.on", "false"); sslOn == "true" {
		sslPort := this.Config.Get("ssl.port", ":443")
		if !strings.HasPrefix(sslPort, ":") {
			sslPort = ":" + sslPort
		}
		ca := this.Config.Get("ssl.ca", "")
		caKey := this.Config.Get("ssl.ca_key", "")
		this.Logger.Debug("listen(ssl) %s", sslPort)
		if err := http.ListenAndServeTLS(sslPort, ca, caKey, nil); err != nil {
			panic(fmt.Sprintf("ssl error: %v", err))
		}
	}

	select {}
}

// 添加中间件
func (this *App) Use(middlewares ...IMiddleware) {
	for _, m := range middlewares {
		elem := reflect.ValueOf(m).Elem()
		mName := elem.Type().Name()
		mServices := []string{}
		//获取所依赖的service
		for i := 0; i < elem.NumField(); i++ {
			field := elem.Type().Field(i)
			fName := field.Name
			if fName != "Middleware" && field.Type.Implements(sBase) {
				if _, ok := c.services[fName]; ok {
					mServices = append(mServices, fName)
				} else {
					panic(fmt.Sprintf("service %s not found", fName))
				}
			}
		}
		c.middlewares = append(c.middlewares, m)
		c.middlewareDepends[mName] = mServices
		this.Logger.Debug("Use middleware %s, depends: %v", mName, mServices)
	}
}

// 设置自定义LogWriter
func (this *App) SetLogWriter(kind string, w util.LogWriter) {
	this.Logger.SetWriter(kind, w)
}
