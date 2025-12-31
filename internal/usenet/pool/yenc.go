package usenet_pool

import (
	"io"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/mnightingale/rapidyenc"
)

var yencLog = logger.Scoped("nntp/stream/yenc")

type YEncHeader struct{ rapidyenc.DecodedMeta }

func (h *YEncHeader) ByteRange() ByteRange {
	return ByteRange{
		Start: h.Begin() - 1,
		End:   h.End(),
	}
}

type prependReader struct {
	prepended []byte
	reader    io.Reader
}

func (p *prependReader) Read(buf []byte) (n int, err error) {
	if len(p.prepended) > 0 {
		n = copy(buf, p.prepended)
		p.prepended = p.prepended[n:]
		return n, nil
	}
	return p.reader.Read(buf)
}

type YEncDecoder struct {
	decoder *rapidyenc.Decoder
	closer  io.Closer
	reader  io.Reader
	header  *YEncHeader
}

func NewYEncDecoder(r io.Reader) *YEncDecoder {
	decoder := rapidyenc.NewDecoder(r)
	yd := &YEncDecoder{
		decoder: decoder,
		reader:  decoder,
	}

	if closer, ok := r.(io.Closer); ok {
		yd.closer = closer
	}

	return yd
}

func (d *YEncDecoder) Header() (*YEncHeader, error) {
	if d.header != nil {
		return d.header, nil
	}

	buf := make([]byte, 32*1024)
	n, err := d.decoder.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	meta := d.decoder.Meta
	d.header = &YEncHeader{meta}

	yencLog.Trace("header parsed", "filename", meta.FileName, "file_size", meta.FileSize, "part_number", meta.PartNumber, "begin", meta.Begin(), "end", meta.End())

	if n > 0 {
		d.reader = &prependReader{
			prepended: buf[:n],
			reader:    d.decoder,
		}
	}

	return d.header, nil
}

func (d *YEncDecoder) Read(p []byte) (n int, err error) {
	if d.header == nil {
		if _, err := d.Header(); err != nil {
			return 0, err
		}
	}

	return d.reader.Read(p)
}

func (d *YEncDecoder) Close() error {
	if d.closer != nil {
		return d.closer.Close()
	}
	return nil
}

type YEncDecodedData struct {
	Header *YEncHeader
	Body   []byte
}

func (d *YEncDecoder) ReadAll() (*YEncDecodedData, error) {
	yencLog.Trace("ReadAll started")

	header, err := d.Header()
	if err != nil {
		return nil, err
	}

	body := make([]byte, 0, header.PartSize)

	buf := make([]byte, 32*1024)
	for {
		n, err := d.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	yencLog.Trace("ReadAll done", "decoded_size", len(body))

	return &YEncDecodedData{
		Header: header,
		Body:   body,
	}, nil
}

type DecodedBody struct{ *YEncDecoder }

func NewDecodedBody(body io.ReadCloser) *DecodedBody {
	return &DecodedBody{NewYEncDecoder(body)}
}
