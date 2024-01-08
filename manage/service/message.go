package service

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"task/manage/config"
	"task/pkg/cas"
	"task/pkg/helper"
	"time"
)

var WSCManage *WSClientManage

type WSClientManage struct {
	clients map[uint]*WSClient
	mux     sync.Mutex
}

type WSClient struct {
	ID      uint
	UserID  uint
	RoleID  uint
	Conn    *websocket.Conn
	Message chan []byte
	Close   chan struct{}
}

func newWSCManage() *WSClientManage {
	return &WSClientManage{
		clients: make(map[uint]*WSClient, 10),
	}
}

func (m *WSClientManage) addClient(c *WSClient) {
	m.mux.Lock()
	m.clients[c.ID] = c
	m.mux.Unlock()
}

func (m *WSClientManage) delClient(id uint) {
	m.mux.Lock()
	delete(m.clients, id)
	m.mux.Unlock()
}

func (m *WSClientManage) pushWSMessage(roleIDS []uint, msg string) {
	m.mux.Lock()
	for _, c := range m.clients {
		if c.RoleID == 1 || helper.IsExistInUintSlice(roleIDS, c.RoleID) {
			c.Message <- []byte(msg)
		}
	}
	m.mux.Unlock()
}

func (c *WSClient) read() {
	defer close(c.Close)
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *WSClient) close() {
	_ = c.Conn.Close()
	WSCManage.delClient(c.ID)
	recycleWSClientID(c.ID)
}

func (c *WSClient) write() {
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-c.Close:
			c.close()
			return
		case msg := <-c.Message:
			err := c.Conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				return
			}
		case <-ticker.C:
			err := c.Conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				c.close()
				return
			}
		}
	}
}

var wsClientIDChan chan uint
var wsClientID uint

func getWSClientID() uint {
	if wsClientIDChan == nil {
		wsClientIDChan = make(chan uint, 100)
	}
	select {
	case wsClientID = <-wsClientIDChan:
		return wsClientID
	default:
		wsClientID = wsClientID + 1
		return wsClientID
	}
}

func recycleWSClientID(ID uint) {
	wsClientIDChan <- ID
}

func message(ctx *gin.Context) {
	token := ctx.GetHeader("Sec-WebSocket-Protocol")
	if token == "" {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	cs := cas.New(config.CasAddress(), config.CasAppId(), token)
	user, err := cs.CheckToken(ctx)
	if err != nil {
		failed(ctx, 1000, "Forbidden Access!!")
		return
	}
	roleID := rbacService.currentUserRoleId(user)
	if roleID == 0 {
		failed(ctx, 1001, "Forbidden Access!!")
		return
	}
	up := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		HandshakeTimeout: 3 * time.Second,
		Subprotocols:     []string{token},
	}
	conn, err := up.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		failed(ctx, 1002, "websocket serve exception,"+err.Error())
		return
	}
	c := &WSClient{
		ID:      getWSClientID(),
		UserID:  user.ID,
		RoleID:  roleID,
		Conn:    conn,
		Message: make(chan []byte, 1024),
		Close:   make(chan struct{}),
	}
	WSCManage.addClient(c)
	go c.read()
	go c.write()
}
