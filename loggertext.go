// Package log2fuse a Traefik HTTP logger plugin.
package log2fuse

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
)

// TextualHTTPLogger a textual logger implementation.
type TextualHTTPLogger struct {
	logger *log.Logger
	writer LogWriter
}

func (thl *TextualHTTPLogger) print(record *LogRecord) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s %s %s: %d %s %s\n",
		record.RemoteAddr, record.Method, record.URL,
		record.StatusCode, http.StatusText(record.StatusCode), record.Proto,
	))

	if len(record.RequestHeaders) > 0 {
		builder.WriteString("\nRequest Headers:\n")
		writeHeaders(&builder, record.RequestHeaders)
	}

	if record.RequestBody.Len() > 0 {
		requestBodyText, _ := record.RequestBodyDecoder.decode(record.RequestBody)
		builder.WriteString("\nRequest Body:\n")
		builder.WriteString(requestBodyText)
		builder.WriteString("\n")
	}

	if len(record.ResponseHeaders) > 0 {
		builder.WriteString("\nResponse Headers:\n")
		writeHeaders(&builder, record.ResponseHeaders)
	}

	builder.WriteString(fmt.Sprintf("\nResponse Content Length: %d\n", record.ResponseContentLength))
	builder.WriteString(fmt.Sprintf("\nDuration: %.3f ms\n", record.DurationMs))

	if record.ResponseBody.Len() > 0 {
		responseBodyText, _ := record.ResponseBodyDecoder.decode(record.ResponseBody)
		builder.WriteString("\nResponse Body:\n")
		builder.WriteString(responseBodyText)
		builder.WriteString("\n")
	}

	builder.WriteString("\n")

	err := thl.writer.Write(builder.String())
	if err != nil {
		thl.logger.Println("Failed to write:", err)
		return
	}
}

func writeHeaders(builder *strings.Builder, header http.Header) {
	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("%s: %s\n", key, strings.Join(header[key], ",")))
	}
}
