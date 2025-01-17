package developer

import (
	"encoding/json"
	"gofly/app/model"
	"gofly/global"
	"gofly/route/middleware"
	"gofly/utils"
	"gofly/utils/results"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/gohouse/gorose/v2"
)

// 用于自动注册路由
type Devapi struct {
}

func init() {
	fpath := Devapi{}
	utils.Register(&fpath, reflect.TypeOf(fpath).PkgPath())
}

// 获取部门列表
func (api *Devapi) Get_list(c *gin.Context) {
	cid := c.DefaultQuery("cid", "0")
	title := c.DefaultQuery("title", "")
	url := c.DefaultQuery("url", "")
	page := c.DefaultQuery("page", "1")
	_pageSize := c.DefaultQuery("pageSize", "10")
	pageNo, _ := strconv.Atoi(page)
	pageSize, _ := strconv.Atoi(_pageSize)
	MDB := model.DB().Table("common_apitext")
	CDB := model.DB().Table("common_apitext")
	if cid != "0" {
		MDB.Where("cid", cid)
		CDB.Where("cid", cid)
	}
	if title != "" {
		MDB.Where("title", "like", "%"+title+"%")
		CDB.Where("title", "like", "%"+title+"%")
	}
	if url != "" {
		MDB.Where("url", "like", "%"+url+"%")
		CDB.Where("url", "like", "%"+url+"%")
	}
	list, err := MDB.Limit(pageSize).Page(pageNo).Order("id desc").Get()
	if err != nil {
		results.Failed(c, err.Error(), nil)
	} else {
		for _, val := range list {
			groupdata, _ := model.DB().Table("common_apitext_group").Where("id", val["cid"]).Fields("name,type_id").First()
			val["groupname"] = groupdata["name"]
			val["type_id"] = groupdata["type_id"]
		}
		var totalCount int64
		totalCount, _ = CDB.Count()
		results.Success(c, "获取数据列表", map[string]interface{}{
			"page":     pageNo,
			"pageSize": pageSize,
			"total":    totalCount,
			"items":    list}, nil)
	}
}

// 获取分组
func (api *Devapi) Get_group(c *gin.Context) {
	list, _ := model.DB().Table("common_apitext_group").Fields("id,pid,name").Order("id asc").Get()
	list = utils.GetMenuChildrenArray(list, 0, "pid")
	if list == nil {
		list = make([]gorose.Data, 0)
	}
	results.Success(c, "获取分组列表", list, nil)
}

// 保存
func (api *Devapi) Save(c *gin.Context) {
	//获取post传过来的data
	body, _ := ioutil.ReadAll(c.Request.Body)
	var parameter map[string]interface{}
	_ = json.Unmarshal(body, &parameter)
	//当前用户
	var f_id float64 = 0
	if parameter["id"] != nil {
		f_id = parameter["id"].(float64)
	}
	parameter["createtime"] = time.Now().Unix()
	if f_id == 0 {
		delete(parameter, "id")
		addId, err := model.DB().Table("common_apitext").Data(parameter).InsertGetId()
		if err != nil {
			results.Failed(c, "添加失败", err)
		} else {
			results.Success(c, "添加成功！", addId, nil)
		}
	} else {
		delete(parameter, "groupname")
		delete(parameter, "type_id")
		res, err := model.DB().Table("common_apitext").
			Data(parameter).
			Where("id", f_id).
			Update()
		if err != nil {
			results.Failed(c, "更新失败", err)
		} else {
			results.Success(c, "更新成功！", res, nil)
		}
	}
}

// 更新状态
func (api *Devapi) UpStatus(c *gin.Context) {
	//获取post传过来的data
	body, _ := ioutil.ReadAll(c.Request.Body)
	var parameter map[string]interface{}
	_ = json.Unmarshal(body, &parameter)
	res2, err := model.DB().Table("common_apitext").Where("id", parameter["id"]).Data(map[string]interface{}{"status": parameter["status"]}).Update()
	if err != nil {
		results.Failed(c, "更新失败！", err)
	} else {
		msg := "更新成功！"
		if res2 == 0 {
			msg = "暂无数据更新"
		}
		results.Success(c, msg, res2, nil)
	}
}

// 删除
func (api *Devapi) Del(c *gin.Context) {
	//获取post传过来的data
	body, _ := ioutil.ReadAll(c.Request.Body)
	var parameter map[string]interface{}
	_ = json.Unmarshal(body, &parameter)
	ids := parameter["ids"]
	res2, err := model.DB().Table("common_apitext").WhereIn("id", ids.([]interface{})).Delete()
	if err != nil {
		results.Failed(c, "删除失败", err)
	} else {
		results.Success(c, "删除成功！", res2, nil)
	}
}

// 获取数据库字段
func (api *Devapi) Get_DBField(c *gin.Context) {
	tablename := c.DefaultQuery("tablename", "")
	if tablename == "" {
		results.Failed(c, "请传数据表名称", nil)
	} else {
		tablename_arr := strings.Split(tablename, ",")
		dbname := global.App.Config.DBconf.Name
		var dielddata_list []map[string]interface{}
		for _, Val := range tablename_arr {
			dielddata, _ := model.DB().Query("select COLUMN_NAME,COLUMN_COMMENT,COLUMN_TYPE,DATA_TYPE,CHARACTER_MAXIMUM_LENGTH,IS_NULLABLE,COLUMN_DEFAULT,NUMERIC_PRECISION from information_schema.columns where TABLE_SCHEMA='" + dbname + "' AND TABLE_NAME='" + Val + "'")
			for _, data := range dielddata {
				data["tablename"] = Val
				dielddata_list = append(dielddata_list, data)
			}
		}
		results.Success(c, "获取数据库字段", dielddata_list, tablename)
	}
}

// 获取token-测试
func (api *Devapi) Get_token(c *gin.Context) {
	//当前用户
	getuser, _ := c.Get("user")
	user := getuser.(*middleware.UserClaims)
	userdata, _ := model.DB().Table("business_wxsys_user").Where("businessID", user.BusinessID).Fields("id,accountID,businessID,name,wxapp_openid").First()
	if userdata == nil {
		results.Failed(c, "账号不存在", nil)
	} else {
		token := middleware.GenerateToken(&middleware.UserClaims{
			ID:             userdata["id"].(int64),
			Accountid:      userdata["accountID"].(int64),
			BusinessID:     userdata["businessID"].(int64),
			Openid:         userdata["wxapp_openid"].(string),
			StandardClaims: jwt.StandardClaims{},
		})
		results.Success(c, "获取测试Token", token, nil)
	}
}
