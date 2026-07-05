package main

import (
	"image"
	"image/color"
	"math"

	"github.com/mattn/go-tflite"
	"gocv.io/x/gocv"
)

const (
	detSize        = 192
	lmSize         = 224
	detScoreThresh = 0.5
	lmScoreThresh  = 0.5
	detNMSThresh   = 0.3
	maxHands       = 2
)

var handConnections = [][2]int{
	{0, 1}, {1, 2}, {2, 3}, {3, 4},
	{0, 5}, {5, 6}, {6, 7}, {7, 8},
	{0, 9}, {9, 10}, {10, 11}, {11, 12},
	{0, 13}, {13, 14}, {14, 15}, {15, 16},
	{0, 17}, {17, 18}, {18, 19}, {19, 20},
	{5, 9}, {9, 13}, {13, 17},
}

type HandTracker struct {
	detInterp *tflite.Interpreter
	lmInterp  *tflite.Interpreter
	anchors   [][4]float32
}

func NewHandTracker(detInterp, lmInterp *tflite.Interpreter) *HandTracker {
	return &HandTracker{
		detInterp: detInterp,
		lmInterp:  lmInterp,
		anchors:   generateAnchors(detSize, detSize),
	}
}

func generateAnchors(w, h int) [][4]float32 {
	strides := []int{8, 16, 16, 16}
	minScale, maxScale := 0.1484375, 0.75
	n := len(strides)

	var anchors [][4]float32
	layer := 0
	for layer < n {
		var scales, ratios []float64
		end := layer
		for end < n && strides[end] == strides[layer] {
			s := minScale + (maxScale-minScale)*float64(end)/float64(n-1)
			ratios = append(ratios, 1.0)
			scales = append(scales, s)
			var sNext float64
			if end == n-1 {
				sNext = 1.0
			} else {
				sNext = minScale + (maxScale-minScale)*float64(end+1)/float64(n-1)
			}
			scales = append(scales, math.Sqrt(s*sNext))
			ratios = append(ratios, 1.0)
			end++
		}

		stride := strides[layer]
		fh := int(math.Ceil(float64(h) / float64(stride)))
		fw := int(math.Ceil(float64(w) / float64(stride)))
		nAnchors := len(scales)

		for y := 0; y < fh; y++ {
			for x := 0; x < fw; x++ {
				xc := (float64(x) + 0.5) / float64(fw)
				yc := (float64(y) + 0.5) / float64(fh)
				for i := 0; i < nAnchors; i++ {
					_ = ratios[i]
					anchors = append(anchors, [4]float32{float32(xc), float32(yc), 1, 1})
				}
			}
		}
		layer = end
	}
	return anchors
}

func sigmoid(x float32) float32 {
	x = max32(-100, min32(100, x))
	return 1 / (1 + float32(math.Exp(float64(-x))))
}

type handRegion struct {
	score      float32
	box        [4]float32
	kps        [7][2]float32
	rectCenter [2]float32
	rectSize   [2]float32
	rotation   float32
	rectPoints [4][2]float32
	landmarks  [][2]int
}

func decodeBoxes(scores, raw []float32, anchors [][4]float32) []handRegion {
	var out []handRegion
	for i, a := range anchors {
		sc := sigmoid(scores[i])
		if sc < detScoreThresh {
			continue
		}
		base := i * 18
		d := make([]float32, 18)
		for j := 0; j < 18; j++ {
			scl, off := a[2], a[0]
			if j%2 == 1 {
				scl, off = a[3], a[1]
			}
			d[j] = raw[base+j]*scl/float32(detSize) + off
		}
		d[2] -= a[0]
		d[3] -= a[1]
		d[0] -= d[2] * 0.5
		d[1] -= d[3] * 0.5
		if d[2] < 0 || d[3] < 0 {
			continue
		}
		var kps [7][2]float32
		for k := 0; k < 7; k++ {
			kps[k] = [2]float32{d[4 + k*2], d[4 + k*2 + 1]}
		}
		out = append(out, handRegion{score: sc, box: [4]float32{d[0], d[1], d[2], d[3]}, kps: kps})
	}
	return out
}

func removeLetterbox(hands []handRegion, pad [4]int) {
	x0 := float32(pad[0]) / float32(detSize)
	y0 := float32(pad[1]) / float32(detSize)
	x1 := 1 - float32(pad[2])/float32(detSize)
	y1 := 1 - float32(pad[3])/float32(detSize)
	xs, ys := x1-x0, y1-y0
	for i := range hands {
		h := &hands[i]
		h.box[0] = (h.box[0] - x0) / xs
		h.box[1] = (h.box[1] - y0) / ys
		h.box[2] /= xs
		h.box[3] /= ys
		for k := range h.kps {
			h.kps[k][0] = (h.kps[k][0] - x0) / xs
			h.kps[k][1] = (h.kps[k][1] - y0) / ys
		}
	}
}

func nms(hands []handRegion) []handRegion {
	if len(hands) == 0 {
		return nil
	}
	sorted := append([]handRegion(nil), hands...)
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].score > sorted[i].score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	var kept []handRegion
	for _, c := range sorted {
		skip := false
		for _, k := range kept {
			if iou(c.box, k.box) >= detNMSThresh {
				skip = true
				break
			}
		}
		if !skip {
			kept = append(kept, c)
			if len(kept) >= maxHands {
				break
			}
		}
	}
	return kept
}

func iou(a, b [4]float32) float32 {
	ax2, ay2 := a[0]+a[2], a[1]+a[3]
	bx2, by2 := b[0]+b[2], b[1]+b[3]
	ix1, iy1 := max32(a[0], b[0]), max32(a[1], b[1])
	ix2, iy2 := min32(ax2, bx2), min32(ay2, by2)
	inter := max32(0, ix2-ix1) * max32(0, iy2-iy1)
	u := a[2]*a[3] + b[2]*b[3] - inter
	if u <= 0 {
		return 0
	}
	return inter / u
}

func toRects(hands []handRegion, imgW, imgH int) {
	const target = float32(math.Pi / 2)
	w, h := float32(imgW), float32(imgH)
	for i := range hands {
		r := &hands[i]
		r.rectSize = [2]float32{r.box[2], r.box[3]}
		r.rectCenter = [2]float32{r.box[0] + r.rectSize[0]/2, r.box[1] + r.rectSize[1]/2}
		x0, y0 := r.kps[0][0], r.kps[0][1]
		x1, y1 := r.kps[2][0], r.kps[2][1]
		r.rotation = normRad(target - float32(math.Atan2(float64(-(y1-y0)), float64(x1-x0))))

		const scale = 2.9
		const shiftY = -0.5
		rw, rh := r.rectSize[0], r.rectSize[1]
		rot := r.rotation
		xShift := -h * rh * shiftY * float32(math.Sin(float64(rot)))
		yShift := h * rh * shiftY * float32(math.Cos(float64(rot)))
		cx := r.rectCenter[0]*w + xShift
		cy := r.rectCenter[1]*h + yShift
		long := max32(rw*w, rh*h)
		r.rectPoints = rotRect(cx, cy, long*scale, long*scale, rot)
	}
}

func normRad(a float32) float32 {
	return a - 2*float32(math.Pi)*float32(math.Floor(float64((a+float32(math.Pi))/(2*float32(math.Pi)))))
}

func rotRect(cx, cy, w, h, rot float32) [4][2]float32 {
	b := float32(math.Cos(float64(rot))) * 0.5
	a := float32(math.Sin(float64(rot))) * 0.5
	p0x, p0y := cx-a*h-b*w, cy+b*h-a*w
	p1x, p1y := cx+a*h-b*w, cy-b*h-a*w
	return [4][2]float32{{p0x, p0y}, {p1x, p1y}, {2*cx - p0x, 2*cy - p0y}, {2*cx - p1x, 2*cy - p1y}}
}

func letterbox(src gocv.Mat) (gocv.Mat, [4]int) {
	tw, th := detSize, detSize
	sw, sh := src.Cols(), src.Rows()
	scale := min64(float64(tw)/float64(sw), float64(th)/float64(sh))
	nw, nh := int(float64(sw)*scale), int(float64(sh)*scale)
	px, py := (tw-nw)/2, (th-nh)/2

	resized := gocv.NewMat()
	gocv.Resize(src, &resized, image.Pt(nw, nh), 0, 0, gocv.InterpolationLinear)
	out := gocv.NewMatWithSize(th, tw, gocv.MatTypeCV8UC3)
	out.SetTo(gocv.NewScalar(0, 0, 0, 0))
	roi := out.Region(image.Rect(px, py, px+nw, py+nh))
	resized.CopyTo(&roi)
	roi.Close()
	resized.Close()
	return out, [4]int{px, py, tw - px - nw, th - py - nh}
}

func matToFloat(m gocv.Mat) []float32 {
	h, w := m.Rows(), m.Cols()
	out := make([]float32, h*w*3)
	k := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := m.GetVecbAt(y, x)
			out[k] = float32(v[2]) / 255
			out[k+1] = float32(v[1]) / 255
			out[k+2] = float32(v[0]) / 255
			k += 3
		}
	}
	return out
}

func affineFrom3(src, dst [3][2]float32) gocv.Mat {
	srcPV := gocv.NewPointVectorFromPoints([]image.Point{
		image.Pt(int(src[0][0]), int(src[0][1])),
		image.Pt(int(src[1][0]), int(src[1][1])),
		image.Pt(int(src[2][0]), int(src[2][1])),
	})
	defer srcPV.Close()
	dstPV := gocv.NewPointVectorFromPoints([]image.Point{
		image.Pt(int(dst[0][0]), int(dst[0][1])),
		image.Pt(int(dst[1][0]), int(dst[1][1])),
		image.Pt(int(dst[2][0]), int(dst[2][1])),
	})
	defer dstPV.Close()
	return gocv.GetAffineTransform(srcPV, dstPV)
}

func warpCrop(frame gocv.Mat, pts [4][2]float32) gocv.Mat {
	src := [3][2]float32{pts[1], pts[2], pts[3]}
	dst := [3][2]float32{{0, 0}, {float32(lmSize), 0}, {float32(lmSize), float32(lmSize)}}
	M := affineFrom3(src, dst)
	defer M.Close()
	out := gocv.NewMat()
	gocv.WarpAffineWithParams(frame, &out, M, image.Pt(lmSize, lmSize), gocv.InterpolationLinear, gocv.BorderConstant, color.RGBA{})
	return out
}

func affinePoint(M gocv.Mat, x, y float32) (int, int) {
	m00 := M.GetDoubleAt(0, 0)
	m01 := M.GetDoubleAt(0, 1)
	m02 := M.GetDoubleAt(0, 2)
	m10 := M.GetDoubleAt(1, 0)
	m11 := M.GetDoubleAt(1, 1)
	m12 := M.GetDoubleAt(1, 2)
	nx := m00*float64(x) + m01*float64(y) + m02
	ny := m10*float64(x) + m11*float64(y) + m12
	return int(nx), int(ny)
}

func (ht *HandTracker) landmarks(frame gocv.Mat, r *handRegion) {
	crop := warpCrop(frame, r.rectPoints)
	defer crop.Close()

	ht.lmInterp.GetInputTensor(0).SetFloat32s(matToFloat(crop))
	if ht.lmInterp.Invoke() != tflite.OK {
		return
	}
	if ht.lmInterp.GetOutputTensor(1).Float32s()[0] < lmScoreThresh {
		return
	}

	raw := ht.lmInterp.GetOutputTensor(0).Float32s()
	src := [3][2]float32{{0, 0}, {1, 0}, {1, 1}}
	dst := [3][2]float32{r.rectPoints[1], r.rectPoints[2], r.rectPoints[3]}
	M := affineFrom3(src, dst)
	defer M.Close()

	r.landmarks = make([][2]int, 21)
	for i := 0; i < 21; i++ {
		nx := raw[i*3] / float32(lmSize)
		ny := raw[i*3+1] / float32(lmSize)
		r.landmarks[i][0], r.landmarks[i][1] = affinePoint(M, nx, ny)
	}
}

func (ht *HandTracker) Process(frame *gocv.Mat) int {
	lb, pad := letterbox(*frame)
	defer lb.Close()

	ht.detInterp.GetInputTensor(0).SetFloat32s(matToFloat(lb))
	if ht.detInterp.Invoke() != tflite.OK {
		return 0
	}

	hands := decodeBoxes(
		ht.detInterp.GetOutputTensor(1).Float32s(),
		ht.detInterp.GetOutputTensor(0).Float32s(),
		ht.anchors,
	)
	removeLetterbox(hands, pad)
	hands = nms(hands)
	if len(hands) == 0 {
		return 0
	}

	toRects(hands, frame.Cols(), frame.Rows())
	for i := range hands {
		ht.landmarks(*frame, &hands[i])
		drawHand(frame, &hands[i])
	}
	return len(hands)
}

func drawHand(mat *gocv.Mat, h *handRegion) {
	if len(h.landmarks) == 0 {
		return
	}
	line := color.RGBA{0, 255, 0, 255}
	dot := color.RGBA{0, 0, 255, 255}
	for _, c := range handConnections {
		a, b := h.landmarks[c[0]], h.landmarks[c[1]]
		gocv.Line(mat, image.Pt(a[0], a[1]), image.Pt(b[0], b[1]), line, 2)
	}
	for _, p := range h.landmarks {
		gocv.Circle(mat, image.Pt(p[0], p[1]), 4, dot, -1)
	}
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}