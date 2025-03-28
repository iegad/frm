package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gox/frm/utils"
)

const (
	levelDebug = iota + 1
	levelInfo
	levelWarn
	levelError
	levelFatal

	insertDebug = "insert_debug"
	insertInfo  = "insert_info"
	insertWarn  = "insert_warning"
	insertError = "insert_error"
	insertFatal = "insert_fatal"
)

var (
	logPath = "" // 日志路径
	// level值 映射 名称
	lvmap = map[int]string{
		levelDebug: "DEBUG",
		levelInfo:  "INFO",
		levelError: "ERROR",
		levelFatal: "FATAL",
		levelWarn:  "WARN",
	}

	lvfnmap = map[int]string{}   // 当前level值 对应的文件名
	lvfmap  = map[int]*os.File{} // 当前level值 对应的文件句柄
	fmtx    = sync.Mutex{}

	rcfg       *Config
	errUrl     = errors.New("url is invalid")
	errProject = errors.New("project is invalid")
	errMod     = errors.New("mod is invalid")
	errUser    = errors.New("user is invalid")
	errKey     = errors.New("key is invalid")
)

type Config struct {
	Url     string `json:"url"`
	Project string `json:"project"`
	Mod     string `json:"mod"`
	User    string `json:"user"`
	Key     string `json:"key"`
}

type logInfo struct {
	Project string `json:"project"`
	Mod     string `json:"mod"`
	Type    int    `json:"type"`
	Tag     string `json:"tag"`
	Content string `json:"content"`
	Time    int64  `json:"time"`
}

type insertLogReq struct {
	User string   `json:"user"`
	Key  string   `json:"key"`
	Log  *logInfo `json:"log"`
}

func (this_ *insertLogReq) Json() []byte {
	data, _ := json.Marshal(this_)
	return data
}

type insertLogRsp struct {
	Code  int64  `json:"Code"`
	Error string `json:"Error,omitempty"`
}

func LoadConfig(c *Config) error {
	if len(c.Url) == 0 {
		return errUrl
	}

	if len(c.Project) == 0 {
		return errProject
	}

	if len(c.Mod) == 0 {
		return errMod
	}

	if len(c.User) == 0 {
		return errUser
	}

	if len(c.Key) == 0 {
		return errKey
	}

	rcfg = c
	return nil
}

func SetPath(path string) {
	n := len(path)

	if n == 0 {
		return
	}

	if path[n-1:n] == "/" {
		logPath = path[:n-1]
	}
	logPath = path

	_, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(logPath, 0755)
			if err != nil {
				panic(err)
			}
		}
	}
}

func Debug(args ...interface{}) {
	base(levelDebug, args...)

	if rcfg != nil {
		remote(levelDebug, args...)
	}
}

func Info(args ...interface{}) {
	base(levelInfo, args...)

	if rcfg != nil {
		remote(levelInfo, args...)
	}
}

func Warn(args ...interface{}) {
	base(levelWarn, args...)

	if rcfg != nil {
		remote(levelWarn, args...)
	}
}

func Error(args ...interface{}) {
	base(levelError, args...)

	if rcfg != nil {
		remote(levelError, args...)
	}
}

// Fatal 致命错误, 当调用此方法后, 进程将退出.
func Fatal(args ...interface{}) {
	base(levelFatal, args...)

	if rcfg != nil {
		remote(levelFatal, args...)
	}
	os.Exit(1)
}

func getFile(lv int, tn time.Time) *os.File {
	fmtx.Lock()
	defer fmtx.Unlock()
	fname := fmt.Sprintf("%s/%s.%s", logPath, tn.Format("2006-01-02"), lvmap[lv])
	if fname != lvfnmap[lv] {
		if lvfmap[lv] != nil {
			lvfmap[lv].Sync()
			lvfmap[lv].Close()
		}
	}

	lvfnmap[lv] = fname
	var err error

	lvfmap[lv], err = os.OpenFile(fname, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return lvfmap[lv]
}

func buildContent(args ...any) string {
	if len(args) == 1 {
		return fmt.Sprintf("%v", args[0])
	}

	if v, ok := args[0].(string); ok {
		return fmt.Sprintf(v, args[1:]...)
	}

	panic("first params must be string")
}

func base(lv int, args ...any) {
	_, file, line, _ := runtime.Caller(2)
	content := buildContent(args...)

	tn := time.Now()

	if len(logPath) > 0 && lv != levelDebug {
		f := getFile(lv, tn)
		if f != nil {
			fmt.Fprintf(f, "[%s %s %s:%d] %v\n", lvmap[lv], tn.Format("2006-01-02 15:04:05.000000"), file, line, content)
		}
	}

	fmt.Printf("[%s %s %s:%d] %v\n", lvmap[lv], tn.Format("2006-01-02 15:04:05.000000"), file, line, content)
}

func remote(lv int, args ...any) {
	req := &insertLogReq{
		User: rcfg.User,
		Key:  rcfg.Key,
		Log: &logInfo{
			Project: rcfg.Project,
			Mod:     rcfg.Mod,
			Type:    lv,
			Content: buildContent(args...),
			Time:    time.Now().Unix(),
		},
	}

	route := ""
	switch lv {
	case levelDebug:
		route = insertDebug

	case levelInfo:
		route = insertInfo

	case levelWarn:
		route = insertWarn

	case levelError:
		route = insertError

	case levelFatal:
		route = insertFatal
	}

	data, err := utils.HttpPostJson(fmt.Sprintf("%v/%v", rcfg.Url, route), req.Json())
	if err != nil {
		base(levelError, err)
		return
	}

	rsp := &insertLogRsp{}
	err = json.Unmarshal(data, rsp)
	if err != nil {
		base(levelError, err)
		return
	}

	if rsp.Code != 0 {
		base(levelError, rsp.Error)
	}
}
