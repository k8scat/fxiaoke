package fxiaoke

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tidwall/gjson"
)

const (
	ObjTypePackage = "package" // 预设对象
	ObjTypeCustom  = "custom"  // 自定义对象

	FieldAPINameName             = "name"
	FieldAPINameOwner            = "owner"
	FieldAPINameCreateTime       = "create_time"
	FieldAPINameCreatedBy        = "created_by"
	FieldAPINameLastModifiedTime = "last_modified_time"
	FieldAPINameLastModifiedBy   = "last_modified_by"
	FieldAPINameRecordType       = "record_type" // 业务类型
	FieldAPINameLifeStatus       = "life_status" // 生命状态

	FilterOperatorEQ         = "EQ"
	FilterOperatorLT         = "LT"
	FilterOperatorLTE        = "LTE"
	FilterOperatorLike       = "LIKE"
	FilterOperatorIs         = "IS"
	FilterOperatorIn         = "IN"
	FilterOperatorBetween    = "BETWEEN"
	FilterOperatorStartWith  = "STARTWITH"
	FilterOperatorContains   = "CONTAINS"
	FilterOperatorGT         = "GT"
	FilterOperatorGTE        = "GTE"
	FilterOperatorNotEqual   = "N" // Not equal
	FilterOperatorNotLike    = "NLIKE"
	FilterOperatorIsNot      = "ISN"
	FilterOperatorNotIn      = "NIN"
	FilterOperatorNotBetween = "NBETWEEN"
	FilterOperatorEndWith    = "ENDWITH"
)

type Object struct {
	APIName     string                 `json:"api_name"`
	DisplayName string                 `json:"display_name"`
	Fields      map[string]interface{} `json:"fields"`
}

type SearchQueryInfo struct {
	Limit           int            `json:"limit"`
	Offset          int            `json:"offset"`
	Filters         []*QueryFilter `json:"filters"`
	FieldProjection []string       `json:"fieldProjection"`
	Orders          []*QueryOrder  `json:"orders"`
}

type QueryOrder struct {
	FieldName string `json:"fieldName"`
	ASC       bool   `json:"isAsc"`
}

type QueryFilter struct {
	Operator    string        `json:"operator"`
	FieldName   string        `json:"field_name"`
	FieldValues []interface{} `json:"field_values"`
}

type QueryResult struct {
	Total    int               `json:"total"`
	Offset   int               `json:"offset"`
	Limit    int               `json:"limit"`
	DataList []json.RawMessage `json:"dataList"`
}

type ChangeOwnerData struct {
	Owner []string `json:"ownerId"`
	ID    string   `json:"objectDataId"`
}

func (c *Client) ListObjs(objType, objApiName string, searchQueryInfo *SearchQueryInfo) ([]json.RawMessage, error) {
	var endpoint string
	switch objType {
	case ObjTypePackage:
		endpoint = "/cgi/crm/v2/data/query"
	case ObjTypeCustom:
		endpoint = "/cgi/crm/custom/v2/data/query"
	default:
		return nil, fmt.Errorf("obj type not support, objType=%s", objType)
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"dataObjectApiName": objApiName,
			"search_query_info": searchQueryInfo,
		},
	}
	content, err := c.Post(endpoint, data, true)
	if err != nil {
		return nil, err
	}
	result := new(QueryResult)
	err = json.Unmarshal([]byte(gjson.Get(content, "data").String()), result)
	if err != nil {
		return nil, err
	}
	return result.DataList, err
}

func (c *Client) ListAllObjs(objType, objApiName string, searchQueryInfo *SearchQueryInfo) (objs []json.RawMessage, err error) {
	var endpoint string
	switch objType {
	case ObjTypePackage:
		endpoint = "/cgi/crm/v2/data/query"
	case ObjTypeCustom:
		endpoint = "/cgi/crm/custom/v2/data/query"
	default:
		err = fmt.Errorf("obj type not support, objType=%s", objType)
		return
	}

	searchQueryInfo.Offset = 0
	searchQueryInfo.Limit = 100
	searchQueryInfo.Orders = append(searchQueryInfo.Orders, &QueryOrder{
		FieldName: FieldAPINameCreateTime,
		ASC:       true,
	})
	objs = make([]json.RawMessage, 0)
	for {
		data := map[string]interface{}{
			"data": map[string]interface{}{
				"dataObjectApiName": objApiName,
				"search_query_info": searchQueryInfo,
			},
		}
		var content string
		if content, err = c.Post(endpoint, data, true); err != nil {
			return
		}
		result := new(QueryResult)
		if err = json.Unmarshal([]byte(gjson.Get(content, "data").String()), result); err != nil {
			return
		}
		objs = append(objs, result.DataList...)
		if len(objs) == result.Total {
			break
		}
		searchQueryInfo.Offset += searchQueryInfo.Limit
	}
	return
}

func (c *Client) GetObjByID(objType, objApiName, id string) (obj []byte, err error) {
	var endpoint string
	switch objType {
	case ObjTypePackage:
		endpoint = "/cgi/crm/v2/data/get"
	case ObjTypeCustom:
		endpoint = "/cgi/crm/custom/v2/data/get"
	default:
		err = fmt.Errorf("obj type not support, objType=%s", objType)
		return
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"dataObjectApiName": objApiName,
			"objectDataId":      id,
		},
	}
	var content string
	content, err = c.Post(endpoint, data, true)
	if err != nil {
		return
	}
	obj = []byte(gjson.Get(content, "data").String())
	return
}

func (c *Client) UpdateObj(objType string, obj map[string]interface{}) error {
	if obj == nil || obj["dataObjectApiName"] == "" || obj["_id"] == "" {
		return fmt.Errorf("obj not valid, obj=%v", obj)
	}

	var endpoint string
	switch objType {
	case ObjTypePackage:
		endpoint = "/cgi/crm/v2/data/update"
	case ObjTypeCustom:
		endpoint = "/cgi/crm/custom/v2/data/update"
	default:
		return fmt.Errorf("obj type not support, objType=%s", objType)
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"object_data": obj,
		},
	}
	_, err := c.Post(endpoint, data, true)
	return err
}

func (c *Client) ChangeOwner(objType, objAPIName string, data []*ChangeOwnerData) error {
	if len(data) == 0 {
		return nil
	}
	if objAPIName == "" {
		return errors.New("objAPIName cannot be empty")
	}

	var endpoint string
	switch objType {
	case ObjTypePackage:
		endpoint = "/cgi/crm/v2/data/changeOwner"
	case ObjTypeCustom:
		endpoint = "/cgi/crm/custom/v2/data/changeOwner"
	default:
		return fmt.Errorf("obj type not support: %s", objType)
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"Data":              data,
			"dataObjectApiName": objAPIName,
		},
	}
	_, err := c.Post(endpoint, payload, true)
	return err
}

// 只能删除已作废的对象
// 该方法不支持 客户对象 中的 删除公海对象接口：https://open.fxiaoke.com/wiki.html#artiId=1258
func (c *Client) DeleteObjs(objType, objAPIName string, idList []string) error {
	var endpoint string
	switch objType {
	case ObjTypePackage:
		endpoint = "/cgi/crm/v2/data/delete"
	case ObjTypeCustom:
		endpoint = "/cgi/crm/custom/v2/data/delete"
	default:
		return fmt.Errorf("obj type not support: %s", objType)
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"idList":            idList,
			"dataObjectApiName": objAPIName,
		},
	}
	_, err := c.Post(endpoint, payload, true)
	return err
}

// 作废对象
func (c *Client) InvalidObj(objType, objAPIName, id string) error {
	var endpoint string
	switch objType {
	case ObjTypePackage:
		endpoint = "/cgi/crm/v2/data/invalid"
	case ObjTypeCustom:
		endpoint = "/cgi/crm/custom/v2/data/invalid"
	default:
		return fmt.Errorf("obj type not support: %s", objType)
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"object_data_id":    id,
			"dataObjectApiName": objAPIName,
		},
	}
	_, err := c.Post(endpoint, payload, true)
	return err
}

func (c *Client) CreateObj(objType string, obj interface{}) (string, error) {
	var endpoint string
	switch objType {
	case ObjTypeCustom:
		endpoint = "/cgi/crm/custom/v2/data/create"
	case ObjTypePackage:
		endpoint = "/cgi/crm/v2/data/create"
	default:
		return "", fmt.Errorf("obj type not support: %s", objType)
	}
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"object_data": obj,
		},
	}
	raw, err := c.Post(endpoint, data, true)
	if err != nil {
		return "", err
	}
	id := gjson.Get(raw, "dataId").String()
	return id, err
}
