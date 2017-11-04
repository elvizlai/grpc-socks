package lib

import (
	"io"

	"github.com/golang/snappy"
	"google.golang.org/grpc/encoding"
)

type snappyCP struct {
}

func Snappy() encoding.Compressor {
	return &snappyCP{}
}

func (c *snappyCP) Compress(w io.Writer) (io.WriteCloser, error) {
	return snappy.NewBufferedWriter(w), nil
}

func (c *snappyCP) Decompress(r io.Reader) (io.Reader, error) {
	return snappy.NewReader(r), nil
}

func (c *snappyCP) Name() string {
	return "snappy"
}
