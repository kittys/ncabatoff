// imgseq contains a few helper tools as well as the local image type Img.
package imgseq

import "path/filepath"
import "fmt"
import "image"
import "os"
import "time"
import "sort"
import "code.google.com/p/ncabatoff/imglib"
import "github.com/golang/glog"

var defaultPrefix = "test"

// ImgInfo identifies images by providing them a unique id, a timestamp, and an
// optional path to the file if any.
type ImgInfo struct {
	SeqNum     int
	CreationTs time.Time
	Path       string
}

// Img is a wrapper for image.Image, imglib.PixelSequence, and ImgInfo.  The
// first two are simply the two kinds of more concrete ways to represent images
// used locally, whereas ImgInfo allows us to have an internal id that defines
// an ordering on a sequence of images, and store some extra metadata.
type Img interface {
	GetImgInfo() ImgInfo
	GetPixelSequence() imglib.PixelSequence
	GetImage() image.Image
}

// RawImg is a simple Img implementation based on a PixelSequence.  It's
// cheap to convert between image.Image and PixelSequence so we could've
// used either.  But I want to emphasize that this is intended for raw
// images like RGB and YUYV.
type RawImg struct {
	ImgInfo
	imglib.PixelSequence
}

// GetPixelSequence returns the image data as a PixelSequence.
func (r *RawImg) GetPixelSequence() imglib.PixelSequence {
	return r.PixelSequence
}

// GetImgInfo returns the ImgInfo.
func (r *RawImg) GetImgInfo() ImgInfo {
	return r.ImgInfo
}

// DirList holds a directory path and a list of its unqualified files.
type DirList struct {
	Path string
	Files []string
}

// GetDirList reads a directory and returns its contents excluding files without extensions.
func GetDirList(dir string) (DirList, error) {
	if fd, err := os.Open(dir); err != nil {
		return DirList{}, err
	} else if fs, err := fd.Readdirnames(0); err != nil {
		return DirList{}, err
	} else {
		dl := DirList{Path: dir, Files: make([]string, len(fs))}
		for i, f := range fs {
			if filepath.Ext(f) != "" {
				dl.Files[i] = f
			}
		}
		sort.Sort(sort.StringSlice(dl.Files))
		return dl, nil
	}
}

// Fqfns returns the files fully qualified.
func (dl DirList) Fqfns() []string {
	ret := make([]string, len(dl.Files))
	for i := range dl.Files {
		ret[i] = filepath.Join(dl.Path, dl.Files[i])
	}
	return ret
}

// ImgInfos returns the ImgInfo for each file in the DirList, if possible
// obtaining the CreationTs from the filename using TimeFromFname.
func (dl DirList) ImgInfos() []ImgInfo {
	ret := make([]ImgInfo, len(dl.Files))
	for i := range dl.Files {
		ret[i] = ImgInfo{Path: filepath.Join(dl.Path, dl.Files[i])}
		ret[i].SeqNum = i
		if t := TimeFromFname(defaultPrefix, dl.Files[i]); t != nil {
			ret[i].CreationTs = *t
		}
	}
	return ret
}

// TimeToFname returns the suggested filename for pix given creation time t and prefix pfx.
func TimeToFname(pfx string, t time.Time) string {
	return fmt.Sprintf("%s%d", pfx, t.UnixNano())
}

// TimeFromFname is the inverse of TimeToFname.  It returns nil if the fname
// can't have come from TimeToFname with the given pfx.
func TimeFromFname(pfx string, fname string) *time.Time {
	epochNanos := int64(0)
	if _, err := fmt.Sscanf(fname, pfx + "%d", &epochNanos); err != nil {
		return nil
	}
	t := time.Unix(0, epochNanos)
	return &t
}

// LoadRawImgOrDie returns an Img for ii, calling glog.Fatalf on failure.
func LoadRawImgOrDie(ii ImgInfo) Img {
	if ps, err := imglib.LoadPixelSequence(ii.Path); err != nil {
		glog.Fatalf("error loading image '%s': %v", ii.Path, err)
		return nil
	} else {
		return &RawImg{ImgInfo: ii, PixelSequence: ps}
	}
}

// LoadRawImgsOrDie calls LoadRawImgsOrDie for each file in dl and sends
// the images to imagechan.
func LoadRawImgsOrDie(dl DirList, imagechan chan<- Img) {
	for _, f := range dl.ImgInfos() {
		imagechan <- LoadRawImgOrDie(f)
	}
	close(imagechan)
}

