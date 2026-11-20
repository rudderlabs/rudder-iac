package ingest

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"time"
)

// readBody never drops a read error, so a cut upload whose prefix is valid
// JSON is not accepted. early answers come before auth, late ones after.
func (h *Handler) readBody(w http.ResponseWriter, r *http.Request, c *Capture) (early, late *gwError) {
	// A test ResponseRecorder has no connection, so this can fail there.
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Now().Add(h.readTimeout))
	// Shutdown expires every read before it drains. A request that gets here
	// after that must not undo it, or a stalled body holds the drain.
	if h.stopping.Load() {
		_ = rc.SetReadDeadline(time.Now())
	}

	raw, err := readAll(http.MaxBytesReader(w, r.Body, maxReqSize), r.ContentLength)
	c.Body, c.BodyComplete = raw, err == nil

	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		// Stopping before the parse keeps memory bounded.
		return &errRequestBodyTooLarge, nil
	case r.Header.Get("Content-Encoding") != "gzip":
		c.Decoded = raw
		if err != nil {
			return nil, &errRequestBodyReadFailed
		}
		return nil, nil
	case err != nil:
		// RudderStack answers a cut gzip body with the uncompress error.
		return &errUncompress, nil
	}
	early, late = gunzip(raw, c)
	c.BodyComplete = early == nil && late == nil
	return early, late
}

// gunzip limits the decoded stream even after the ISIZE check, because the
// sender controls ISIZE.
func gunzip(raw []byte, c *Capture) (early, late *gwError) {
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return &errUncompress, nil
	}
	isize := binary.LittleEndian.Uint32(raw[len(raw)-4:])
	if isize > maxReqSize {
		return &errRequestBodyTooLarge, nil
	}

	decoded, err := readAll(io.LimitReader(zr, maxReqSize+1), int64(isize))
	switch {
	case len(decoded) > maxReqSize:
		return &errRequestBodyTooLarge, nil
	case err != nil:
		return nil, &errRequestBodyReadFailed
	}
	c.Decoded = decoded
	return nil, nil
}

// readAll is io.ReadAll with a size hint, so a large body allocates once
// instead of growing through copies.
func readAll(r io.Reader, hint int64) ([]byte, error) {
	size := 512
	if hint > 0 {
		size = int(min(hint, maxReqSize)) + 1
	}
	buf := make([]byte, 0, size)
	for {
		if len(buf) == cap(buf) {
			buf = append(buf, 0)[:len(buf)]
		}
		n, err := r.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		if errors.Is(err, io.EOF) {
			return buf, nil
		}
		if err != nil {
			return buf, err
		}
	}
}
