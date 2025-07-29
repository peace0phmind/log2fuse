// Package log2fuse a Traefik HTTP logger plugin.
package log2fuse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// JSONHTTPLogger a JSON logger implementation.
type JSONHTTPLogger struct {
	clock         LoggerClock
	uuidGenerator UUIDGenerator
	logger        *log.Logger
	writer        LogWriter
}

func (jhl *JSONHTTPLogger) print(record *LogRecord) {
	requestBodyText, _ := record.RequestBodyDecoder.decode(record.RequestBody)
	responseBodyText, _ := record.ResponseBodyDecoder.decode(record.ResponseBody)
	logData := struct {
		Level                 string              `json:"log.level,omitempty"`
		Time                  string              `json:"@timestamp"`
		Message               string              `json:"message,omitempty"`
		System                string              `json:"systemName,omitempty"`
		RemoteAddr            string              `json:"remoteAddress,omitempty"`
		Method                string              `json:"method"`
		URL                   string              `json:"path"`
		Status                int                 `json:"status"`
		StatusText            string              `json:"statusText"`
		Proto                 string              `json:"proto"`
		DurationMs            float64             `json:"durationMs"`
		RequestHeaders        map[string][]string `json:"requestHeaders,omitempty"`
		RequestBody           string              `json:"requestBody,omitempty"`
		ResponseHeaders       map[string][]string `json:"responseHeaders,omitempty"`
		ResponseContentLength int                 `json:"responseContentLength"`
		ResponseBody          string              `json:"responseBody,omitempty"`
		EcsVersion            string              `json:"ecs.version,omitempty"`
		LogID                 string              `json:"logId,omitempty"`
	}{
		Level:                 "info",
		Time:                  jhl.clock.Now().UTC().Format("2006-01-02T15:04:05.999Z07:00"),
		Message:               fmt.Sprintf("%s %s %s %d", record.Method, record.URL, record.Proto, record.StatusCode),
		System:                record.System,
		RemoteAddr:            record.RemoteAddr,
		Method:                record.Method,
		URL:                   record.URL,
		Status:                record.StatusCode,
		StatusText:            http.StatusText(record.StatusCode),
		Proto:                 record.Proto,
		DurationMs:            record.DurationMs,
		RequestHeaders:        record.RequestHeaders,
		RequestBody:           requestBodyText,
		ResponseHeaders:       record.ResponseHeaders,
		ResponseContentLength: record.ResponseContentLength,
		ResponseBody:          responseBodyText,
		EcsVersion:            "1.6.0",
		LogID:                 jhl.uuidGenerator.Generate(),
	}

	logBytes, err := json.Marshal(logData)
	if err != nil {
		jhl.logger.Println("Failed to marshal json log data")
		return
	}

	var builder strings.Builder
	builder.Write(logBytes)
	builder.WriteString("\n")

	err = jhl.writer.Write(builder.String())
	if err != nil {
		jhl.logger.Println("Failed to write:", err)
		return
	}
}
