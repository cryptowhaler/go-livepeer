package core

import (
	"testing"

	"github.com/livepeer/go-livepeer/drivers"
	"github.com/livepeer/go-livepeer/net"
	"github.com/livepeer/lpms/ffmpeg"

	"github.com/flyingmutant/rapid"
	"github.com/stretchr/testify/assert"
)

func TestCapability_NewString(t *testing.T) {
	assert := assert.New(t)

	// simple case
	str := NewCapabilityString([]Capability{-10, -1, 0, 1, 2, 3, 4, 5})
	assert.Equal(CapabilityString([]uint64{63}), str)

	// test skipping
	str = NewCapabilityString([]Capability{193, 192})
	assert.Equal(CapabilityString([]uint64{0, 0, 0, 3}), str)

	// out of order inserts
	str = NewCapabilityString([]Capability{193, 54, 192, 79})
	assert.Equal(CapabilityString([]uint64{1 << 54, 1 << 15, 0, 3}), str)

}

func TestCapability_CompatibleBitstring(t *testing.T) {
	assert := assert.New(t)

	// sanity check a simple case
	compatible := NewCapabilityString([]Capability{0, 1, 2, 3}).CompatibleWith([]uint64{15})
	assert.True(compatible)

	rapid.Check(t, func(t *rapid.T) {

		// generate initial list of caps
		nbCaps := rapid.IntsRange(0, 512).Draw(t, "nbCaps").(int)
		isSet := rapid.IntsRange(0, 1)
		caps := []Capability{}
		for i := 0; i < nbCaps; i++ {
			if 1 == isSet.Draw(t, "isSet").(int) {
				caps = append(caps, Capability(i))
			}
		}

		// generate a subset of caps
		reductionSz := rapid.IntsRange(0, len(caps)).Draw(t, "reductionSz").(int)
		subsetCaps := make([]Capability, len(caps))
		copy(subsetCaps, caps)
		for i := 0; i < reductionSz; i++ {
			// select an index k, and remove it
			k := rapid.IntsRange(0, len(subsetCaps)-1).Draw(t, "k").(int)
			subsetCaps[k] = subsetCaps[len(subsetCaps)-1]
			subsetCaps = subsetCaps[:len(subsetCaps)-1]
		}
		assert.Len(subsetCaps, len(caps)-reductionSz) // sanity check

		c1 := NewCapabilityString(subsetCaps)
		c2 := NewCapabilityString(caps)

		// caps should be compatible with subset
		assert.True(c1.CompatibleWith(c2), "caps is not compatible with subset")

		if reductionSz > 0 {
			// subset should not be compatible with caps
			assert.False(c2.CompatibleWith(c1), "subset was compatible with caps")
		} else {
			assert.Equal(c2, c1)
		}
	})
}

func TestCapability_JobCapabilities(t *testing.T) {
	assert := assert.New(t)

	checkSuccess := func(params *StreamParameters, caps []Capability) bool {
		jobCaps, err := JobCapabilities(params)
		ret := assert.Nil(err)
		expectedCaps := &Capabilities{bitstring: NewCapabilityString(caps)}
		ret = assert.Equal(expectedCaps, jobCaps) && ret
		return ret
	}

	// check with everything empty
	assert.True(checkSuccess(&StreamParameters{}, []Capability{
		Capability_H264,
	}), "failed with empty params")

	// check with everything enabled
	profs := []ffmpeg.VideoProfile{
		{Format: ffmpeg.FormatMPEGTS},
		{Format: ffmpeg.FormatMP4},
		{FramerateDen: 1},
	}
	storage := drivers.NewS3Driver("", "", "", "").NewSession("")
	params := &StreamParameters{Profiles: profs, OS: storage}
	assert.True(checkSuccess(params, []Capability{
		Capability_H264,
		Capability_MP4,
		Capability_MPEGTS,
		Capability_FractionalFramerates,
		Capability_StorageS3,
	}), "failed with everything enabled")

	// check fractional framerates
	params.Profiles = []ffmpeg.VideoProfile{{FramerateDen: 1}}
	params.OS = nil
	assert.True(checkSuccess(params, []Capability{
		Capability_H264,
		Capability_MPEGTS,
		Capability_FractionalFramerates,
	}), "failed with fractional framerates")

	// check error case with profiles
	params.Profiles = []ffmpeg.VideoProfile{{Format: -1}}
	_, err := JobCapabilities(params)
	assert.Equal(capFormatConv, err)

	// check error case with storage
	params.Profiles = nil
	params.OS = &stubOS{storageType: -1}
	_, err = JobCapabilities(params)
	assert.Equal(capStorageConv, err)
}

func TestCapability_FormatToCapability(t *testing.T) {
	assert := assert.New(t)
	// Ensure all ffmpeg-enumerated formats are represented during conversion
	for _, format := range ffmpeg.ExtensionFormats {
		_, err := formatToCapability(format)
		assert.Nil(err)
	}
	// ensure error is triggered for unrepresented values
	c, err := formatToCapability(-100)
	assert.Equal(Capability_Invalid, c)
	assert.Equal(capFormatConv, err)
}

const stubOSMagic = 0x1337

type stubOS struct {
	//storageType net.OSInfo_StorageType
	storageType int32
}

func (os *stubOS) GetInfo() *net.OSInfo {
	if os.storageType == stubOSMagic {
		return nil
	}
	return &net.OSInfo{StorageType: net.OSInfo_StorageType(os.storageType)}
}
func (os *stubOS) EndSession()                             {}
func (os *stubOS) SaveData(string, []byte) (string, error) { return "", nil }
func (os *stubOS) IsExternal() bool                        { return false }

func TestCapability_StorageToCapability(t *testing.T) {
	assert := assert.New(t)
	for _, storageType := range net.OSInfo_StorageType_value {
		os := &stubOS{storageType: storageType}
		_, err := storageToCapability(os)
		assert.Nil(err)
	}

	// test error case
	c, err := storageToCapability(&stubOS{storageType: -1})
	assert.Equal(Capability_Invalid, c)
	assert.Equal(capStorageConv, err)

	// test unused caps
	c, err = storageToCapability(&stubOS{storageType: stubOSMagic})
	assert.Equal(Capability_Unused, c)
	assert.Nil(err)

	c, err = storageToCapability(nil)
	assert.Equal(Capability_Unused, c)
	assert.Nil(err)
}
