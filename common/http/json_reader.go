package http

import (
	"encoding/json"
	"io"
)

func NewJsonReader(v any) (io.Reader, error) {
	jsonP, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	jsonReader := &jsonReader{
		data: jsonP,
	}
	return jsonReader, nil
}

type jsonReader struct {
	data []byte
	i    int64 // current index
}

func (j *jsonReader) Read(p []byte) (int, error) {
	if j.i >= int64(len(j.data)) {
		return 0, io.EOF
	}
	copiedNumber := copy(p, j.data[j.i:])
	j.i += int64(copiedNumber)
	return copiedNumber, nil
}
