// Copyright 2019 ecgo Author. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at :  http://www.apache.org/licenses/LICENSE-2.0

// 一个golang编写的简单的API开发框架。
//
// 更多内容请参考： http://github.com/tim1020/ecgoAPI
//
// 基本使用方法：
//
//	package main
//	import (
//	    "github.com/tim1020/ecgoAPI"
//  )
//	type Context struct {
//		*ecgoAPI.Context
//	}
//	func main() {
//		e := ecgoAPI.New(&Context{}，nil)
//		e.Run()
//	}
//	//controller
//	func (this *Context) Action() {
//		//this.Out("Hello ecgo")
//	}
//
package ecgo

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tim1020/ecgo/util"
)

//框架全局常量
const (
	VER    = "1.0"
	AUTHOR = "Tim<tim8670@gmail.com>"
)

var (
	Config   *util.IniConfig
	Logger   *util.Logger
	RootPath string
)

//初始化: 确定运行路径
func init() {
	file, _ := filepath.Abs(os.Args[0])
	RootPath = filepath.Dir(file) //将执行文件所在的路径设为应用的根路径
	cf := RootPath + "/conf/application.ini"
	Config = util.NewIniConfig(cf)
	if Config == nil {
		log.Fatalln("[error]: 配置文件不存在")
	}
	//加载Logger
	Logger = util.NewLogger(nil, 100)
	ws := Config.GetSection("logger")

	for k, v := range ws {
		switch {
		case v == "none":
			Logger.SetWriter(k, nil)
		case v == "file":
			Logger.SetWriter(k, &util.FileWriter{Path: RootPath + "/logs"})
		case strings.HasPrefix(v, "file:"):
			p := strings.Split(v, ":")
			Logger.SetWriter(k, &util.FileWriter{Path: p[1]})
		default:
			Logger.SetWriter(k, &util.ConsoleWriter{})
		}
	}
}
