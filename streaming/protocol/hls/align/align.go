package align

const (
	syncms        = 2 // ms
	H264DefaultHZ = 90
)

type Align struct {
	frameNum  uint64
	frameBase uint64
}

func (a *Align) Align(dts *uint64, inc uint32) {
	aFrameDts := *dts
	estPts := a.frameBase + a.frameNum*uint64(inc)
	var dPts uint64
	if estPts >= aFrameDts {
		dPts = estPts - aFrameDts
	} else {
		dPts = aFrameDts - estPts
	}

	if dPts <= uint64(syncms)*H264DefaultHZ {
		a.frameNum++
		*dts = estPts
		return
	}
	a.frameNum = 1
	a.frameBase = aFrameDts
}
