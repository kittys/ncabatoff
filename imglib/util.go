package imglib

// scaleUpRowTriple copies byte-triples from src to dest, putting two consecutive
// copies in dest, so if src is abcdefghi, dest is abcabcdefdefghighi
func scaleUpRowTriple(src, dest []byte) {
	for i := 0; i+2 < len(src); i += 3 {
		dest[i*2+0] = src[i+0]
		dest[i*2+1] = src[i+1]
		dest[i*2+2] = src[i+2]
		dest[i*2+3] = src[i+0]
		dest[i*2+4] = src[i+1]
		dest[i*2+5] = src[i+2]
	}
}

// ScaleUpPackedTriple scales up a packed triple (e.g. RGB, BGR) by 2x.
func ScaleUpPackedTriple(pix []byte, stride int) []byte {
	r := make([]byte, 4*len(pix))
	j := 0
	for i := 0; i+stride <= len(pix); i += stride {
		srcrow := pix[i : i+stride]
		destrow1 := r[4*i : 4*i+2*stride]
		destrow2 := r[4*i+2*stride : 4*i+4*stride]
		scaleUpRowTriple(srcrow, destrow1)
		copy(destrow2, destrow1)
		j++
	}
	return r
}

func scaleUpRowQuad(src, dest []byte) {
	for i := 0; i+2 < len(src); i += 4 {
		dest[i*2+0] = src[i+0]
		dest[i*2+1] = src[i+1]
		dest[i*2+2] = src[i+2]
		dest[i*2+3] = src[i+3]
		dest[i*2+4] = src[i+0]
		dest[i*2+5] = src[i+1]
		dest[i*2+6] = src[i+2]
		dest[i*2+7] = src[i+3]
	}
}

// ScaleUpPackedQuad scales up a packed triple (e.g. RGBA) by 2x.
func ScaleUpPackedQuad(pix []byte, stride int) []byte {
	r := make([]byte, 4*len(pix))
	j := 0
	for i := 0; i+stride <= len(pix); i += stride {
		srcrow := pix[i : i+stride]
		destrow1 := r[4*i : 4*i+2*stride]
		destrow2 := r[4*i+2*stride : 4*i+4*stride]
		scaleUpRowQuad(srcrow, destrow1)
		copy(destrow2, destrow1)
		j++
	}
	return r
}
