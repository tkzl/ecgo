package ecgo

//定义Context对象响应输出的相关方法

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// 输出响应内容
func (this *Response) Out(content string) {
	fmt.Fprintf(this.ResponseWriter, content)
	this.Length = len(content)
}

//json格式的正常响应，{code: ,error:null, data: content}
func (this *Response) JsonOk(content interface{}) {
	this.SetHeader("Content-Type", "application/json;charset=utf-8")
	jData := make(map[string]interface{})
	jData["code"], _ = strconv.Atoi(Config.Get("err_code.ok", "0"))
	jData["error"] = nil
	jData["data"] = content

	jsonStr, err := json.Marshal(jData)
	if err != nil {

	} else {
		this.Out(string(jsonStr))
	}
}

//json格式的错误响应 {code: errcode, error: errMsg, data: interface}
func (this *Response) JsonErr(code int, err string) {
	this.SetHeader("Content-Type", "application/json;charset=utf-8")
	jData := make(map[string]interface{})
	jData["code"] = code
	jData["error"] = err
	jData["data"] = nil

	jsonStr, _ := json.Marshal(jData)
	this.Out(string(jsonStr))
}

// 访问控制器不存在时，输出404
func (this *Response) NotFound() {
	this.WriteHeader(404)
	this.Code = 404
	this.Out("<h2>404 Not Found!</h2>")
}

// 设置响应header
func (this *Response) SetHeader(k, v string) {
	this.Header().Set(k, v)
}

//设置cookie
//支持三种方式 :
// 1. SetCookie(name,val string)
// 2.SetCookie(name string,val string,expire int)
// 3.SetCookie(c *http.Cookie)
func (this *Response) SetCookie(c ...interface{}) {
	len := len(c)
	switch {
	case len == 1: //http.Cookie
		if cookie, ok := c[0].(*http.Cookie); ok {
			http.SetCookie(this.ResponseWriter, cookie)
			return
		}
	case len == 2: //name,val
		name, ok1 := c[0].(string)
		val, ok2 := c[1].(string)
		if ok1 && ok2 {
			http.SetCookie(this.ResponseWriter, &http.Cookie{Name: name, Value: val})
			return
		}
	case len == 3: //name,val,expire
		name, ok1 := c[0].(string)
		val, ok2 := c[1].(string)
		t, ok3 := c[2].(int)
		if ok1 && ok2 && ok3 {
			expires := time.Now().Add(time.Second * time.Duration(t))
			http.SetCookie(this.ResponseWriter, &http.Cookie{Name: name, Value: val, Expires: expires})
			return
		}
	}
}
