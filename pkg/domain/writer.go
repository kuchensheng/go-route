package domain

import (
	"io"
	"os"
	"path/filepath"
)

type syslogEvent struct {
	level string
	msg   string
}
type syslogWriter struct {
	events []syslogEvent
}

func (w *syslogWriter) Write(p []byte) (int, error) {
	return 0, nil
}
func (w *syslogWriter) Trace(m string) error {
	writerTrace.Write([]byte(m))
	return nil
}
func (w *syslogWriter) Debug(m string) error {
	writerDebug.Write([]byte(m))
	return nil
}
func (w *syslogWriter) Info(m string) error {
	writerInfo.Write([]byte(m))
	return nil
}
func (w *syslogWriter) Warning(m string) error {
	writerWarn.Write([]byte(m))
	return nil
}
func (w *syslogWriter) Err(m string) error {
	writerError.Write([]byte(m))
	return nil
}
func (w *syslogWriter) Emerg(m string) error {
	writerOther.Write([]byte(m))
	w.events = append(w.events, syslogEvent{"Emerg", m})
	return nil
}
func (w *syslogWriter) Crit(m string) error {
	w.events = append(w.events, syslogEvent{"Crit", m})
	return nil
}

var writerTrace io.Writer
var writerInfo io.Writer
var writerDebug io.Writer
var writerWarn io.Writer
var writerError io.Writer
var writerOther io.Writer

func InitWriter() {
	// 创建日志目录
	logDir := filepath.Join(".", "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		_ = os.Mkdir(logDir, os.ModePerm)
	}
	// 创建日志文件
	writerInfo = getWriter(logDir, "app-info")
	writerDebug = getWriter(logDir, "app-debug")
	writerWarn = getWriter(logDir, "app-warn")
	writerError = getWriter(logDir, "app-error")
	writerOther = getWriter(logDir, "app-other")
	writerTrace = getWriter(logDir, "app-trace")
}
