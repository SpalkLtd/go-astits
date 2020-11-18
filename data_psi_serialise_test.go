package astits

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

type testRun struct {
	inputPacket            []byte
	PrintInternalStructure bool
	PrintBytes             bool
}

func TestSerialisePATData(t *testing.T) {
	cases := map[string]testRun{
		"Multi-Program": testRun{
			inputPacket:            []byte{0x47, 0x40, 0x0, 0x18, 0x0, 0x0, 0xb0, 0x15, 0x7, 0x44, 0xef, 0x0, 0x0, 0x0, 0x0, 0xe0, 0x10, 0xe8, 0x80, 0xe1, 0x1, 0xe8, 0x98, 0xff, 0xc8, 0xa6, 0x6d, 0x35, 0xda, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			PrintInternalStructure: true,
		},
	}
	for name, c := range cases {
		b := c.inputPacket
		pkt, err := ParsePacket(b)
		require.NoError(t, err, name)
		b2 := make([]byte, 188)
		_, err = pkt.Serialise(b2)
		require.NoError(t, err, name)
		require.True(t, bytes.Equal(b, b2), name)
		d, err := ParsePSIPacket(pkt)
		require.NoError(t, err, name)
		if c.PrintInternalStructure { // Useful for inspecting astits structures and comparing with expectation
			fmt.Printf("%#v\n", d)
			for _, s := range d.Sections {
				fmt.Printf("\t%#v\n", s)
				fmt.Printf("\t\t%#v\n", s.Header)
				fmt.Printf("\t\t%#v\n", s.Syntax)
				if s.Syntax != nil {
					fmt.Printf("\t\t\t%#v\n", s.Syntax.Header)
					fmt.Printf("\t\t\t\t%#v\n", s.Syntax.Data.PAT)
					if s.Syntax.Data.PAT != nil {
						for _, p := range s.Syntax.Data.PAT.Programs {
							fmt.Printf("\t\t\t\t\t%#v\n", p)
						}
					}
				}

			}
		}
		b3 := make([]byte, 188)

		pkt2 := pkt
		pkt2.Payload = nil
		n, err := pkt2.Serialise(b3)
		require.NoError(t, err, name)
		n, err = d.Serialise(b3[n:])
		require.NoError(t, err, name)
		require.True(t, bytes.Equal(b, b3), name)
		require.True(t, bytes.Equal(b2, b3), name)
	}
}

func TestSerialisePMTData(t *testing.T) {

	cases := map[string]testRun{
		"Multi-PES": testRun{
			inputPacket: []byte{0x47, 0x41, 0x1, 0x1a, 0x0, 0x2, 0xb0, 0x88, 0xe8, 0x80, 0xef, 0x0, 0x0, 0xe1, 0x0, 0xf0, 0x0, 0x1b, 0xe1, 0x11, 0xf0, 0x3, 0x52, 0x1, 0x0, 0x11, 0xe1, 0x12, 0xf0, 0x7, 0x7c, 0x2, 0x2e, 0x0, 0x52, 0x1, 0x10, 0x11, 0xe1, 0x13, 0xf0, 0x7, 0x7c, 0x2, 0x2e, 0x0, 0x52, 0x1, 0x11, 0x6, 0xe1, 0x16, 0xf0, 0x8, 0x52, 0x1, 0x30, 0xfd, 0x3, 0x0, 0x8, 0x3d, 0xb, 0xe3, 0x84, 0xf0, 0x28, 0x13, 0x4, 0x0, 0x0, 0x0, 0x1, 0x14, 0xd, 0x0, 0x40, 0x0, 0x0, 0x8, 0x80, 0x0, 0x0, 0x0, 0xff, 0xff, 0xff, 0xff, 0x52, 0x1, 0x40, 0xfd, 0xe, 0x0, 0xa0, 0xa4, 0x0, 0x0, 0x0, 0xa, 0x0, 0x64, 0x0, 0x0, 0x0, 0x1, 0x1f, 0x5, 0xe1, 0xf4, 0xf0, 0x4, 0xfd, 0x2, 0x0, 0xa3, 0x11, 0xe1, 0x14, 0xf0, 0x7, 0x7c, 0x2, 0x2e, 0x0, 0x52, 0x1, 0x12, 0x11, 0xe1, 0x15, 0xf0, 0x7, 0x7c, 0x2, 0x2e, 0x0, 0x52, 0x1, 0x13, 0x9b, 0xfd, 0xa6, 0x32, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
		"Single-PES": testRun{
			inputPacket: []byte{0x47, 0x50, 0x0, 0x10, 0x0, 0x2, 0xb0, 0x17, 0x0, 0x1, 0xc1, 0x0, 0x0, 0xe1, 0x0, 0xf0, 0x0, 0x1b, 0xe1, 0x0, 0xf0, 0x0, 0x11, 0xe1, 0x1, 0xf0, 0x0, 0x5f, 0x91, 0xfc, 0x8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
	}
	for name, c := range cases {
		b := c.inputPacket
		pkt, err := ParsePacket(b)
		require.NoError(t, err, name)
		b2 := make([]byte, 188)
		_, err = pkt.Serialise(b2)
		require.NoError(t, err, name)
		require.True(t, bytes.Equal(b, b2), name)
		d, err := ParsePSIPacket(pkt)
		require.NoError(t, err, name)
		if c.PrintInternalStructure { // Useful for inspecting astits structures and comparing with expectation
			fmt.Printf("%#v\n", d)
			for _, s := range d.Sections {
				fmt.Printf("\t%#v\n", s)
				fmt.Printf("\t\t%#v\n", s.Header)
				fmt.Printf("\t\t%#v\n", s.Syntax)
				if s.Syntax != nil {
					fmt.Printf("\t\t\t%#v\n", s.Syntax.Header)
					fmt.Printf("\t\t\t\t%#v\n", s.Syntax.Data.PMT)
					if s.Syntax.Data.PMT != nil {
						for _, p := range s.Syntax.Data.PMT.ElementaryStreams {
							fmt.Printf("\t\t\t\t\t%#v\n", p)
							for _, des := range p.ElementaryStreamDescriptors {
								fmt.Printf("\t\t\t\t\t\t Descriptor Tag: %d Length: %d\n", des.Tag, des.Length)

							}
						}
					}
				}

			}
		}

		b3 := make([]byte, 188)

		pkt2 := pkt
		pkt2.Payload = nil
		n, err := pkt2.Serialise(b3)
		require.NoError(t, err, name)
		log.Println("TS Header: ", n)
		n, err = d.Serialise(b3[n:])
		require.NoError(t, err, name)

		if c.PrintBytes { //Useful for debugging where they are different
			for i := 0; i < 63; i++ {
				fmt.Printf("%2x,", b[i])
			}
			fmt.Printf("\n")
			for i := 0; i < 63; i++ {
				fmt.Printf("%2x,", b3[i])
			}
			fmt.Printf("\n\n")
			for i := 63; i < 188; i++ {
				fmt.Printf("%2x,", b[i])
			}
			fmt.Printf("\n")
			for i := 63; i < 188; i++ {
				fmt.Printf("%2x,", b3[i])
			}
		}
		require.True(t, bytes.Equal(b, b3), name)
		require.True(t, bytes.Equal(b2, b3), name)
	}
}
