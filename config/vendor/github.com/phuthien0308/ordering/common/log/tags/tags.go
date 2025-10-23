package tags

import (
	"log/slog"
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
