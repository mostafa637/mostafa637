package storage

import (
	"encoding/binary"
	"fmt"
)

// Stat is the four-word metadata record stored by the original iSH fakefs.
// Its field order and 16-byte little-endian representation match C's
// struct ish_stat on the supported little-endian targets.
type Stat struct {
	Mode uint32
	UID  uint32
	GID  uint32
	Rdev uint32
}

const statBlobSize = 16

func (s Stat) MarshalBinary() ([]byte, error) {
	b := make([]byte, statBlobSize)
	binary.LittleEndian.PutUint32(b[0:4], s.Mode)
	binary.LittleEndian.PutUint32(b[4:8], s.UID)
	binary.LittleEndian.PutUint32(b[8:12], s.GID)
	binary.LittleEndian.PutUint32(b[12:16], s.Rdev)
	return b, nil
}

func (s *Stat) UnmarshalBinary(b []byte) error {
	if len(b) != statBlobSize {
		return fmt.Errorf("storage: invalid ish_stat blob length %d, want %d", len(b), statBlobSize)
	}
	s.Mode = binary.LittleEndian.Uint32(b[0:4])
	s.UID = binary.LittleEndian.Uint32(b[4:8])
	s.GID = binary.LittleEndian.Uint32(b[8:12])
	s.Rdev = binary.LittleEndian.Uint32(b[12:16])
	return nil
}

func decodeStat(b []byte) (Stat, error) {
	var s Stat
	if err := s.UnmarshalBinary(b); err != nil {
		return Stat{}, err
	}
	return s, nil
}
