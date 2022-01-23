package align

const (
	syncms          uint64 = 2 // ms
	H264_default_hz uint64 = 90
)

type Align struct {
	frameNum  uint64
	frameBase uint64
}

func (a *Align) Align(dts uint64, inc uint32) uint64 {
	aFrameDts := dts
	estPts := a.frameBase + a.frameNum*uint64(inc)
	var dPts uint64
	if estPts >= aFrameDts {
		dPts = estPts - aFrameDts
	} else {
		dPts = aFrameDts - estPts
	}

	if dPts <= syncms*H264_default_hz {
		a.frameNum++
		dts = estPts
		return dts
	}

	a.frameNum = 1
	a.frameBase = aFrameDts

	return dts
}
