package imaging

import (
	"encoding/binary"
	"errors"
)

var errGIFScan = errors.New("invalid gif structure")

// gifInfo GIFの画素データを展開せずに得られる情報
type gifInfo struct {
	width      int
	height     int
	frameCount int
}

// scanGIF GIFのブロック構造のみを走査し、論理画面サイズとフレーム数を取得します
func scanGIF(data []byte) (gifInfo, error) {
	var info gifInfo
	s := gifScanner{data: data}

	// Header + Logical Screen Descriptor
	header, ok := s.next(13)
	if !ok {
		return info, errGIFScan
	}
	if v := string(header[:6]); v != "GIF87a" && v != "GIF89a" {
		return info, errGIFScan
	}
	info.width = int(binary.LittleEndian.Uint16(header[6:8]))
	info.height = int(binary.LittleEndian.Uint16(header[8:10]))
	if !s.skipColorTable(header[10]) {
		return info, errGIFScan
	}

	for {
		c, ok := s.next(1)
		if !ok {
			return info, errGIFScan
		}
		switch c[0] {
		case 0x21: // Extension
			if _, ok := s.next(1); !ok { // label
				return info, errGIFScan
			}
			if !s.skipSubBlocks() {
				return info, errGIFScan
			}
		case 0x2C: // Image Descriptor
			desc, ok := s.next(9)
			if !ok {
				return info, errGIFScan
			}
			if !s.skipColorTable(desc[8]) {
				return info, errGIFScan
			}
			if _, ok := s.next(1); !ok { // LZW minimum code size
				return info, errGIFScan
			}
			if !s.skipSubBlocks() {
				return info, errGIFScan
			}
			info.frameCount++
		case 0x3B: // Trailer
			return info, nil
		default:
			return info, errGIFScan
		}
	}
}

type gifScanner struct {
	data []byte
	pos  int
}

func (s *gifScanner) next(n int) ([]byte, bool) {
	if n > len(s.data)-s.pos {
		return nil, false
	}
	b := s.data[s.pos : s.pos+n]
	s.pos += n
	return b, true
}

// skipColorTable packed fieldsにカラーテーブルフラグが立っていればカラーテーブルを読み飛ばします
func (s *gifScanner) skipColorTable(fields byte) bool {
	if fields&0x80 == 0 {
		return true
	}
	_, ok := s.next(3 * (1 << (1 + uint(fields&0x07))))
	return ok
}

// skipSubBlocks Block Terminatorまでのデータサブブロックを読み飛ばします
func (s *gifScanner) skipSubBlocks() bool {
	for {
		size, ok := s.next(1)
		if !ok {
			return false
		}
		if size[0] == 0 {
			return true
		}
		if _, ok := s.next(int(size[0])); !ok {
			return false
		}
	}
}
