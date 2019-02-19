package ecgo

import (
	"fmt"
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
		//TODO: 注入models,dao等
		this.services[reflect.TypeOf(v).Elem().Name()] = v
	}

	//再次扫描容器中所有services，为每个service绑定所依赖的其它service
	for _, v := range this.services {
		elem := reflect.ValueOf(v).Elem()
		for i := 0; i < elem.NumField(); i++ {
			serviceName := elem.Type().Field(i).Name
			if service, ok := container.services[serviceName]; ok {
				fmt.Printf("serviceName=%s\n", serviceName)
				s := reflect.ValueOf(service)
				elem.FieldByName(serviceName).Set(s)
			}
		}
	}
}
