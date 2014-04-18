// v4l is a simple video4linux implementation in Go.
package v4l

/*
#cgo freebsd CFLAGS: -I/usr/local/include
#cgo freebsd LDFLAGS: -I/usr/local/lib -lv4lconvert -lv4l2
#cgo linux LDFLAGS: -lv4lconvert -lv4l2
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
#include <unistd.h>
#include <errno.h>
#include <stdint.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/select.h>
#include <sys/mman.h>
#include <sys/ioctl.h>
#include <linux/videodev2.h>
#include <libv4lconvert.h>
#include <libv4l2.h>

#define CLEAR(x) memset(&(x), 0, sizeof(x))

int myV4l2Open(char *name)
{
    return v4l2_open(name, O_RDWR | O_NONBLOCK);
}

void init_v4l2_fmtdesc(struct v4l2_fmtdesc *fmtdesc, int idx)
{
	CLEAR(*fmtdesc);
	fmtdesc->type = V4L2_BUF_TYPE_VIDEO_CAPTURE;
	fmtdesc->index = idx;
}

void set_v4l2_parmcap_fps(struct v4l2_streamparm *parm, unsigned nom, unsigned denom)
{
	parm->parm.capture.timeperframe.numerator = nom;
	parm->parm.capture.timeperframe.denominator = denom;
}

void get_v4l2_parmcap_fps(struct v4l2_streamparm *parm, unsigned *nom, unsigned *denom)
{
	*nom = parm->parm.capture.timeperframe.numerator;
	*denom = parm->parm.capture.timeperframe.denominator;
}

void get_v4l2_fmtdesc(struct v4l2_fmtdesc *fmtdesc, unsigned *f) //, char *desc)
{
	*f = fmtdesc->pixelformat;
	// memcpy(desc, fmtdesc->description, 32);
}

void init_v4l2_format(struct v4l2_format *fmt, int w, int h, int pf)
{
	CLEAR(*fmt);
	fmt->type = V4L2_BUF_TYPE_VIDEO_CAPTURE;
	fmt->fmt.pix.width = w;
	fmt->fmt.pix.height = h;
	fmt->fmt.pix.pixelformat = pf; // V4L2_PIX_FMT_JPEG;
	// fmt->fmt.pix.pixelformat = V4L2_PIX_FMT_YUV420;
	// fmt->fmt.pix.pixelformat = V4L2_PIX_FMT_RGB24;
	fmt->fmt.pix.field = V4L2_FIELD_ANY;
}

void get_v4l2_format(struct v4l2_format *fmt, unsigned *w, unsigned *h, unsigned *f)
{
	*w = fmt->fmt.pix.width;
	*h = fmt->fmt.pix.height;
	*f = fmt->fmt.pix.pixelformat;
}

void init_v4l2_requestbuffers(struct v4l2_requestbuffers *reqbufs, int n)
{
	CLEAR(*reqbufs);
	reqbufs->count = n;
	reqbufs->type = V4L2_BUF_TYPE_VIDEO_CAPTURE;
	reqbufs->memory = V4L2_MEMORY_MMAP;
}

void init_v4l2_buffer(struct v4l2_buffer *buf, int i)
{
	CLEAR(*buf);
	buf->index = i;
	buf->type = V4L2_BUF_TYPE_VIDEO_CAPTURE;
	buf->memory = V4L2_MEMORY_MMAP;
}

int get_v4l2_buffer_offset(struct v4l2_buffer *buf)
{
	return buf->m.offset;
}

int wait_for_fd(int fd)
{
	fd_set fds;
	struct timeval tv;

	FD_ZERO(&fds);
	FD_SET(fd, &fds);

	tv.tv_sec = 2;
	tv.tv_usec = 0;

	return select(fd + 1, &fds, NULL, NULL, &tv);
}

*/
import "C"
import "bytes"
import "fmt"
import "image"
import "os"
import "syscall"
import "time"
import "unsafe"
import "code.google.com/p/ncabatoff/imglib"

const (
	FormatYuyv = C.V4L2_PIX_FMT_YUYV
	FormatRgb = C.V4L2_PIX_FMT_RGB24
	FormatJpeg = C.V4L2_PIX_FMT_JPEG
)

type Device struct {
	name string
	file *os.File
	useV4lConvert bool
	format Format
	buffers [][]byte
	fpsnom, fpsdenom int
	capturing bool
}

type Format struct {
	Height int
	Width int
	Format uint32
}

// bufId is the actual bufnum+1 - thus the zero value is an invalid bufnum
type bufId int

type Frame struct {
	bufnum bufId
	Pix []byte
	ReqTime time.Time
	RecvTime time.Time
}

// CopyPix is used to create a new frame with the same pix but no buffer reference,
// allowing the existing one to be released with DoneFrame.
func (f Frame) CopyPix() Frame {
	newFrame := Frame{ReqTime: f.ReqTime, RecvTime: f.RecvTime}
	newFrame.Pix = make([]byte, len(f.Pix))
	copy(newFrame.Pix, f.Pix)
	return newFrame
}

func ioctl(file *os.File, req int, arg unsafe.Pointer) syscall.Errno {
	for {
		_, _, err := syscall.RawSyscall(syscall.SYS_IOCTL, file.Fd(), uintptr(req), uintptr(arg))
		if err == 0 {
			return err
		}

		if err != C.EINTR && err != C.EAGAIN {
			return err
		}
	}
}

func openWithV4lConvert(name string) (*os.File, error) {
	if fd := C.myV4l2Open(C.CString(name)); fd < 0 {
		return nil, fmt.Errorf("error doing v4l2_open on device '%s'", name)
	} else {
		return os.NewFile(uintptr(fd), name), nil
	}
}

// Open opens the named video device at the specified resolution.  If useV4lConvert is
// true it attempts to use libv4lconvert.
func OpenDevice(name string, useV4lConvert bool) (*Device, error) {
	dev := &Device{name: name, useV4lConvert: useV4lConvert}
	if useV4lConvert {
		if f, err := openWithV4lConvert(name); err != nil {
			return nil, err
		} else {
			dev.file = f
		}
	} else {
		if f, err := os.OpenFile(name, os.O_RDWR, 0); err != nil {
			return nil, fmt.Errorf("error opening capture device '%s': %v", name, err)
		} else {
			dev.file = f
		}
	}

	if err := dev.verify(); err != nil {
		if err := dev.file.Close(); err != nil {
			// should we log?
		}
		return nil, err
	}
	return dev, nil
}

func (v *Device) err(fmtstr string, args ...interface{}) error {
	errmsg := fmt.Sprintf(fmtstr, args)
	return fmt.Errorf("error on capture device %s: %s", v.name, errmsg)
}

func (v *Device) CloseDevice() error {
	err := v.file.Close()
	v.file = nil
	return err
}

func (v *Device) verify() error {
	var vcap C.struct_v4l2_capability
	if errno := ioctl(v.file, C.VIDIOC_QUERYCAP, unsafe.Pointer(&vcap)); errno != 0 {
		return v.err("error doing VIDIOC_QUERYCAP: %d", errno)
	}

	if 0 == (vcap.capabilities & C.V4L2_CAP_VIDEO_CAPTURE) {
		return v.err("not a video capture device")
	}

	if 0 == (vcap.capabilities & C.V4L2_CAP_STREAMING) {
		return v.err("does not support streaming i/o")
	}

	return nil
}

func (v *Device) SetFormat(vf Format) error {
	if v.capturing {
		return v.err("can't set format while capturing")
	}
	if len(v.buffers) > 0 {
		return v.err("can't set format while buffers allocated")
	}

	if v.useV4lConvert {
		if err := v.setFormatUseV4lConvert(vf); err != nil {
			return err
		}
	} else {
		if err := v.setFormatNoV4lConvert(vf); err != nil {
			return err
		}
	}

	var sparm C.struct_v4l2_streamparm
	sparm._type = C.V4L2_BUF_TYPE_VIDEO_CAPTURE
	if errno := ioctl(v.file, C.VIDIOC_G_PARM, unsafe.Pointer(&sparm)); errno != 0 {
		return v.err("g_parm failed for format %v: errno=%d", vf, errno)
	}
	var nom, denom C.uint
	C.get_v4l2_parmcap_fps(&sparm, &nom, &denom)
	v.fpsnom = int(nom)
	v.fpsdenom = int(denom)
	// log.Printf("g_parm fps=%d/%d", uint(nom), uint(denom))

	if errno := ioctl(v.file, C.VIDIOC_S_PARM, unsafe.Pointer(&sparm)); errno != 0 {
		return v.err("s_parm failed for format %v: errno=%d", vf, errno)
	}
	return nil
}

func (v *Device) setFormatNoV4lConvert(vf Format) error {
	var vfmt C.struct_v4l2_format
	C.init_v4l2_format(&vfmt, C.int(vf.Width), C.int(vf.Height), C.int(vf.Format))

	if errno := ioctl(v.file, C.VIDIOC_TRY_FMT, unsafe.Pointer(&vfmt)); errno != 0 {
		return v.err("try_fmt failed for format %v: errno=%d", vf, errno)
	}

	if errno := ioctl(v.file, C.VIDIOC_S_FMT, unsafe.Pointer(&vfmt)); errno != 0 {
		return v.err("s_fmt failed for format %v: errno=%d", vf, errno)
	}

	v.format = vf
	// C.get_v4l2_format(&vfmt, &w, &h, &f)
	// log.Printf("asked for pxlfmt=%x (%dx%d), got=%x (%dx%d)\n", askedf, width, height, f, w, h)
	// pxlfmt = V4lPixelFormat(f)
	return nil
}

func (v *Device) setFormatUseV4lConvert(vf Format) error {
	var vfmt C.struct_v4l2_format
	C.init_v4l2_format(&vfmt, C.int(vf.Width), C.int(vf.Height), C.int(vf.Format))

	v4lconvert_data := C.v4lconvert_create(C.int(v.file.Fd()))
	if v4lconvert_data == nil {
		return v.err("v4lconvert_create failed: %s", C.GoString(C.v4lconvert_get_error_message(v4lconvert_data)))
	}
	var src_fmt C.struct_v4l2_format
	if C.v4lconvert_try_format(v4lconvert_data, &vfmt, &src_fmt) != 0 {
		return v.err("v4lconvert_try_format failed: %s", C.GoString(C.v4lconvert_get_error_message(v4lconvert_data)))
	}
	if errno := ioctl(v.file, C.VIDIOC_S_FMT, unsafe.Pointer(&src_fmt)); errno != 0 {
		return v.err("s_fmt failed for format %v: errno=%d", vf, errno)
	}
	v.format = vf
	return nil
}

// GetSupportedFormats queries the driver for the opened video device to return a list of formats.
func (v *Device) GetSupportedFormats() ([]uint32, error) {
	var fmts []uint32
	var fmtdesc C.struct_v4l2_fmtdesc
	for i := 0; ; i++ {
		C.init_v4l2_fmtdesc(&fmtdesc, C.int(i))
		if errno := ioctl(v.file, C.VIDIOC_ENUM_FMT, unsafe.Pointer(&fmtdesc)); errno != 0 {
			if errno == C.EINVAL {
				break
			}
			return nil, v.err("ENUM_FMT failed: errno=%d", errno)
		}
		var f C.uint
		C.get_v4l2_fmtdesc(&fmtdesc, &f)
		fmts = append(fmts, uint32(f))
	}
	return fmts, nil
}

// InitBuffers initializes capture buffers for the opened video device given by file.
// Only the MMAP method is supported.
func (v *Device) InitBuffers(n int) error {
	if v.capturing {
		return v.err("can't init buffers while capturing")
	}
	if len(v.buffers) > 0 {
		return v.err("can't init buffers while buffers allocated")
	}

	var reqbufs C.struct_v4l2_requestbuffers
	C.init_v4l2_requestbuffers(&reqbufs, C.int(n))
	if errno := ioctl(v.file, C.VIDIOC_REQBUFS, unsafe.Pointer(&reqbufs)); errno != 0 {
		return v.err("failed to ioctl VIDIOC_REQBUFS to %d buffers: errno=%d", n, errno)
	}

	if int(reqbufs.count) < n {
		return v.err("not enough memory for %d buffers", n)
	}

	for i := 0; i < n; i++ {
		var buf C.struct_v4l2_buffer
		C.init_v4l2_buffer(&buf, C.int(i))

		if errno := ioctl(v.file, C.VIDIOC_QUERYBUF, unsafe.Pointer(&buf)); errno != 0 {
			return v.err("failed to ioctl VIDIOC_QUERYBUF: errno=%d", errno)
		}
		offset := C.get_v4l2_buffer_offset(&buf)
		// log.Printf("mmaping to offset=%d, length=%d\n", offset, buf.length)
		buffer, err := syscall.Mmap(int(v.file.Fd()), int64(offset), int(buf.length),
			syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			return v.err("failed to mmap buffer %d: %v", i, err)
		}
		v.buffers = append(v.buffers, buffer)
	}
	return nil
}

// DoneBuffers releases the buffers mmaped by InitBuffers.
func (v *Device) DoneBuffers() error {
	if v.capturing {
		return v.err("can't release buffers while capturing")
	}
	if len(v.buffers) == 0 {
		return v.err("no buffers to release")
	}

	for _, buf := range v.buffers {
		err := syscall.Munmap(buf)
		if err != nil {
			// log.Printf("error doing munmap: %v", err)
		}
	}
	return nil
}

// Capture enables video capture once Open and InitBuffers have been called.
func (v *Device) Capture() error {
	if v.capturing {
		return v.err("already capturing")
	}
	if len(v.buffers) == 0 {
		return v.err("can't capture without buffers")
	}
	for i := range v.buffers {
		var buf C.struct_v4l2_buffer
		C.init_v4l2_buffer(&buf, C.int(i))
		if errno := ioctl(v.file, C.VIDIOC_QBUF, unsafe.Pointer(&buf)); errno != 0 {
			return v.err("failed to ioctl VIDIOC_QBUF: errno=%d", errno)
		}
	}
	var buftype C.enum_v4l2_buf_type
	buftype = C.V4L2_BUF_TYPE_VIDEO_CAPTURE
	if errno := ioctl(v.file, C.VIDIOC_STREAMON, unsafe.Pointer(&buftype)); errno != 0 {
		return fmt.Errorf("failed to ioctl VIDIOC_STREAMON: %d", errno)
	}
	v.capturing = true
	return nil
}

// EndCapture turns off video capture.
func (v *Device) EndCapture() error {
	if !v.capturing {
		return v.err("not capturing")
	}
	var buftype C.enum_v4l2_buf_type
	buftype = C.V4L2_BUF_TYPE_VIDEO_CAPTURE
	if errno := ioctl(v.file, C.VIDIOC_STREAMOFF, unsafe.Pointer(&buftype)); errno != 0 {
		return v.err("failed to ioctl VIDIOC_QBUF: errno=%d", errno)
	}
	v.capturing = false
	return nil
}

// GetFrame returns the next frame from the capture stream.
func (v *Device) GetFrame() (Frame, error) {
	if !v.capturing {
		return Frame{}, v.err("not capturing")
	}
	// log.Printf("waiting for fd=%d\n", file.Fd())
	reqtime := time.Now()
	r, errno := C.wait_for_fd(C.int(v.file.Fd()))
	if r == 0 {
		return Frame{}, v.err("timeout on select while getting frame")
	} else if r < 0 {
		return Frame{}, v.err("error on select while getting frame: errno=%d", errno)
	}

	var buf C.struct_v4l2_buffer
	C.init_v4l2_buffer(&buf, 0)

	if errno := ioctl(v.file, C.VIDIOC_DQBUF, unsafe.Pointer(&buf)); errno != 0 {
		return Frame{}, v.err("failed to ioctl VIDIOC_DQBUF: errno=%d", errno)
	}
	f := Frame{RecvTime: time.Now(), ReqTime: reqtime}
	f.Pix, f.bufnum = v.buffers[buf.index], bufId(int(buf.index)+1)
	// log.Printf("success! bytesused=%d, length=%d\n", buf.bytesused, buf.length)
	return f, nil
}

// DoneFrame is used to return the buffer contained in Frame to the driver so it may be reused.
// For best performance call DoneFrame as soon as possible after GetFrame.
func (v *Device) DoneFrame(frame Frame) error {
	realBufnum := int(frame.bufnum - 1)
	if realBufnum < 0 || realBufnum >= len(v.buffers) {
		return v.err("invalid buffer number in frame")
	}
	var buf C.struct_v4l2_buffer
	C.init_v4l2_buffer(&buf, C.int(realBufnum))
	if errno := ioctl(v.file, C.VIDIOC_QBUF, unsafe.Pointer(&buf)); errno != 0 {
		return v.err("failed to ioctl VIDIOC_QBUF: errno=%d", errno)
	}
	return nil
}

// GetImage builds an Image from the provided Frame.
// Supported formats: YUYV returns a *imglib.YUYV, RGB24 returns a *imglib.RGB, and JPEG returns a image.Jpeg.
func (v *Device) GetImage(frame Frame) (image.Image, error) {
	switch v.format.Format {
	case C.V4L2_PIX_FMT_YUYV:
		img := imglib.NewYUYV(image.Rect(0, 0, v.format.Width, v.format.Height))
		copy(img.Pix, frame.Pix)
		return img, nil
	case C.V4L2_PIX_FMT_RGB24:
		img := imglib.NewRGB(image.Rect(0, 0, v.format.Width, v.format.Height))
		copy(img.Pix, frame.Pix)
		return img, nil
	case C.V4L2_PIX_FMT_JPEG:
		if img, _, err := image.Decode(bytes.NewReader(frame.Pix)); err != nil {
			return nil, err
		} else {
			return img, nil
		}
	}
	return nil, v.err("can't get image from frame of format %d", v.format.Format)
}
