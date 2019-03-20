//ecgoAPI的工具包
//包括：
//  ini配置读取
//  日志处理
//
package util

import (
	"crypto/md5"
	"fmt"
	"strconv"
)

//生成一个md5字符串,源可以是string,[]byte,或int,int64,不指定长度返回32字符
func Md5(in interface{}, length ...int) string {
	var s []byte
	switch v := in.(type) {
	case string:
		s = []byte(v)
	case []byte:
		s = v
	case int:
		s = []byte(strconv.Itoa(v))
	case int64:
		s = strconv.AppendInt([]byte{}, v, 10)
	default:
		return ""
	}

	str := fmt.Sprintf("%x", md5.Sum(s))
	l := 32
	if len(length) == 1 {
		l = length[0]
	}
	if l > len(str) {
		l = len(str)
	}
	return str[0:l]
}

//slice去重
func SliceStrUniq(slc []string) []string {
	result := []string{} // 存放结果
	for _, v := range slc {
		flag := true
		for _, vv := range result {
			if v == vv {
				flag = false // 存在重复元素，标识为false
				break
			}
		}
		if flag {
			result = append(result, v)
		}
	}
	return result
}
