package ecgo

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
