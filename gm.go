package gm

import (
	"bufio"
	"errors"
	"image"
	"image/color"
	_ "image/jpeg"
	"io"
	"io/ioutil"
	"unsafe"
)

// #cgo pkg-config: GraphicsMagick
// #cgo openbsd LDFLAGS: -L/usr/X11R6/lib -lX11
// #cgo linux LDFLAGS: -lX11
// #cgo freebsd LDFLAGS: -lX11
// #include <magick/api.h>
// #include <string.h>
/*

void setMagick(Image *im, char *m){
	strncpy(im->magick, m, MaxTextExtent);
}


*/
import "C"

func Decode(r io.Reader) (out image.Image, err error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return out, err
	}

	var exc C.ExceptionInfo
	C.GetExceptionInfo(&exc)

	info := C.CloneImageInfo(nil)
	defer C.DestroyImageInfo(info)

	im := C.BlobToImage(info, unsafe.Pointer(&data[0]), C.size_t(len(data)), &exc)
	if im == nil {
		err = errors.New("Unable to decode image")
	} else {
		img := image.NewRGBA(image.Rect(0, 0, int(im.columns), int(im.rows)))

		m := C.CString("RGBA")
		defer C.free(unsafe.Pointer(m))
		if C.DispatchImage(im, 0, 0, im.columns, im.rows, m,
			C.CharPixel, unsafe.Pointer(&img.Pix[0]), &exc) == 0 {
			err = errors.New("Unable to dispatch image")
		} else {
			out = img
		}

		C.DestroyImage(im)
	}

	return out, err
}

func DecodeConfig(r io.Reader) (cfg image.Config, err error) {
	b := bufio.NewReader(r)
	if err != nil {
		return cfg, err
	}

	sz := b.Buffered()
	data, err := b.Peek(sz)
	if err != nil {
		return cfg, err
	}

	var exc C.ExceptionInfo
	C.GetExceptionInfo(&exc)

	info := C.CloneImageInfo(nil)
	defer C.DestroyImageInfo(info)

	im := C.PingBlob(info, unsafe.Pointer(&data[0]), C.size_t(sz), &exc)
	if im == nil {
		err = errors.New("Unable to decode image")
	} else {
		cfg.Width = int(im.columns)
		cfg.Height = int(im.rows)
		cfg.ColorModel = color.RGBAModel
		C.DestroyImage(im)
	}

	return cfg, err
}

func EncodeRGBA(w io.Writer, im *image.RGBA, kind string) error {
	var exc C.ExceptionInfo
	C.GetExceptionInfo(&exc)

	sz := im.Bounds().Max
	m := C.CString("RGBA")
	img := C.ConstituteImage(C.ulong(sz.X), C.ulong(sz.Y), m, C.CharPixel, unsafe.Pointer(&im.Pix[0]), &exc)
	C.free(unsafe.Pointer(m))

	if img != nil {
		info := C.CloneImageInfo(nil)
		defer C.DestroyImageInfo(info)

		mg := C.CString(kind)
		defer C.free(unsafe.Pointer(mg))

		C.setMagick(img, mg)

		var size C.size_t
		data := C.ImageToBlob(info, img, &size, &exc)
		defer C.free(data)

		w.Write(C.GoBytes(data, C.int(size)))
	} else {
		return errors.New("Unable to constitute image")
	}

	return nil
}

var fmts = map[string]string{
	"dpx": "SDPX",
	"psd": "8BPS",
	"xcf": "gimp xcf",
	"bmp": "BM",
}

func init() {
	C.InitializeMagick(nil)

	for k, v := range fmts {
		image.RegisterFormat(k, v, Decode, DecodeConfig)
	}
}
