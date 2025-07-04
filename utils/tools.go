package utils

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/nfnt/resize"
)

var errImage = errors.New("image type is invalid")

func Conditional[T any](expr bool, a, b T) T {
	if expr {
		return a
	}

	return b
}

func ResizeImage(rbuf []byte) ([]byte, error) {
	debuf, layout, err := image.Decode(bytes.NewReader(rbuf))
	if err != nil {
		return nil, err
	}

	set := resize.Resize(64, 64, debuf, resize.Lanczos3)
	nbuf := bytes.Buffer{}

	switch layout {
	case "png":
		err = png.Encode(&nbuf, set)

	case "jpeg", "jpg":
		err = jpeg.Encode(&nbuf, set, &jpeg.Options{Quality: 80})

	default:
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return nbuf.Bytes(), nil
}
