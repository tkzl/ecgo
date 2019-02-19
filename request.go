package ecgo

//Request对象方法

import (
	"net/http"
	"net/url"
	"strings"
)

//同名参数分隔符
var pSep = "`"

/**
 * 对http请求进行格式化处理, 并将结果存入App的成员变量 Get/Post/Cookie/Header/UpFile
 */
func (this *Request) parse() {
	multipart := false
	this.Header = getHeader(this.Request)
	ct := this.Header["Content-Type"]
	if strings.HasPrefix(ct, "multipart/form-data") {
		multipart = true
		this.ParseMultipartForm(10 << 20)
	} else if strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
		this.ParseForm()
	}
	this.QueryData = getGet(this.Request)
	this.PostData = getPost(this.Request, multipart)
	this.Header = getHeader(this.Request)
	this.Cookie = getCookie(this.Request)

	//TODO: 文件上传处理

	// if multipart {
	//
	// }

	this.Path = this.URL.Path
	return
}

//获取header
func getHeader(req *http.Request) (header map[string]string) {
	header = make(map[string]string)
	for k, v := range req.Header {
		header[k] = strings.Join(v, ";")
	}
	return
}

//获取cookie
func getCookie(req *http.Request) (cookie map[string]string) {
	cookie = make(map[string]string)
	for _, v := range req.Cookies() {
		cookie[v.Name] = v.Value
	}
	return
}

//获取GET参数，同名参数内容以req_sep串接
func getGet(req *http.Request) (get map[string]string) {
	get = make(map[string]string)
	q, _ := url.ParseQuery(req.URL.RawQuery)
	for k, v := range q {
		k = strings.TrimSuffix(k, "[]") //如果是xxx[]方式的key,只保留xx,所以 xx和xx[]会相互覆盖
		get[k] = strings.Join(v, pSep)
	}
	return
}

//获取post参数,m表示是否multiPart方式请求,同名参数内容以req_sep串接
func getPost(req *http.Request, multipart bool) (post map[string]string) {
	post = make(map[string]string)
	var vals url.Values
	if multipart && req.MultipartForm != nil {
		vals = req.MultipartForm.Value
	} else {
		vals = req.PostForm
	}
	for k, v := range vals {
		k = strings.TrimSuffix(k, "[]") //如果是xxx[]方式的key,只保留xx,所以 xx和xx[]会相互覆盖
		post[k] = strings.Join(v, pSep)
	}
	return
}

//TODO: XSS处理

//<<<<<----- 定义Context对象的获取请求内容的相关方法

//获取get参数，当有第二参数时，第二参数为缺省值，不指定缺省值且未设置时，ok为false
func (this *Request) Get(k ...string) (string, bool) {
	key := k[0]
	if val, exists := this.QueryData[key]; exists {
		return val, true
	}
	if len(k) > 1 {
		return k[1], true
	}
	return "", false
}

// 批量获取get参数，当参数未设置时，值为空字符串
func (this *Request) Gets(keys ...string) map[string]string {
	val := make(map[string]string)
	for _, key := range keys {
		if v, exists := this.QueryData[key]; exists {
			val[key] = v
		}
	}
	return val
}

//获取post,当有第二参数时，第二参数为缺省值，不指定缺省值且未设置时，返回nil
func (this *Request) Post(k ...string) (string, bool) {
	key := k[0]
	if val, exists := this.PostData[key]; exists {
		return val, true
	}
	if len(k) > 1 {
		return k[1], true
	}
	return "", false
}

// 批量获取post参数,当参数未设置时，值为空字符串
func (this *Request) Posts(keys ...string) map[string]string {
	val := make(map[string]string)
	for _, key := range keys {
		if v, exists := this.PostData[key]; exists {
			val[key] = v
		}
	}
	return val
}

//获取指定cookie
func (this *Request) GetCookie(key string) string {
	if val, exists := this.Cookie[key]; exists {
		return val
	}
	return ""
}

//获取指定header
func (this *Request) GetHeader(key string) string {
	if val, exists := this.Header[key]; exists {
		return val
	}
	return ""
}
