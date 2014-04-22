package motion

// ringbuf adapted from https://github.com/kylelemons/iq/blob/master/iq_ring.go
type ringbuf struct {
    cnt, i int
    data []interface{}
}

func (rb *ringbuf) Size() int {
    return rb.cnt
}

func (rb *ringbuf) Peek() interface{} {
    return rb.data[rb.i]
}

func (rb *ringbuf) Enqueue(x interface{}) {
    if rb.cnt >= len(rb.data) {
        panic("no room in buffer")
    }
    rb.data[(rb.i + rb.cnt) % len(rb.data)] = x
    rb.cnt++
}

func (rb *ringbuf) Dequeue() {
    rb.cnt, rb.i = rb.cnt - 1, (rb.i + 1) % len(rb.data)
}
