package web

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gox/frm/log"
	"github.com/gox/frm/utils"
	"github.com/redis/go-redis/v9"
)

var (
	ErrUserID     = errors.New("user_id is invalid")
	ErrCheckCode  = errors.New("check_code is invalid")
	ErrData       = errors.New("data is invalid")
	ErrUuid       = errors.New("uuid is invalid")
	ErrIdempotent = errors.New("idempotent is invalid")
)

type UserSession struct {
	UserID     int64  `json:"user_id"`
	Info       any    `json:"info,omitempty"`
	Uuid       string `json:"uuid,omitempty"`
	Token      string `json:"token,omitempty"`
	Idempotent int64  `json:"idempotent,omitempty"`
}

func (this_ *UserSession) String() string {
	return utils.ToJson(this_)
}

func GetUserSessionFromRedis(rc *redis.Client, userID any) (*UserSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	v, err := rc.Get(ctx, fmt.Sprintf("user_session:%v", userID)).Result()
	if err != nil {
		return nil, err
	}

	sess := &UserSession{}
	err = json.Unmarshal([]byte(v), sess)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

func SetUserSessionToRedis(rc *redis.Client, sess *UserSession, ttl ...time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	expire := time.Duration(0)
	if len(ttl) > 0 && ttl[0] > 0 {
		expire = ttl[0]
	}

	return rc.Set(ctx, fmt.Sprintf("user_session:%v", sess.UserID), sess.String(), expire).Err()
}

type BasicRequest struct {
	UserID     int64  `json:"user_id,omitempty"`
	Idempotent int64  `json:"idempotent,omitempty"`
	Data       string `json:"data,omitempty"`
	AesKey     []byte `json:"-"`
}

func MakeBasicRequest(c *gin.Context, salt string, req ...any) (*BasicRequest, error) {
	bReq := &BasicRequest{}
	err := c.BindJSON(bReq)
	if err != nil {
		return nil, err
	}

	if bReq.UserID < 0 {
		return nil, ErrUserID
	}

	if bReq.Idempotent <= 1000000 {
		return nil, ErrIdempotent
	}

	if len(bReq.Data) == 0 {
		return nil, ErrData
	}

	uid := c.GetHeader("X-UUID")
	if len(uid) != 36 {
		log.Error("X-UUID is invalid")
		return nil, ErrUuid
	}

	xcode := c.GetHeader("X-Code")
	if len(xcode) != 32 {
		log.Error("X-Code is invalid")
		return nil, ErrCheckCode
	}

	raw := fmt.Sprintf("user_id=%d&idempotent=%d&data=%d&salt=%s&uuid=%s", bReq.UserID, bReq.Idempotent, len(bReq.Data), salt, uid)
	checkCode := utils.MD5Hex(raw)
	if checkCode != xcode {
		log.Error("check_code not matched: %s <> %s", checkCode, xcode)
		return nil, ErrCheckCode
	}

	pos := bReq.Idempotent % 10
	bReq.AesKey = []byte(uid)[pos : pos+16]

	if len(req) > 0 {
		raw, err := base64.StdEncoding.DecodeString(bReq.Data)
		if err != nil {
			log.Error("base64.StdEncoding.DecodeString failed: %v", err)
			return nil, ErrData
		}

		data, err := utils.AesGcmDecrypt(raw, bReq.AesKey)
		if err != nil {
			log.Error("AesGcmDecrypt failed: %v", err)
			return nil, ErrData
		}

		err = json.Unmarshal(data, req[0])
		if err != nil {
			return nil, err
		}
	}

	return bReq, nil
}

func PostJSON(url string, req, rsp any) error {
	data, _ := json.Marshal(req)
	ret, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer ret.Body.Close()

	body, err := io.ReadAll(ret.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, rsp)
}

func Post(url string, req string, rsp any) error {
	log.Debug(url)
	log.Debug(req)
	ret, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(req))
	if err != nil {
		return err
	}
	defer ret.Body.Close()

	body, err := io.ReadAll(ret.Body)
	if err != nil {
		return err
	}

	log.Debug(string(body))
	return json.Unmarshal(body, rsp)
}

func PostRaw(url string, req string) (string, error) {
	ret, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(req))
	if err != nil {
		return "", err
	}
	defer ret.Body.Close()

	body, err := io.ReadAll(ret.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func Get(url string, rsp any) error {
	ret, err := http.Get(url)
	if err != nil {
		return err
	}
	defer ret.Body.Close()

	body, err := io.ReadAll(ret.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, rsp)
}
