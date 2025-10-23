package log

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/phuthien0308/ordering/common/log/tags"
)

type Logger interface {
	Debug(ctx context.Context, msg string, tags ...tags.Tag)
	Info(ctx context.Context, msg string, tags ...tags.Tag)
	Warn(ctx context.Context, msg string, tags ...tags.Tag)
	Error(ctx context.Context, msg string, err error, tags ...tags.Tag)
}

type defaultLoggerSetup struct {
	level       LogLevel
	writer      io.Writer
	defaultTags []tags.Tag
}
type LogOptions func(*defaultLoggerSetup)

func WithLogLevel(loglevel LogLevel) LogOptions {
	return func(dls *defaultLoggerSetup) {
		dls.level = loglevel
	}
}

func WithOutputSource(writer io.Writer) LogOptions {
	return func(dls *defaultLoggerSetup) {
		dls.writer = writer
	}
}

func WithDefaultTags(tags []tags.Tag) LogOptions {
	return func(dls *defaultLoggerSetup) {
		dls.defaultTags = tags
	}
}

func NewDefaultLogger(opts ...LogOptions) Logger {
	optionsImp := &defaultLoggerSetup{
		level:  DEBUG,
		writer: os.Stdout,
	}
	for _, opt := range opts {
		opt(optionsImp)
	}

	logger := slog.New(slog.NewJSONHandler(optionsImp.writer, &slog.HandlerOptions{Level: toSlogLevel(optionsImp.level)}))

	var attrs []any
	for _, tag := range optionsImp.defaultTags {
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

func (s *slogImpl) Debug(ctx context.Context, msg string, tags ...tags.Tag) {
	s.write(ctx, slog.LevelDebug, msg, tags...)
}
func (s *slogImpl) Info(ctx context.Context, msg string, tags ...tags.Tag) {
	s.write(ctx, slog.LevelInfo, msg, tags...)
}
func (s *slogImpl) Warn(ctx context.Context, msg string, tags ...tags.Tag) {
	s.write(ctx, slog.LevelWarn, msg, tags...)
}
func (s *slogImpl) Error(ctx context.Context, msg string, err error, t ...tags.Tag) {
	t = append(t, tags.ErrorTag(err)...)
	s.write(ctx, slog.LevelError, msg, t...)
}

func (s *slogImpl) write(ctx context.Context, level slog.Level, msg string, tags ...tags.Tag) {
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
