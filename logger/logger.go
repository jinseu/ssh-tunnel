package logger

import (
	"bufio"
	"fmt"
	"os"
	"time"
	"path/filepath"
	"bytes"
	"runtime"
)

type LEVEL byte

const (
	DEBUG LEVEL = iota
	INFO
	WARN
	ERROR
	OFF
)

const (
	DATEFORMAT             = "2006010215"
	TIMEFORMAT             = "2006-01-02 15:04:05"
	DEFAULT_FILE_SIZE      = 1600000000 //1.6G
	DEFAULT_LOG_QUEUE_SIZE = 2000
	DEFAULT_LOG_LEVEL      = DEBUG
)


// FileLogger 日志按小时拆分，有单文件大小限制
type FileLogger struct {
	fileDir     string
	fileName    string
	fileMaxSize int64

	logFile     *os.File
	logWriter   *bufio.Writer
	logSize int64
	logTime string

	logChan  chan string
	logLevel LEVEL

	flushCh     chan bool
	flushDoneCh chan bool
}

func NewLogger(fileDir, fileName string, fileMaxSize int64, logLevel LEVEL, logQueueCount int) *FileLogger {
	if fileDir == "" || fileName == "" {
		return nil
	}

	logger := &FileLogger{
		fileDir:       fileDir,
		fileName:      fileName,
		fileMaxSize:   fileMaxSize,
		logChan:       make(chan string, logQueueCount),
		logLevel:      logLevel,
		flushCh:     make(chan bool),
		flushDoneCh: make(chan bool),
	}

	logger.initLogger()
	if logger.logFile == nil {
		return nil
	}

	return logger
}

func (f *FileLogger) initLogger() {
	f.logTime = time.Now().Format(DATEFORMAT)
	logFile := joinFilePath(f.fileDir, f.fileName+"_"+f.logTime+".log")
	if !isExist(f.fileDir) {
		os.Mkdir(f.fileDir, 0755)
	}
	var err error
	f.logFile, err = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("OpenFile [%s] fail: %v\n", logFile, err)
		f.logFile = nil
		return
	}

	f.logWriter = bufio.NewWriter(f.logFile)
	f.logSize = fileSize(logFile)

	go f.doLogData()
}

func (f *FileLogger) doLogData() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("FileLogger catch panic: %v\n", err)
		}
	}()

	seqTimer := time.NewTicker(time.Second)
	for {
		select {
		case str := <-f.logChan:

			f.checkSplit()

			l := int64(len(str))
			if l + f.logSize <= f.fileMaxSize {
				f.logSize += l
				f.logWriter.WriteString(str)
			}
		case <-seqTimer.C:
			f.logWriter.Flush()

		case <-f.flushCh:
			f.flush()
		}
	}
}

func (f *FileLogger) flush() {

	f.checkSplit()
	flg := true
	for flg {
		select {
		case str := <-f.logChan:

			l := int64(len(str))
			if l + f.logSize <= f.fileMaxSize {
				f.logSize += l
				f.logWriter.WriteString(str)
			}
		default:
			flg = false
		}
	}
	f.logWriter.Flush()
	f.flushDoneCh <- true
}

func (f *FileLogger) Flush() {
	f.flushCh <- true
	<-f.flushDoneCh
}

func (f *FileLogger) checkSplit() {

	now := time.Now().Format(DATEFORMAT)
	if now == f.logTime {
		return
	}

	f.logTime = now
	logFile := joinFilePath(f.fileDir, f.fileName+"_"+f.logTime+".log")

	if f.logFile != nil {
		f.logFile.Close()
	}

	f.logFile, _ = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	f.logWriter = bufio.NewWriter(f.logFile)
	f.logSize = fileSize(logFile)
}

func (f *FileLogger) Close() error {
	close(f.logChan)
	return f.logFile.Close()
}

func (f *FileLogger) SetLogLevel(logLevel LEVEL) {
	f.logLevel = logLevel
}

func (f *FileLogger) log(logLevel LEVEL, s string) {
	if f.logLevel > logLevel {
		return
	}
	_, file, line, _ := runtime.Caller(2)

	select {
	case f.logChan <- fmt.Sprintf("%s [%s:%d]", time.Now().Format(TIMEFORMAT), shortFileName(file), line) + s:

	default:
		
	}
}

func (f *FileLogger) Debug(format string, v ...interface{}) {
	f.log(DEBUG, fmt.Sprintf("[DEBUG] " + format + "\n", v...))
}

func (f *FileLogger) Info(format string, v ...interface{}) {
	f.log(INFO, fmt.Sprintf("[INFO] " + format + "\n", v...))
}

func (f *FileLogger) Warn(format string, v ...interface{}) {
	f.log(WARN, fmt.Sprintf("[WARN] " + format + "\n", v...))
}

func (f *FileLogger) Error(format string, v ...interface{}) {
	f.log(ERROR, fmt.Sprintf("[ERROR] " + format + "\n", v...))
}

// flush日志文件
func Flush() {
	if globalLogger != nil {
		return
	}
	globalLogger.Flush()
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func joinFilePath(path, file string) string {
	return filepath.Join(path, file)
}

func fileSize(file string) int64 {
	f, e := os.Stat(file)
	if e != nil {
		return 0
	}

	return f.Size()
}

func shortFileName(file string) string {
	return filepath.Base(file)
}

func GetStackTrace(skip int, traceDepth int) string {

	var buf bytes.Buffer

	pc := make([]uintptr, traceDepth)
	n := runtime.Callers(skip+2, pc)

	rpc := make([]uintptr, n)
	for i := 0; i < n; i++ {
		rpc[n-1-i] = pc[i]
	}

	frames := runtime.CallersFrames(rpc)
	for {
		frame, more := frames.Next()
		if buf.Len() > 0 {
			buf.WriteString("->")
		}
		buf.WriteString(fmt.Sprintf("%s:%d", shortFileName(frame.File), frame.Line))
		if !more {
			break
		}
	}

	return buf.String()
}

var globalLogger *FileLogger = nil
var log2Stdout bool = false
var globalLevel LEVEL = DEBUG

// InitGlobalLogger 日志是有缓存的，1s Flush 一次
func InitGlobalLogger(fileDir, fileName string, fileMaxSize int64, logLevel LEVEL) {
	if globalLogger != nil {
		return
	}
	globalLogger = NewLogger(fileDir, fileName, fileMaxSize, logLevel, DEFAULT_LOG_QUEUE_SIZE)
	globalLevel = logLevel
}

func SetLog2Stdout(flg bool) {
	log2Stdout = flg
}

func log(logLevel LEVEL, s string) {

	_, file, line, _ := runtime.Caller(2)
	b := fmt.Sprintf("%s [%s:%d]%s", time.Now().Format(TIMEFORMAT), shortFileName(file), line, s)

	if log2Stdout || globalLogger == nil {
		fmt.Printf("%s", b)
	}

	if globalLogger == nil || globalLogger.logLevel > logLevel {
		return
	}

	select {

	case globalLogger.logChan <- b:
		
	default:
		
	}
}

func logStack(logLevel LEVEL, s string, traceDepth int) {

	b := fmt.Sprintf("%s [%s] %s", time.Now().Format(TIMEFORMAT), GetStackTrace(2, traceDepth), s)

	if log2Stdout || globalLogger == nil {
		fmt.Printf("%s", b)
	}

	if globalLogger == nil || globalLogger.logLevel > logLevel {
		return
	}

	select {

	case globalLogger.logChan <- b:

	default:

	}
}

// 设置新的全局日志等级
func SetLogLevel(logLevel LEVEL) {
	globalLevel = logLevel
	if globalLogger != nil {
		globalLogger.SetLogLevel(logLevel)
	}
}

// 使用全局日志记录trace log
func Debug(format string, v ...interface{}) {
	if globalLevel > DEBUG {
		return
	}
	if !log2Stdout && globalLogger == nil {
		return
	}
	log(DEBUG, fmt.Sprintf("[TRACE] " + format + "\n", v...))
}

// 使用全局日志记录info log
func Info(format string, v ...interface{}) {
	if globalLevel > INFO {
		return
	}
	if !log2Stdout && globalLogger == nil {
		return
	}
	log(INFO, fmt.Sprintf("[INFO] " + format + "\n", v...))
}

// 使用全局日志记录warning log
func Warn(format string, v ...interface{}) {
	if globalLevel > WARN {
		return
	}
	if !log2Stdout && globalLogger == nil {
		return
	}
	log(WARN, fmt.Sprintf("[WARN] " + format + "\n", v...))
}

// 使用全局日志记录error log
func Error(format string, v ...interface{}) {
	if globalLevel > ERROR {
		return
	}
	if !log2Stdout && globalLogger == nil {
		return
	}
	log(ERROR, fmt.Sprintf("[ERROR] " + format + "\n", v...))
}

func Fatal(format string, v ...interface{}){
	log(ERROR, fmt.Sprintf("[ERROR] " + format + "\n", v...))
	Flush()
	os.Exit(1)
}

func ErrorStack(traceDepth int, format string, v ...interface{}) {
	if globalLevel > ERROR {
		return
	}
	if !log2Stdout && globalLogger == nil {
		return
	}
	logStack(ERROR, fmt.Sprintf("[ERROR] " + format + "\n", v...), traceDepth)
}
