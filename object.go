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

	ActionQuery       = "query"
	ActionGet         = "get"
	ActionInvalid     = "invalid"
	ActionChangeOwner = "changeOwner"
	ActionUpdate      = "update"
	ActionCreate      = "create"
	ActionDelete      = "delete"

	ParamTriggerWorkFlow     = "triggerWorkFlow"     // 触发工作流
	ParamTriggerApprovalFlow = "triggerApprovalFlow" // 触发审批流
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
	OwnerID []string `json:"ownerId"`
	ObjID   string   `json:"objectDataId"`
}

func (c *Client) ListObjs(objType, objApiName string, searchQueryInfo *SearchQueryInfo, params map[string]interface{}) (objs []json.RawMessage, total int, err error) {
	var endpoint string
	endpoint, err = GetEndpoint(objType, ActionQuery)
	if err != nil {
		return
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"dataObjectApiName": objApiName,
			"search_query_info": searchQueryInfo,
		},
	}
	for k, v := range params {
		data["data"].(map[string]interface{})[k] = v
	}

	var content string
	content, err = c.Post(endpoint, data, true)
	if err != nil {
		return
	}

	result := new(QueryResult)
	err = json.Unmarshal([]byte(gjson.Get(content, "data").String()), result)
	if err != nil {
		return
	}
	objs = result.DataList
	total = result.Total
	return
}

func (c *Client) ListAllObjs(objType, objApiName string, searchQueryInfo *SearchQueryInfo) (allObjs []json.RawMessage, err error) {
	searchQueryInfo.Offset = 0
	searchQueryInfo.Limit = 100
	searchQueryInfo.Orders = append(searchQueryInfo.Orders, &QueryOrder{
		FieldName: FieldAPINameCreateTime,
		ASC:       true,
	})
	allObjs = make([]json.RawMessage, 0)
	for {
		objs := make([]json.RawMessage, 0)
		var total int
		objs, total, err = c.ListObjs(objType, objApiName, searchQueryInfo, nil)
		if err != nil {
			return
		}
		allObjs = append(allObjs, objs...)
		if len(allObjs) >= total {
			break
		}
		searchQueryInfo.Offset += searchQueryInfo.Limit
	}
	return
}

func (c *Client) GetObjByID(objType, objApiName, id string) (obj []byte, err error) {
	var endpoint string
	endpoint, err = GetEndpoint(objType, ActionGet)
	if err != nil {
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

func (c *Client) UpdateObj(objType string, obj map[string]interface{}, params map[string]interface{}) error {
	if obj == nil || obj["dataObjectApiName"] == "" || obj["_id"] == "" {
		return fmt.Errorf("obj not valid, obj=%v", obj)
	}

	endpoint, err := GetEndpoint(objType, ActionUpdate)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"object_data": obj,
		},
	}
	for k, v := range params {
		data[k] = v
	}
	_, err = c.Post(endpoint, data, true)
	return err
}

func (c *Client) ChangeOwner(objType, objAPIName string, data []*ChangeOwnerData) error {
	if len(data) == 0 {
		return nil
	}
	if objAPIName == "" {
		return errors.New("objAPIName cannot be empty")
	}

	endpoint, err := GetEndpoint(objType, ActionChangeOwner)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"Data":              data,
			"dataObjectApiName": objAPIName,
		},
	}
	_, err = c.Post(endpoint, payload, true)
	return err
}

// 只能删除已作废的对象
// 该方法不支持 客户对象 中的 删除公海对象接口：https://open.fxiaoke.com/wiki.html#artiId=1258
func (c *Client) DeleteObjs(objType, objAPIName string, idList []string) error {
	endpoint, err := GetEndpoint(objType, ActionDelete)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"idList":            idList,
			"dataObjectApiName": objAPIName,
		},
	}
	_, err = c.Post(endpoint, payload, true)
	return err
}

// 作废对象
func (c *Client) InvalidObj(objType, objAPIName, id string) error {
	endpoint, err := GetEndpoint(objType, ActionInvalid)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"object_data_id":    id,
			"dataObjectApiName": objAPIName,
		},
	}
	_, err = c.Post(endpoint, payload, true)
	return err
}

func (c *Client) CreateObj(objType string, obj interface{}, params map[string]interface{}) (string, error) {
	endpoint, err := GetEndpoint(objType, ActionCreate)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"object_data": obj,
		},
	}
	for k, v := range params {
		data["data"].(map[string]interface{})[k] = v
	}
	raw, err := c.Post(endpoint, data, true)
	if err != nil {
		return "", err
	}
	id := gjson.Get(raw, "dataId").String()
	return id, err
}

func (c *Client) DescribeObj(objAPIName string, includeDetail bool) (string, error) {
	endpoint := "/cgi/crm/v2/object/describe"
	data := map[string]interface{}{
		"apiName":       objAPIName,
		"includeDetail": includeDetail,
	}
	raw, err := c.Post(endpoint, data, true)
	if err != nil {
		return "", err
	}
	return gjson.Get(raw, "data").String(), err
}
