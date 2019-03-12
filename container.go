package ecgo

import (
	"reflect"
)

// 将controller加入容器中
func (this *Container) addController(c ...Controller) {
	this.controllers = make(map[string]Controller)
	for _, v := range c {
		this.controllers[reflect.TypeOf(v).Elem().Name()] = v
	}
}

// 将service加入容器中
func (this *Container) addService(s ...Servicer) {
	this.services = make(map[string]Servicer)
	service := reflect.ValueOf(&Service{Logger: Logger, Config: Config})
	for _, v := range s {
		reflect.ValueOf(v).Elem().FieldByName("Service").Set(service)
		this.services[reflect.TypeOf(v).Elem().Name()] = v
	}

	//再次扫描容器中所有services，为每个service绑定其它依赖
	for _, v := range this.services {
		elem := reflect.ValueOf(v).Elem()
		for i := 0; i < elem.NumField(); i++ {
			name := elem.Type().Field(i).Name
			//model
			if objType := elem.Type().Field(i).Type.String(); objType == "*ecgo.Model" {
				//TODO: Table Name与Model Name的对应转换
				elem.FieldByName(name).Set(reflect.ValueOf(this.getModel(name)))
			}
			//其它service
			if service, ok := container.services[name]; ok {
				elem.FieldByName(name).Set(reflect.ValueOf(service))
			}
		}
	}
}

// 从容器中获取model对象，如果未有先创建
func (this *Container) getModel(name string) *Model {
	if nil == this.models {
		this.models = make(map[string]*Model)
	}
	model, exists := this.models[name]
	if !exists {
		model = newModel(name)
		if err := model.LastError(); err != nil {
			panic(err)
		}
		this.models[name] = model
	}
	return model
}
