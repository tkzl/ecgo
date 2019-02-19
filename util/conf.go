//ini格式配置文件的读取处理

package util

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	bComment      = []byte{';'}
	bSectionStart = []byte{'['}
	bSectionEnd   = []byte{']'}
	bEqual        = []byte{'='}
)

type IniConfig struct {
	fd io.Reader
	content map[string]interface{}
	Mtime int64
}

func NewIniConfig(file string) *IniConfig {
	fd, err := os.Open(file)
	defer fd.Close()
	if err != nil {
		return nil
	}
	stat, _ := fd.Stat()
	cfg := &IniConfig{fd, make(map[string]interface{}), stat.ModTime().Unix()}
	cfg.parse()
	return cfg
}

//读取单项配置, key = "k" 或 key="section.k",没有返回缺省值
func (this *IniConfig) Get(args ...string) string{
	defaultVal:=""
	if len(args) > 1 {
		defaultVal = args[1]
	}
	key := args[0]
	keys := strings.SplitN(key, ".",2)
	
	if(len(keys) > 1) {
		if sItem, ok := this.content[keys[0]]; ok {
			if val, ok := sItem.(map[string]string)[keys[1]]; ok {
				return val
			}
		}
	} else {
		if val, ok := this.content[key];ok {
			return val.(string)
		}
	}
	return defaultVal
}

//读取一个section
func (this *IniConfig) GetSection(key string) (map[string]string) {
	val := make(map[string]string)
	if items, ok := this.content[key]; ok {
		for k, v := range items.(map[string]string) {
			val[k] = v
		}
	}
	return val
}

//解析
func (this *IniConfig) parse() (err error) {
	buf := bufio.NewReader(this.fd)
	section := ""
	for ln := 1; ; ln++ {
		line, err := buf.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return err
			} else if len(line) == 0 {
				break
			}
		}
		line = bytes.TrimSpace(line)
		if line == nil || bytes.HasPrefix(line, bComment) { //空白行或注释
			continue
		}
		if bytes.HasPrefix(line, bSectionStart) && bytes.HasSuffix(line, bSectionEnd) {
			section = strings.ToLower(string(line[1 : len(line)-1]))
			this.content[section] = make(map[string]string)
			continue
		}
		keyValue := bytes.SplitN(line, bEqual, 2)
		if len(keyValue) != 2 {
			return  errors.New(fmt.Sprintf("配置文件读取错误: line=%d", ln))
		}
		key := string(bytes.TrimSpace(keyValue[0]))
		val := bytes.TrimSpace(keyValue[1])
		val = bytes.Trim(val, `"'`) //如果有，去掉引号
		if section == "" {
			this.content[key] = string(val)
		} else {
			this.content[section].(map[string]string)[key] = string(val)
		}
	}
	return nil
}