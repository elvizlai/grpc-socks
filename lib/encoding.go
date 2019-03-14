package lib

import (
	"io"

	"github.com/golang/snappy"
	"google.golang.org/grpc/encoding"
)

const UDPMaxSize = 65507

type snappyCP struct {
}

// Snappy compressor
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
