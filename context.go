package ecgo

import (
	"fmt"
	"reflect"
)

// 请求的上下文相关方法

//设置当前请求上下文的指定参数值
func (this *Context) Store(k string, v interface{}) {
	if nil == this.data {
		this.data = make(map[string]interface{})
	}
	this.data[k] = v
}

// 获取当前请求上下文的指定参数值
func (this *Context) Load(k string) interface{} {
	if v, exists := this.data[k]; exists {
		return v
	}
	return nil
}

// 执行控制器及方法
func (this *Context) Execute(c, act string) {
	if controller, exists := container.controllers[c]; exists {
		c := reflect.ValueOf(controller)
		c.Elem().FieldByName("Context").Set(reflect.ValueOf(this))
		// 注入service

		elem := reflect.ValueOf(controller).Elem()
		req := reflect.ValueOf(this.Request)
		for i := 0; i < elem.NumField(); i++ {
			serviceName := elem.Type().Field(i).Name
			fmt.Printf("sname=%s\n", serviceName)
			if service, ok := container.services[serviceName]; ok {
				s := reflect.ValueOf(service)
				s.Elem().FieldByName("Request").Set(req)
				elem.FieldByName(serviceName).Set(s)
			}
		}

		method := c.MethodByName(act)
		if method.IsValid() {
			begin := c.MethodByName("Begin")
			if begin.IsValid() {
				begin.Call(nil)
			}
			method.Call(nil)
			end := c.MethodByName("End")
			if end.IsValid() {
				end.Call(nil)
			}
		} else {
			this.Logger.Debug("action(%s) not found", act)
			this.NotFound()
		}
	} else {
		this.Logger.Debug("controller(%s) not found", c)
		this.NotFound()
	}
}
