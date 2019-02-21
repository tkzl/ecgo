//日志相关操作
package util

//TODO: access_log，buffer bugfix

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var kinds = []string{"trace", "debug", "warn", "event", "error"}

type LogMsg struct {
	Time    int64
	Content string
}
type LogWriter interface {
	Write(kind string, msg []LogMsg)
}

type Logger struct {
	writer   map[string]LogWriter
	buffc    map[string]chan string
	buffsize int
}

//生成Logger对象
func NewLogger(writer map[string]LogWriter, size ...int) *Logger {
	defaultWriter := &ConsoleWriter{}
	if writer == nil {
		writer = make(map[string]LogWriter)
	}
	for _, k := range kinds {
		if _, ok := writer[k]; !ok {
			writer[k] = defaultWriter
		}
	}
	buffc := make(map[string]chan string)
	buffsize := 100
	if len(size) > 0 {
		buffsize = size[0]
	}
	return &Logger{writer, buffc, buffsize}
}

//设置日志类型对应的writer
func (this *Logger) SetWriter(kind string, w LogWriter) {
	this.writer[kind] = w
}

//设置buffsize
func (this *Logger) SetBufferSize(size int) {
	this.buffsize = size
}

//记录trace日志
func (this *Logger) Trace(format string, vals ...interface{}) {
	this.write("trace", format, vals...)
}

//记录debug日志
func (this *Logger) Debug(format string, vals ...interface{}) {
	this.write("debug", format, vals...)
}

//记录warnning日志
func (this *Logger) Warn(format string, vals ...interface{}) {
	this.write("warn", format, vals...)
}

//记录event日志
func (this *Logger) Event(format string, vals ...interface{}) {
	this.write("event", format, vals...)
}

//记录error日志
func (this *Logger) Error(format string, vals ...interface{}) {
	this.write("error", format, vals...)
}

//记录指定日志
func (this *Logger) write(kind string, format string, vals ...interface{}) {
	if this.writer[kind] == nil {
		return
	}
	if _, ok := this.buffc[kind]; !ok {
		this.buffc[kind] = make(chan string, this.buffsize)
		go func(this *Logger, kind string) {
			msgs := []LogMsg{}
			for {
				select {
				case msg := <-this.buffc[kind]:
					msgs = append(msgs, LogMsg{Time: time.Now().Unix(), Content: msg})
				case <-time.After(time.Second):
					if len(msgs) > 0 { //指定时间没有新内容，发送已有内容
						this.writer[kind].Write(kind, msgs)
						msgs = msgs[0:0]
					}
				}

				if len(msgs) > this.buffsize {
					this.writer[kind].Write(kind, msgs)
					msgs = msgs[0:0]
				}
			}
		}(this, kind)
	}
	this.buffc[kind] <- fmt.Sprintf(format+"\n", vals...)
}

// >######## 内置LogWriter

//输出到console
type ConsoleWriter struct {
}

func (*ConsoleWriter) Write(kind string, msg []LogMsg) {
	for _, v := range msg {
		fmt.Printf("%s[%5s]: %s", time.Unix(v.Time, 0).Format("2006/01/02 15:04:05 "), kind, v.Content)
	}
}

//输出到文件
type FileWriter struct {
	Path string
	init bool
}

func (this *FileWriter) Write(kind string, msg []LogMsg) {
	//初次调用，判断path
	if !this.init {
		path := filepath.Dir(this.Path)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				os.MkdirAll(path, os.ModePerm)
			} else {
				log.Printf("[error]: %v", err)
				return
			}
		}
		this.init = true
	}
	dst := this.Path + "/" + kind
	//判断是否需要rotate
	stat, err := os.Stat(dst)
	if err == nil {
		//TODO: Size判断
		modify := stat.ModTime().Format("20060102") //最后修改日期
		today := time.Now().Format("20060102")
		if today != modify {
			os.Rename(dst, dst+"_"+modify)
		}
	}
	//写入
	fd, err := os.OpenFile(dst, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		log.Printf("[error]: %v", err)
		return
	}
	defer fd.Close()
	wFile := bufio.NewWriter(fd)
	for _, v := range msg {
		wFile.WriteString(fmt.Sprintf("%s %s", time.Unix(v.Time, 0).Format("2006/01/02 15:04:05 "), v.Content))
	}
	wFile.Flush()
}
