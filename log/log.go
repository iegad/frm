package log

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	levelDebug = iota + 1
	levelInfo
	levelWarn
	levelError
	levelFatal
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
)

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
}

func Info(args ...interface{}) {
	base(levelInfo, args...)
}

func Warn(args ...interface{}) {
	base(levelWarn, args...)
}

func Error(args ...interface{}) {
	base(levelError, args...)
}

// Fatal 致命错误, 当调用此方法后, 进程将退出.
func Fatal(args ...interface{}) {
	base(levelFatal, args...)
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

func base(lv int, args ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	n := len(args)
	content := ""

	if n == 1 {
		content = fmt.Sprintf("%v", args[0])
	} else if n > 1 {
		if v, ok := args[0].(string); ok {
			content = fmt.Sprintf(v, args[1:]...)
		} else {
			panic("args[0].(string) failed")
		}
	}

	tn := time.Now()

	if len(logPath) > 0 && lv != levelDebug {
		f := getFile(lv, tn)
		if f != nil {
			fmt.Fprintf(f, "[%s %s %s:%d] %v\n", lvmap[lv], tn.Format("2006-01-02 15:04:05.000000"), file, line, content)
		}
	}

	fmt.Printf("[%s %s %s:%d] %v\n", lvmap[lv], tn.Format("2006-01-02 15:04:05.000000"), file, line, content)
}
