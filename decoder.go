package log2fuse

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"io"
	"log"
)

// HTTPBodyDecoderFactory selects which decoder should run.
type HTTPBodyDecoderFactory struct {
	rawDecoder         *RawHTTPDecoder
	gzipDecoder        *GZipHTTPDecoder
	compressDecoder    *CompressHTTPDecoder
	deflateHTTPDecoder *DeflateHTTPDecoder
}

func (f *HTTPBodyDecoderFactory) create(encoding string) HTTPBodyDecoder {
	if encoding == "gzip" {
		return f.gzipDecoder
	}
	if encoding == "compress" {
		return f.compressDecoder
	}
	if encoding == "deflate" {
		return f.deflateHTTPDecoder
	}
	return f.rawDecoder // identity and any unsupported encoding
}

func createHTTPBodyDecoderFactory(logger *log.Logger) *HTTPBodyDecoderFactory {
	return &HTTPBodyDecoderFactory{
		rawDecoder:         &RawHTTPDecoder{},
		gzipDecoder:        &GZipHTTPDecoder{logger: logger},
		compressDecoder:    &CompressHTTPDecoder{logger: logger},
		deflateHTTPDecoder: &DeflateHTTPDecoder{logger: logger},
	}
}

// HTTPBodyDecoder a body decoder strategy.
type HTTPBodyDecoder interface {
	// decodes the content
	decode(content *bytes.Buffer) (string, error)
}

// RawHTTPDecoder just returns the content as-is.
type RawHTTPDecoder struct{}

func (d *RawHTTPDecoder) decode(content *bytes.Buffer) (string, error) {
	return content.String(), nil
}

// GZipHTTPDecoder extracts the Lempel-Ziv coding (LZ77) with a 32-bit CRC.
type GZipHTTPDecoder struct {
	logger *log.Logger
}

func (d *GZipHTTPDecoder) decode(content *bytes.Buffer) (string, error) {
	gzReader, err := gzip.NewReader(content)
	if err != nil {
		d.logger.Printf("Failed to create gzip reader: %s", err)
		return "", err
	}
	defer tryClose(gzReader, d.logger)
	result, err := io.ReadAll(gzReader)
	if err != nil {
		d.logger.Printf("Failed to read gzip: %s", err)
		return "", err
	}
	return string(result), nil
}

// CompressHTTPDecoder extracts with the Lempel-Ziv-Welch (LZW) algorithm.
type CompressHTTPDecoder struct {
	logger *log.Logger
}

func (d *CompressHTTPDecoder) decode(content *bytes.Buffer) (string, error) {
	reader := lzw.NewReader(content, lzw.MSB, 8)
	defer tryClose(reader, d.logger)
	result, err := io.ReadAll(reader)
	if err != nil {
		d.logger.Printf("Failed to read compress: %s", err)
		return "", err
	}
	return string(result), nil
}

// DeflateHTTPDecoder extracts the zlib structure with the deflate compression algorithm.
type DeflateHTTPDecoder struct {
	logger *log.Logger
}

func (d *DeflateHTTPDecoder) decode(content *bytes.Buffer) (string, error) {
	reader := flate.NewReader(content)
	defer tryClose(reader, d.logger)
	result, err := io.ReadAll(reader)
	if err != nil {
		d.logger.Printf("Failed to read deflate: %s", err)
		return "", err
	}
	return string(result), nil
}
