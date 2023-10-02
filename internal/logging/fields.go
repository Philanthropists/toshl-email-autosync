package logging

import (
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
)

func Time[S ~string](s S, t time.Time) Field {
	return zap.Time(string(s), t)
}

func Duration[S ~string](s S, t time.Duration) Field {
	return zap.Duration(string(s), t)
}

func Any[S ~string](s S, v any) Field {
	return zap.Any(string(s), v)
}

func Int[S ~string, T constraints.Signed](s S, v T) Field {
	return zap.Int64(string(s), int64(v))
}

func Uint[S ~string, T constraints.Unsigned](s S, v T) Field {
	return zap.Uint64(string(s), uint64(v))
}

func Float[S ~string, T constraints.Float](s S, v T) Field {
	return zap.Float64(string(s), float64(v))
}

func Error(err error) Field {
	return zap.Error(err)
}

func String[U, V ~string](s U, v V) Field {
	return zap.String(string(s), string(v))
}
