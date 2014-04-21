package imgseq

import "path/filepath"
import "fmt"
import "image"
import "os"
import "time"
import "sort"
import "code.google.com/p/ncabatoff/imglib"
import "github.com/golang/glog"

// ImgInfo identifies images by providing them a unique id, a timestamp, and a path to the file.
type ImgInfo struct {
	SeqNum     int
	CreationTs time.Time
	Path       string
}

type Img struct {
	ImgInfo
	image.Image
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
		if t := TimeFromFname(dl.Files[i]); t != nil {
			ret[i].CreationTs = *t
		}
	}
	return ret
}

func TimeFromFname(fname string) *time.Time {
	epochNanos := int64(0)
	if _, err := fmt.Sscanf(fname[4:], "%d", &epochNanos); err != nil {
		// errLg.Printf("Could not get time from fname=%s: %v", fname, err)
		return nil
	}
	t := time.Unix(0, epochNanos)
	return &t
}

// Read and possibly convert or decode the input file
func LoadImage(path string) (image.Image, error) {
	if filepath.Ext(path) == ".yuv" {
		if yuyv, err := imglib.NewYUYVFromFile(path); err != nil {
			return nil, err
		} else {
			return yuyv, nil
		}
	}

	if file, err := os.Open(path); err != nil {
		return nil, err
	} else {
		defer file.Close()
		img, _, err := image.Decode(file)
		return img, err
	}
}

func LoadImg(ii ImgInfo) Img {
	if im, err := LoadImage(ii.Path); err != nil {
		glog.Fatalf("error loading image '%s': %v", ii.Path, err)
		return Img{}
	} else {
		return Img{ImgInfo: ii, Image: im}
	}
}

func LoadImages(dl DirList, imagechan chan Img) {
	for _, f := range dl.ImgInfos() {
		imagechan <- LoadImg(f)
	}
	close(imagechan)
}

