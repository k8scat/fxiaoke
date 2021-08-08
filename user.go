package fxiaoke

import (
	"encoding/json"

	"github.com/tidwall/gjson"
)

type User struct {
	OpenUserID    string `json:"openUserId"`
	Name          string `json:"name"`
	NickName      string `json:"nickName"`
	LeaderID      string `json:"leaderId"`
	Position      string `json:"position"`
	Email         string `json:"email"`
	DepartmentIDs []int  `json:"departmentIds"`
	Mobile        string `json:"mobile"`
	CreateTime    int64  `json:"createTime"`
}

func (c *Client) ListUsersByDepartmentID(departmentID int, fetchChild bool) (users []*User, err error) {
	data := map[string]interface{}{
		"departmentId": departmentID,
		"fetchChild":   fetchChild,
	}
	var content string
	if content, err = c.Post("/cgi/user/list", data, true); err != nil {
		return
	}
	err = json.Unmarshal([]byte(gjson.Get(content, "userList").String()), &users)
	return
}

func (c *Client) GetUserByOpenID(openUserID string) (user *User, err error) {
	data := map[string]interface{}{
		"openUserId": openUserID,
	}
	var content string
	if content, err = c.Post("/cgi/user/get", data, true); err != nil {
		return
	}
	err = json.Unmarshal([]byte(content), &user)
	return
}
