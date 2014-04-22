package motion

import "image"

// slate is a simple tool used in rectJoiner to 
type slate struct {
	cols  []int
	rects []image.Rectangle
}

// return a new slate width pixels wide
func newSlate(width int) *slate {
	return &slate{cols: make([]int, width)}
}

// clean slate
func (s *slate) clean() {
	for i := range s.cols {
		s.cols[i] = 0
	}
}

// put the ith rect on the slate by incrementing the value of columns it overlaps
func (s *slate) draw(i int) {
	r := s.rects[i]
	for x := r.Min.X; x < r.Max.X; x++ {
		s.cols[x] = i + 1
	}
}

// put all rects on the slate
func (s *slate) drawall() {
	for i := range s.rects {
		s.draw(i)
	}
}

// add is used to add a new rect from the next row.
// add looks at each slate column covered by r and for each rect found, wipes the
// rect from s.rects and adds c to the union.
// finally it adds the new rect (or union) to s.rects and draws it to cols.
func (s *slate) add(o image.Rectangle) {
	r := image.Rectangle(o)
	ru := r.Inset(-1)

	maxx := ru.Max.X
	if maxx > len(s.cols) {
		maxx = len(s.cols)
	}
	minx := ru.Min.X
	if minx < 0 {
		minx = 0
	}

	lastc := 0
	for x := minx; x < maxx; x++ {
		if c := s.cols[x]; c != 0 {
			if c != lastc {
				ru = ru.Union(s.rects[c-1].Inset(-1))
				s.rects[c-1] = image.ZR
				lastc = c
			}
		}
	}

	r = ru.Inset(1)
	s.rects = append(s.rects, r)
	s.draw(len(s.rects) - 1)
}

type rectjoiner struct {
	*slate
	done    []image.Rectangle
	rectbuf []image.Rectangle
}

func newRectJoiner(width int) *rectjoiner {
	rj := rectjoiner{rectbuf: make([]image.Rectangle, 10), slate: newSlate(width)}
	return &rj
}

func (rj *rectjoiner) addrow(rs []image.Rectangle) {
	if len(rj.rects) == 0 {
		// TODO simplify copy
		if len(rs) != 0 {
			for i := range rs {
				rj.rects = append(rj.rects, image.Rectangle(rs[i]))
			}
		}
		return
	}

	if len(rs) == 0 {
		rj.done = append(rj.done, rj.rects...)
		rj.rects = rj.rects[:0]
		rj.slate.clean()
	} else {
		rj.merge(rs)
	}
}

func (rj *rectjoiner) merge(rs []image.Rectangle) {
	rj.slate.clean()
	rj.slate.drawall()

	for i := range rs {
		rj.slate.add(rs[i])
	}

	rj.rectbuf = rj.rectbuf[:0]
	for _, r := range rj.rects {
		if r != image.ZR {
			if r.Max.Y < rs[0].Max.Y {
				rj.done = append(rj.done, r)
			} else {
				rj.rectbuf = append(rj.rectbuf, r)
			}
		}
	}
	rj.rectbuf, rj.rects = rj.rects, rj.rectbuf
}

// Given a slice of RowRects , return the rectangles formed by uniting adjacent
// rectangles.
func FindConnectedRects(width int, rrects []RowRects) []image.Rectangle {
	rj := newRectJoiner(width)
	for i := range rrects {
		rj.addrow(rrects[i])
	}

	rj.done = append(rj.done, rj.rects...)
	return rj.done
}

