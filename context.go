package ecgo

import (
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
func (this *Context) Execute(control, act string) {
	this.Logger.Debug("excute: control=%s, action=%s", control, act)
	if controller, exists := container.controllers[control]; exists {
		c := reflect.ValueOf(controller)
		c.Elem().FieldByName("Context").Set(reflect.ValueOf(this))

		// 注入service
		elem := reflect.ValueOf(controller).Elem()
		req := reflect.ValueOf(this.Request)
		for i := 0; i < elem.NumField(); i++ {
			serviceName := elem.Type().Field(i).Name
			if service, ok := container.services[serviceName]; ok {
				s := reflect.ValueOf(service)
				s.Elem().FieldByName("Request").Set(req)
				Logger.Debug("inject service %s to %s", serviceName, control)

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
		this.Logger.Debug("controller(%s) not found", control)
		this.NotFound()
	}
}
