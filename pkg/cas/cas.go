package cas

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var (
	tokenNotExist = errors.New("token不存在")
)

type httpResponse struct {
	Code    uint        `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type CAS struct {
	addr  string
	appId int
	token string
}

func New(addr string, appId int, token string) *CAS {
	return &CAS{
		addr:  addr,
		appId: appId,
		token: token,
	}
}

type User struct {
	ID        uint   `json:"id"`
	Phone     string `json:"phone"`
	RealName  string `json:"real_name"`
	Job       string `json:"job"`
	JobNumber string `json:"job_number"`
	HiredDate string `json:"hired_date"`
	WorkPlace uint   `json:"work_place"`
	Status    uint   `json:"status"`
	WorkType  uint   `json:"work_type"`
	IsAdmin   uint   `json:"is_admin"`
	AddTime   string `json:"add_time"`
}

func (c *CAS) CheckToken(ctx *gin.Context) (*User, error) {
	if c.token == "" {
		return nil, tokenNotExist
	}
	url := c.addr + "/service/user-info.html"
	var err error
	var response *http.Response
	var headToken string
	var pToken string
	if len(c.token) > 13 {
		headToken = c.token
	} else {
		pToken = c.token
	}
	cookieVal, _ := ctx.Cookie("admin_auth")
	payload := strings.NewReader("{\"token\":\"" + pToken + "\",\"cToken\":\"" + cookieVal + "\"}")
	request, _ := http.NewRequest("POST", url, payload)
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("token", headToken)
	response, err = http.DefaultClient.Do(request)
	refreshToken := response.Header.Get("Automatic-Refresh-Token")
	if refreshToken != "" {
		ctx.Header("Access-Control-Expose-Headers", "Automatic-Refresh-Token")
		ctx.Header("Automatic-Refresh-Token", refreshToken)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var hr httpResponse
	err = json.NewDecoder(strings.NewReader(string(body))).Decode(&hr)
	if err != nil {
		return nil, err
	}
	if hr.Code != 0 {
		return nil, errors.New("请求失败")
	}
	data, _ := json.Marshal(hr.Data)
	user := &User{}
	_ = json.Unmarshal(data, user)
	return user, nil
}

func (c *CAS) Logout() {
	if c.token == "" {
		return
	}
	url := c.addr + "/auth/logout.html"
	request, _ := http.NewRequest("POST", url, strings.NewReader(""))
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("token", c.token)
	response, _ := http.DefaultClient.Do(request)
	defer func() {
		_ = response.Body.Close()
	}()
}

type UserList struct {
	Total uint   `json:"total"`
	List  []User `json:"list"`
}

func (c *CAS) AppUsers() (*UserList, error) {
	if c.token == "" {
		return nil, tokenNotExist
	}
	url := c.addr + "/service/app-users.html"
	payload := strings.NewReader("{\"app_id\":\"" + strconv.Itoa(c.appId) + "\",\"page\":1,\"page_size\":100}")
	request, _ := http.NewRequest("POST", url, payload)
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("token", c.token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var hr httpResponse
	err = json.NewDecoder(strings.NewReader(string(body))).Decode(&hr)
	if err != nil {
		return nil, err
	}
	if hr.Code != 0 {
		return nil, errors.New("请求失败")
	}
	data, _ := json.Marshal(hr.Data)
	userList := &UserList{}
	_ = json.Unmarshal(data, userList)
	return userList, nil
}
