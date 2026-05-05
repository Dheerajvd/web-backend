package logging

import "go.uber.org/zap"

type Logger = zap.Logger

func New(env string) *zap.Logger {
	if env == "prod" {
		l, err := zap.NewProduction()
		if err != nil {
			panic(err)
		}
		return l
	}
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return l
}

func Err(err error) zap.Field       { return zap.Error(err) }
func Str(k, v string) zap.Field     { return zap.String(k, v) }
func Int(k string, v int) zap.Field { return zap.Int(k, v) }
