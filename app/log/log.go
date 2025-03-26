package log

import (
	"github.com/sirupsen/logrus"
	"github.com/v03413/bepusdt/app/conf"
	"io"
	"os"
)

var logger *logrus.Logger

func Init() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		ForceQuote:      true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	logger.SetLevel(logrus.InfoLevel)

	output, err := os.OpenFile(conf.GetOutputLog(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {

		panic(err)
	}

	logger.SetOutput(output)
}

func Debug(args ...interface{}) {

	logger.Debugln(args...)
}

func Info(args ...interface{}) {

	logger.Infoln(args...)
}

func Error(args ...interface{}) {

	logger.Errorln(args...)
}

func Warn(args ...interface{}) {

	logger.Warnln(args...)
}

func GetWriter() *io.PipeWriter {

	return logger.Writer()
}
