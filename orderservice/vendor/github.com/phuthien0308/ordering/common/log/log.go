package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
)

type TagName string

const (
	AppName      TagName = "app_name"
	BuildVersion TagName = "build_version"
	OsVersion    TagName = "os_version"
	UserAgent    TagName = "user_agent"
	RequestId    TagName = "request_id"
	RequestBody  TagName = "request_body"
)

type Tag struct {
	Key   string
	Value any
}

func AppNameTag(vl string) Tag {
	return Tag{
		Key:   string(AppName),
		Value: slog.StringValue(vl),
	}
}
func AppBuildversionTag(version string) Tag {
	return Tag{
		Key:   string(BuildVersion),
		Value: slog.StringValue(version),
	}
}
func OsVersionTag(osVersion string) Tag {
	return Tag{
		Key:   string(OsVersion),
		Value: slog.StringValue(osVersion),
	}
}
func NewTag(key string, value any) Tag {
	return Tag{
		Key:   key,
		Value: value,
	}
}

func UserContextTag(userContext string) Tag {
	return Tag{
		Key:   string(UserAgent),
		Value: slog.StringValue(userContext),
	}
}

func ErrorTag(err error) []Tag {
	return []Tag{Tag{
		Key:   "error",
		Value: slog.AnyValue(err),
	}, Tag{
		Key:   "stack",
		Value: slog.StringValue(string(debug.Stack())),
	}}
}

type Logger interface {
	Debug(ctx context.Context, msg string, tags ...Tag)
	Info(ctx context.Context, msg string, tags ...Tag)
	Warn(ctx context.Context, msg string, tags ...Tag)
	Error(ctx context.Context, msg string, err error, tags ...Tag)
}

func NewLogger(logLevel LogLevel, writer io.Writer, defaultTags ...Tag) Logger {

	if writer == nil {
		writer = os.Stdout
	}

	logger := slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: toSlogLevel(logLevel)}))

	var attrs []any
	for _, tag := range defaultTags {
		attrs = append(attrs, slog.Attr{Key: string(tag.Key), Value: slog.AnyValue(tag.Value)})
	}
	logger = logger.With(attrs...)
	return &slogImpl{
		log: logger,
	}
}

type slogImpl struct {
	log *slog.Logger
}

func (s *slogImpl) Debug(ctx context.Context, msg string, tags ...Tag) {
	s.write(ctx, slog.LevelDebug, msg, tags...)
}
func (s *slogImpl) Info(ctx context.Context, msg string, tags ...Tag) {
	s.write(ctx, slog.LevelInfo, msg, tags...)
}
func (s *slogImpl) Warn(ctx context.Context, msg string, tags ...Tag) {
	s.write(ctx, slog.LevelWarn, msg, tags...)
}
func (s *slogImpl) Error(ctx context.Context, msg string, err error, tags ...Tag) {
	tags = append(tags, ErrorTag(err)...)
	s.write(ctx, slog.LevelError, msg, tags...)
}

func (s *slogImpl) write(ctx context.Context, level slog.Level, msg string, tags ...Tag) {
	var attrs []any
	for _, tag := range tags {
		attrs = append(attrs, slog.Attr{Key: string(tag.Key), Value: slog.AnyValue(tag.Value)})
	}
	s.log.Log(ctx, level, msg, attrs...)
}

func toSlogLevel(level LogLevel) slog.Level {
	switch level {
	case DEBUG:
		return slog.LevelDebug
	case INFO:
		return slog.LevelInfo
	case WARM:
		return slog.LevelWarn
	case ERROR:
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}

type zapLogImpl struct {
}
