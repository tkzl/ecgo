package ecgo

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

//mysql操作对象
type Model struct {
	*Dao
	Table       string        //表名
	order       string        //排序，“id desc”
	field       string        //查询的字段
	pageIndex   int           //页码(由1开始)
	pageSize    int           //每页大小
	pkName      string        //主键名称
	pkId        int64         //主键值
	where       string        //条件
	whereHolder []interface{} //where的占位值
	fields      []string      //表的字段列表
}

//TODO: 记录所有执行过的sql语句和执行时间

//创建model对象(基于mysql)

func newModel(table string) (model *Model) {
	Logger.Debug(">>>>>new Model")
	dbConf := Config.GetSection("db")
	db, err := sql.Open("mysql", dbConf["mysql_dsn"])
	if err == nil {
		db.Ping()
		model = &Model{Dao: &Dao{DB: db}, Table: table}
		model.err = model.desc()
		model.reset()
	}
	return
}

//重置model默认参数
func (this *Model) reset() {
	this.tx = nil
	this.pageIndex = 1
	this.pageSize = 30
	this.pkId = 0
	this.field = "*"
	this.where = ""
	this.order = ""
}

// 设置查询的字段，field = "*", field="a,b b1, d as d1"
func (this *Model) Select(field string) *Model {
	field = strings.Trim(field, " ")
	if field != "*" {
		fields := strings.Split(field, ",")
		fs := []string{}
		for _, v := range fields {
			f := strings.SplitN(strings.Trim(v, " "), " ", 2)
			if !this.fieldExists(f[0]) {
				this.err = errors.New(fmt.Sprintf("field invalid: %s", f[0]))
				return nil
			}
			fStr := fmt.Sprintf("`%s`", f[0])
			if len(f) > 1 {
				fStr += " " + f[1]
			}
			fs = append(fs, fStr)
		}
		this.field = strings.Join(fs, ",")
	} else {
		this.field = "*"
	}
	return this
}

// 设置排序字段,
func (this *Model) Order(oStr ...string) *Model {
	if len(oStr) < 1 {
		this.err = errors.New("Order param not set")
		return nil
	}
	o := []string{}
	for _, v := range oStr { //oStr = ["a desc","b asc","c"]
		s := strings.Split(strings.Trim(v, " "), " ") //
		if !this.fieldExists(s[0]) {
			this.err = errors.New(fmt.Sprintf("field invalid: %s", s[0]))
			return nil
		}
		d := "asc"
		if len(s) > 1 && strings.ToLower(s[1]) == "desc" {
			d = "desc"
		}
		o = append(o, fmt.Sprintf("`%s` %s", s[0], d))
	}
	this.order = strings.Join(o, ",")
	return this
}

//设置limit
func (this *Model) Limit(page, size int) *Model {
	this.pageIndex = page
	this.pageSize = size
	return this
}

func (this *Model) Id(id int64) *Model {
	this.pkId = id
	return this
}

// 设置where条件,支持直接条件表达式或占位
func (this *Model) Where(wh string, val ...interface{}) *Model {
	//TODO:判断where中的字段是否存在

	//判断占位符数量是否与val一致
	if len(val) != strings.Count(wh, "?") {
		this.err = errors.New(fmt.Sprintf("holder not match: wh=%s,params=%v", wh, val))
		return nil
	}
	this.where = wh
	this.whereHolder = val
	return this
}

//统计指定条件的记录数
func (this *Model) GetCount() (nums int) {
	sqlStr := fmt.Sprintf("select count(*) nums from `%s`", this.Table)
	where, _ := this.getWhere()
	if where != "" {
		sqlStr = sqlStr + " where " + where
	}
	rows := this.DB.QueryRow(sqlStr)
	rows.Scan(&nums)
	return
}

// 查询单条
func (this *Model) Get() (map[string]string, error) {
	this.Limit(1, 1)
	sqlStr, vals := this.buildSelect()
	result, err := this.Query(sqlStr, vals...)
	if err == nil && len(result) > 0 {
		return result[0], nil
	} else {
		return nil, err
	}
}

// 查询记录集(多条)
func (this *Model) GetAll() ([]map[string]string, error) {
	sqlStr, vals := this.buildSelect()
	return this.Query(sqlStr, vals...)
}

// 删除
func (this *Model) Delete() (int64, error) {
	where, vals := this.getWhere()
	if where == "" {
		return 0, errors.New("Can not Delete all,set where pls")
	}
	sqlStr := fmt.Sprintf("delete from `%s` where %s", this.Table, where)
	return this.Exec(sqlStr, vals...)
}

// 更新指定条件的记录
func (this *Model) Update(data map[string]string) (int64, error) {
	var fields []string
	var vals []interface{}
	for k, v := range data {
		if !this.fieldExists(k) {
			return 0, errors.New(fmt.Sprintf("field invalid: %s ", k))
		}
		fields = append(fields, k+"= ?")
		vals = append(vals, v)
	}
	where, vals1 := this.getWhere()
	if where == "" {
		return 0, errors.New("Can not Update all,set where pls")
	}
	if vals1 != nil {
		vals = append(vals, vals)
	}
	sqlStr := fmt.Sprintf("update `%s` set %s where %s", this.Table, strings.Join(fields, ","), where)
	return this.Exec(sqlStr, vals...)
}

//插入一条数据,成功时返回自增ID(若无自增字段返回0),统一用string类型插入
func (this *Model) Add(data map[string]string) (int64, error) {
	var fields []string
	var vals []interface{}
	for k, v := range data {
		if !this.fieldExists(k) {
			return 0, errors.New(fmt.Sprintf("field invalid: %s ", k))
		}
		fields = append(fields, k+"= ?")
		vals = append(vals, v)
	}
	sqlStr := fmt.Sprintf("insert into `%s` set %s", this.Table, strings.Join(fields, ","))
	return this.Exec(sqlStr, vals...)
}

// 执行update/delete/insert语句
func (this *Model) Exec(sqlStr string, vals ...interface{}) (id int64, err error) {
	if this.err != nil {
		Logger.Error("sqlError: %v", this.err)
		err = this.err
		return
	}
	sqlStr = strings.Trim(sqlStr, " ")
	Logger.Debug("sql=%s; params=%v", sqlStr, vals)
	this.reset()
	var stmt *sql.Stmt
	if this.tx != nil {
		stmt, err = this.tx.Prepare(sqlStr)
	} else {
		stmt, err = this.DB.Prepare(sqlStr)
	}
	if err == nil {
		defer stmt.Close()
		res, err1 := stmt.Exec(vals...)
		if err1 == nil {
			switch prefix := strings.ToLower(sqlStr[:6]); prefix {
			case "insert":
				id, err = res.LastInsertId()
			case "update", "delete":
				id, err = res.RowsAffected()
			default:
				err = errors.New("不支持的SQL语句,select操作请使用Query方法")
			}
		} else {
			err = err1
		}
	}
	this.err = err
	return
}

// 执行查询语句select,返回结果集
func (this *Model) Query(sqlStr string, vals ...interface{}) (result []map[string]string, err error) {
	if this.err != nil {
		Logger.Error("sqlError=%v", this.err)
		err = this.err
		return
	}
	sqlStr = strings.Trim(sqlStr, " ")
	Logger.Debug("sql=%s; params=%v", sqlStr, vals)
	this.reset()
	if !strings.HasPrefix(strings.ToLower(sqlStr), "select") && !strings.HasPrefix(strings.ToLower(sqlStr), "desc") {
		err = errors.New("不支持的SQL语句")
		return
	}

	var rows *sql.Rows
	if this.tx != nil {
		rows, err = this.tx.Query(sqlStr, vals...)
	} else {
		rows, err = this.DB.Query(sqlStr, vals...)
	}

	if err == nil { //处理结果
		defer rows.Close()
		cols, _ := rows.Columns()
		l := len(cols)
		rawResult := make([][]byte, l)

		dest := make([]interface{}, l) // A temporary interface{} slice
		for i, _ := range rawResult {
			dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
		}
		for rows.Next() {
			rowResult := make(map[string]string)
			err = rows.Scan(dest...)
			if err == nil {
				for i, raw := range rawResult {
					key := cols[i]
					if raw == nil {
						rowResult[key] = "\\N"
					} else {
						rowResult[key] = string(raw)
					}
				}
				result = append(result, rowResult)
			}
		}
	}
	this.err = err
	return
}

//开启事务
func (this *Model) TransStart() error {
	tx, err := this.DB.Begin()
	if err != nil {
		return err
	}
	this.err = nil
	this.tx = tx
	return nil
}

//提交事务，如果事务中有错误发生，则自动回滚，并返回错误
func (this *Model) TransCommit() (err error) {
	if this.err != nil {
		err = this.err
		this.tx.Rollback()
	} else {
		err = this.tx.Commit()
	}
	this.tx = nil
	return
}

//手工回滚事务
func (this *Model) TransRollback() (err error) {
	err = this.tx.Rollback()
	this.tx = nil
	return
}

//返回最后发生的错误
func (this *Model) LastError() error {
	return this.err
}

// 释放连接
func (this *Model) Close() {
	if this.DB != nil {
		this.DB.Close()
	}
}

//组装select语句
func (this *Model) buildSelect() (string, []interface{}) {
	sqlStr := fmt.Sprintf("select %s from `%s`", this.field, this.Table)
	where, vals := this.getWhere()
	if where != "" {
		sqlStr = sqlStr + " where " + where
	}
	if this.order != "" {
		sqlStr += " order by " + this.order
	}
	sqlStr += fmt.Sprintf(" limit %d, %d", (this.pageIndex-1)*this.pageSize, this.pageSize)
	return sqlStr, vals
}

// 获取where字串以及对应的占位值
func (this *Model) getWhere() (string, []interface{}) {
	if this.pkId != 0 { //有设置主键值，直接使用主键
		return fmt.Sprintf("`%s` = %d", this.pkName, this.pkId), nil
	} else {
		return this.where, this.whereHolder
	}
}

//获取表结构
func (this *Model) desc() error {
	re, err := this.Query(fmt.Sprintf("DESC `%s`", this.Table))
	if err == nil {
		for _, row := range re {
			this.fields = append(this.fields, row["Field"])
			if row["Key"] == "PRI" {
				this.pkName = row["Field"]
			}
		}
	}
	return err
}

//字段是否存在
func (this *Model) fieldExists(f string) bool {
	exists := false
	for _, v := range this.fields {
		if v == f {
			exists = true
			break
		}
	}
	return exists
}
