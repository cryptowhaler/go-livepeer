package core

import (
	"errors"

	"github.com/livepeer/go-livepeer/drivers"
	"github.com/livepeer/go-livepeer/net"
	"github.com/livepeer/lpms/ffmpeg"
)

type Capability int
type CapabilityString []uint64
type Constraints struct{}
type Capabilities struct {
	bitstring   CapabilityString
	constraints Constraints
}

const (
	Capability_Invalid Capability = iota - 2
	Capability_Unused
	Capability_H264
	Capability_MPEGTS
	Capability_MP4
	Capability_FractionalFramerates
	Capability_StorageDirect
	Capability_StorageS3
	Capability_StorageGCS
)

var capFormatConv = errors.New("capability: unknown format")
var capStorageConv = errors.New("capability: unknown storage")

func NewCapabilityString(caps []Capability) CapabilityString {
	capStr := []uint64{}
	for _, v := range caps {
		if v <= Capability_Unused {
			continue
		}
		int_index := int(v) / 64 // floors automatically
		bit_index := int(v) % 64
		// grow capStr until it's of length int_index
		for len(capStr) <= int_index {
			capStr = append(capStr, 0)
		}
		capStr[int_index] |= uint64(1 << bit_index)
	}
	return capStr

}

func (bcast CapabilityString) CompatibleWith(orch CapabilityString) bool {
	// checks: ( bcastCap AND orchCap ) == bcastCap
	if len(bcast) > len(orch) {
		return false
	}
	for i := range bcast {
		if (bcast[i] & orch[i]) != bcast[i] {
			return false
		}
	}
	return true
}

func JobCapabilities(params *StreamParameters) (*Capabilities, error) {
	caps := make(map[Capability]bool)
	caps[Capability_H264] = true

	// capabilities based on requested output
	for _, v := range params.Profiles {
		// set format
		c, err := formatToCapability(v.Format)
		if err != nil {
			return nil, err
		}
		caps[c] = true

		// fractional framerates
		if v.FramerateDen > 0 {
			caps[Capability_FractionalFramerates] = true
		}
	}

	// capabilities based on broadacster or stream properties

	// set expected storage
	storageCap, err := storageToCapability(params.OS)
	if err != nil {
		return nil, err
	}
	caps[storageCap] = true

	// generate bitstring
	capList := []Capability{}
	for k, _ := range caps {
		capList = append(capList, k)
	}

	return &Capabilities{bitstring: NewCapabilityString(capList)}, nil
}

func (bcast *Capabilities) CompatibleWith(orch *net.Capabilities) bool {
	if bcast == nil {
		// Weird golang behavior: interface value can evaluate to non-nil
		// even if the underlying concrete type is nil.
		// cf. common.CapabilityComparator
		return false
	}
	return bcast.bitstring.CompatibleWith(orch.Bitstring)
}

func (bcast *Capabilities) ToNetCapabilities() *net.Capabilities {
	return &net.Capabilities{Bitstring: bcast.bitstring}
}

func NewCapabilities(caps []Capability) *Capabilities {
	return &Capabilities{bitstring: NewCapabilityString(caps)}
}

func formatToCapability(format ffmpeg.Format) (Capability, error) {
	switch format {
	case ffmpeg.FormatNone:
		return Capability_MPEGTS, nil
	case ffmpeg.FormatMPEGTS:
		return Capability_MPEGTS, nil
	case ffmpeg.FormatMP4:
		return Capability_MP4, nil
	}
	return Capability_Invalid, capFormatConv
}

func storageToCapability(os drivers.OSSession) (Capability, error) {
	if os == nil || os.GetInfo() == nil {
		return Capability_Unused, nil // unused
	}
	switch os.GetInfo().StorageType {
	case net.OSInfo_S3:
		return Capability_StorageS3, nil
	case net.OSInfo_GOOGLE:
		return Capability_StorageGCS, nil
	case net.OSInfo_DIRECT:
		return Capability_StorageDirect, nil
	}
	return Capability_Invalid, capStorageConv
}

var legacyCapabilities = []Capability{
	Capability_H264,
	Capability_MPEGTS,
	Capability_MP4,
	Capability_FractionalFramerates,
	Capability_StorageDirect,
	Capability_StorageS3,
	Capability_StorageGCS,
}
var legacyCapabilityString = NewCapabilityString(legacyCapabilities)

func (bcast *Capabilities) LegacyOnly() bool {
	if bcast == nil {
		// Weird golang behavior: interface value can evaluate to non-nil
		// even if the underlying concrete type is nil.
		// cf. common.CapabilityComparator
		return false
	}
	return bcast.bitstring.CompatibleWith(legacyCapabilityString)
}
