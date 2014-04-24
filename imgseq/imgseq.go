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

type Img interface {
	GetImgInfo() ImgInfo
	GetPixelSequence() imglib.PixelSequence
	GetImage() image.Image
}

type RawImg struct {
	ImgInfo
	imglib.PixelSequence
}

func (r *RawImg) GetPixelSequence() imglib.PixelSequence {
	return r.PixelSequence
}

func (r *RawImg) GetImgInfo() ImgInfo {
	return r.ImgInfo
}

type DirList struct {
	// path to dir
	Path string
	// contents (filenames) under path
	Files []string
}

func GetDirList(dir string) (dl DirList) {
	fd, _ := os.Open(dir)
	fs, _ := fd.Readdirnames(0)
	dl.Path = dir
	for _, f := range fs {
		// TODO filter by regexp
		if filepath.Ext(f) != "" {
			dl.Files = append(dl.Files, f)
		}
	}
	sort.Sort(sort.StringSlice(dl.Files))
	return
}

func (dl DirList) Fqfns() []string {
	ret := make([]string, len(dl.Files))
	for i := range dl.Files {
		ret[i] = filepath.Join(dl.Path, dl.Files[i])
	}
	return ret
}

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

func TimeToFname(pfx string, t time.Time) string {
	return fmt.Sprintf("%s%d", pfx, t.UnixNano())
}

func TimeFromFname(pfx string, fname string) *time.Time {
	epochNanos := int64(0)
	if _, err := fmt.Sscanf(fname, pfx + "%d", &epochNanos); err != nil {
		return nil
	}
	t := time.Unix(0, epochNanos)
	return &t
}

func LoadRawImg(ii ImgInfo) Img {
	if ps, err := imglib.LoadPixelSequence(ii.Path); err != nil {
		glog.Fatalf("error loading image '%s': %v", ii.Path, err)
		return nil
	} else {
		return &RawImg{ImgInfo: ii, PixelSequence: ps}
	}
}

func LoadRawImgs(dl DirList, imagechan chan Img) {
	for _, f := range dl.ImgInfos() {
		imagechan <- LoadRawImg(f)
	}
	close(imagechan)
}

