package h264

import (
	"bytes"
	"fmt"
	"io"
)

const (
	// nalu_type_not_define = 0
	nalu_type_slice = 1 // slice_layer_without_partioning_rbsp() sliceheader
	// nalu_type_dpa        = 2  // slice_data_partition_a_layer_rbsp( ), slice_header
	// nalu_type_dpb        = 3  // slice_data_partition_b_layer_rbsp( )
	// nalu_type_dpc        = 4  // slice_data_partition_c_layer_rbsp( )
	nalu_type_idr = 5 // slice_layer_without_partitioning_rbsp( ),sliceheader
	nalu_type_sei = 6 // sei_rbsp( )
	nalu_type_sps = 7 // seq_parameter_set_rbsp( )
	nalu_type_pps = 8 // pic_parameter_set_rbsp( )
	nalu_type_aud = 9 // access_unit_delimiter_rbsp( )
	// nalu_type_eoesq      = 10 // end_of_seq_rbsp( )
	// nalu_type_eostream   = 11 // end_of_stream_rbsp( )
	// nalu_type_filler     = 12 // filler_data_rbsp( )
)

const (
	naluBytesLen int = 4
	maxSpsPpsLen int = 2 * 1024
)

var (
	ErrDecDataNil        = fmt.Errorf("dec buf is nil")
	ErrSpsData           = fmt.Errorf("sps data error")
	ErrPpsHeader         = fmt.Errorf("pps header error")
	ErrPpsData           = fmt.Errorf("pps data error")
	ErrNaluHeaderInvalid = fmt.Errorf("nalu header invalid")
	ErrVideoDataInvalid  = fmt.Errorf("video data not match")
	ErrDataSizeNotMatch  = fmt.Errorf("data size not match")
	ErrNaluBodyLen       = fmt.Errorf("nalu body len error")
)

var startCode = []byte{0x00, 0x00, 0x00, 0x01}
var naluAud = []byte{0x00, 0x00, 0x00, 0x01, 0x09, 0xf0}

type Parser struct {
	specificInfo []byte
	pps          *bytes.Buffer
}

type sequenceHeader struct {
	configVersion        byte //8bits
	avcProfileIndication byte //8bits
	profileCompatility   byte //8bits
	avcLevelIndication   byte //8bits
	reserved1            byte //6bits
	naluLen              byte //2bits
	reserved2            byte //3bits
	spsNum               byte //5bits
	ppsNum               byte //8bits
	spsLen               int
	ppsLen               int
}

func NewParser() *Parser {
	return &Parser{
		pps: bytes.NewBuffer(make([]byte, maxSpsPpsLen)),
	}
}

//return value 1:sps, value2 :pps
func (p *Parser) parseSpecificInfo(src []byte) error {
	if len(src) < 9 {
		return ErrDecDataNil
	}
	sps := []byte{}
	pps := []byte{}

	var seq sequenceHeader
	seq.configVersion = src[0]
	seq.avcProfileIndication = src[1]
	seq.profileCompatility = src[2]
	seq.avcLevelIndication = src[3]
	seq.reserved1 = src[4] & 0xfc
	seq.naluLen = src[4]&0x03 + 1
	seq.reserved2 = src[5] >> 5

	//get sps
	seq.spsNum = src[5] & 0x1f
	seq.spsLen = int(src[6])<<8 | int(src[7])

	if len(src[8:]) < seq.spsLen || seq.spsLen <= 0 {
		return ErrSpsData
	}
	sps = append(sps, startCode...)
	sps = append(sps, src[8:(8+seq.spsLen)]...)

	//get pps
	tmpBuf := src[(8 + seq.spsLen):]
	if len(tmpBuf) < 4 {
		return ErrPpsHeader
	}
	seq.ppsNum = tmpBuf[0]
	seq.ppsLen = int(0)<<16 | int(tmpBuf[1])<<8 | int(tmpBuf[2])
	if len(tmpBuf[3:]) < seq.ppsLen || seq.ppsLen <= 0 {
		return ErrPpsData
	}

	pps = append(pps, startCode...)
	pps = append(pps, tmpBuf[3:]...)

	p.specificInfo = append(p.specificInfo, sps...)
	p.specificInfo = append(p.specificInfo, pps...)

	return nil
}

func (p *Parser) isNaluHeader(src []byte) bool {
	if len(src) < naluBytesLen {
		return false
	}
	return src[0] == 0x00 &&
		src[1] == 0x00 &&
		src[2] == 0x00 &&
		src[3] == 0x01
}

func (p *Parser) naluSize(src []byte) (int, error) {
	if len(src) < naluBytesLen {
		return 0, fmt.Errorf("nalusizedata invalid")
	}
	buf := src[:naluBytesLen]
	size := int(0)
	for i := 0; i < len(buf); i++ {
		size = size<<8 + int(buf[i])
	}
	return size, nil
}

func (p *Parser) getAnnexbH264(src []byte, w io.Writer) error {
	dataSize := len(src)
	if dataSize < naluBytesLen {
		return ErrVideoDataInvalid
	}
	p.pps.Reset()
	_, err := w.Write(naluAud)
	if err != nil {
		return err
	}

	index := 0
	nalLen := 0
	hasSpsPps := false
	hasWriteSpsPps := false

	for dataSize > 0 {
		nalLen, err = p.naluSize(src[index:])
		if err != nil {
			return ErrDataSizeNotMatch
		}
		index += naluBytesLen
		dataSize -= naluBytesLen
		if dataSize >= nalLen && len(src[index:]) >= nalLen && nalLen > 0 {
			nalType := src[index] & 0x1f
			switch nalType {
			case nalu_type_aud:
			case nalu_type_idr:
				if !hasWriteSpsPps {
					hasWriteSpsPps = true
					if !hasSpsPps {
						if _, err := w.Write(p.specificInfo); err != nil {
							return err
						}
					} else {
						if _, err := w.Write(p.pps.Bytes()); err != nil {
							return err
						}
					}
				}
				fallthrough
			case nalu_type_slice:
				fallthrough
			case nalu_type_sei:
				_, err := w.Write(startCode)
				if err != nil {
					return err
				}
				_, err = w.Write(src[index : index+nalLen])
				if err != nil {
					return err
				}
			case nalu_type_sps:
				fallthrough
			case nalu_type_pps:
				hasSpsPps = true
				_, err := p.pps.Write(startCode)
				if err != nil {
					return err
				}
				_, err = p.pps.Write(src[index : index+nalLen])
				if err != nil {
					return err
				}
			}
			index += nalLen
			dataSize -= nalLen
		} else {
			return ErrNaluBodyLen
		}
	}
	return nil
}

func (p *Parser) Parse(b []byte, isSeq bool, w io.Writer) (err error) {
	switch isSeq {
	case true:
		err = p.parseSpecificInfo(b)
	case false:
		// is annexb
		if p.isNaluHeader(b) {
			_, err = w.Write(b)
		} else {
			err = p.getAnnexbH264(b, w)
		}
	}
	return
}
