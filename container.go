package ecgo

import (
	"reflect"
)

// 将service加入容器中
func (this *App) AddService(s ...IService) {
	if len(s) < 1 {
		panic("Service empty")
	}
	this.Logger.Debug("AddService")
	//遍历加入容器，同时记录每个service所依赖的service和model
	for _, v := range s {
		elem := reflect.ValueOf(v).Elem()
		sName := elem.Type().Name()
		dServices := []string{}
		dModels := []string{}
		for i := 0; i < elem.NumField(); i++ {
			field := elem.Type().Field(i)
			fName := field.Name
			if _, ok := c.services[fName]; fName != "Service" && (ok || field.Type.Implements(sBase)) {
				dServices = append(dServices, fName)
			} else {
				if objType := field.Type.String(); objType == "*ecgo.Model" {
					dModels = append(dModels, fName)
				}
			}
		}

		c.services[sName] = v
		c.sevicesDepends[sName] = serviceNode{dServices, dModels}
		this.Logger.Debug(" > service %s depends: services=%v, models=%v", sName, dServices, dModels)
	}
	this.Logger.Debug("services: %v", c.services)
}

// 将controller加入容器中
func (this *App) AddController(controller ...IController) {
	if c.services == nil {
		panic("Call AddService before AddController pls")
	}
	if len(controller) < 1 {
		panic("Controller empty")
	}
	this.Logger.Debug("AddController")
	for _, v := range controller {
		elem := reflect.ValueOf(v).Elem()
		cName := reflect.TypeOf(v).Elem().Name()

		//记录每个controller依赖的service
		cServices := []string{}
		for i := 0; i < elem.NumField(); i++ {
			field := elem.Type().Field(i)
			fName := field.Name
			if field.Type.Implements(sBase) {
				if _, ok := c.services[fName]; ok {
					cServices = append(cServices, fName)
				} else {
					//panic(fmt.Sprintf("service %s not found", fName))
				}
			}
		}
		c.controllers[cName] = v
		c.controllerDepends[cName] = cServices
		this.Logger.Debug(" > controller %s depends: service=%v", cName, cServices)
	}
	this.Logger.Debug("controllers: %v", c.controllers)
}

// 加载中间件
func (this *App) LoadMiddleware() []IMiddleware {
	tmpM := []IMiddleware{}
	this.Logger.Debug("LoadMiddleware")
	//TODO: 计算所有用到的中间件所依赖的service
	middlewares := []string{}
	rvM := reflect.ValueOf(&Middleware{&Context{App: this}})
	for _, v := range c.middlewares {
		tmpM = append([]IMiddleware{v}, tmpM...) //倒排
		elem := reflect.ValueOf(v).Elem()
		mName := elem.Type().Name()
		elem.FieldByName("Middleware").Set(rvM)
		middlewares = append(middlewares, mName)
		this.Logger.Debug(" > middleware %s depends: services=%v", mName, c.middlewareDepends[mName])
	}
	this.Logger.Debug("middlewares: %v", middlewares)
	return tmpM
}

//从容器中克隆
func Clone(typ, name string) reflect.Value {
	var src interface{}
	var exists = false
	switch typ {
	case "controller":
		src, exists = c.controllers[name]
	case "service":
		src, exists = c.services[name]
	}
	if exists {
		tye := reflect.TypeOf(src).Elem()
		return reflect.New(tye)
	} else {
		return reflect.Value{}
	}
}
