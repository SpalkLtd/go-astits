package astits

import (
	"errors"
	"fmt"

	"github.com/asticode/go-astikit"
)

// PSI table IDs
const (
	PSITableTypeBAT     = "BAT"
	PSITableTypeDIT     = "DIT"
	PSITableTypeEIT     = "EIT"
	PSITableTypeNIT     = "NIT"
	PSITableTypeNull    = "Null"
	PSITableTypePAT     = "PAT"
	PSITableTypePMT     = "PMT"
	PSITableTypeRST     = "RST"
	PSITableTypeSDT     = "SDT"
	PSITableTypeSIT     = "SIT"
	PSITableTypeST      = "ST"
	PSITableTypeTDT     = "TDT"
	PSITableTypeTOT     = "TOT"
	PSITableTypeUnknown = "Unknown"
)

// PSIData represents a PSI data
// https://en.wikipedia.org/wiki/Program-specific_information
type PSIData struct {
	PointerField int // Present at the start of the TS packet payload signaled by the payload_unit_start_indicator bit in the TS header. Used to set packet alignment bytes or content before the start of tabled payload data.
	Sections     []*PSISection
}

// PSISection represents a PSI section
type PSISection struct {
	CRC32  uint32 // A checksum of the entire table excluding the pointer field, pointer filler bytes and the trailing CRC32.
	Header *PSISectionHeader
	Syntax *PSISectionSyntax
}

// PSISectionHeader represents a PSI section header
type PSISectionHeader struct {
	PrivateBit             bool   // The PAT, PMT, and CAT all set this to 0. Other tables set this to 1.
	SectionLength          uint16 // The number of bytes that follow for the syntax section (with CRC value) and/or table data. These bytes must not exceed a value of 1021.
	SectionSyntaxIndicator bool   // A flag that indicates if the syntax section follows the section length. The PAT, PMT, and CAT all set this to 1.
	TableID                int    // Table Identifier, that defines the structure of the syntax section and other contained data. As an exception, if this is the byte that immediately follow previous table section and is set to 0xFF, then it indicates that the repeat of table section end here and the rest of TS data payload shall be stuffed with 0xFF. Consequently the value 0xFF shall not be used for the Table Identifier.
	TableType              string
}

// PSISectionSyntax represents a PSI section syntax
type PSISectionSyntax struct {
	Data   *PSISectionSyntaxData
	Header *PSISectionSyntaxHeader
}

// PSISectionSyntaxHeader represents a PSI section syntax header
type PSISectionSyntaxHeader struct {
	CurrentNextIndicator bool   // Indicates if data is current in effect or is for future use. If the bit is flagged on, then the data is to be used at the present moment.
	LastSectionNumber    uint8  // This indicates which table is the last table in the sequence of tables.
	SectionNumber        uint8  // This is an index indicating which table this is in a related sequence of tables. The first table starts from 0.
	TableIDExtension     uint16 // Informational only identifier. The PAT uses this for the transport stream identifier and the PMT uses this for the Program number.
	VersionNumber        uint8  // Syntax version number. Incremented when data is changed and wrapped around on overflow for values greater than 32.
}

// PSISectionSyntaxData represents a PSI section syntax data
type PSISectionSyntaxData struct {
	EIT *EITData
	NIT *NITData
	PAT *PATData
	PMT *PMTData
	SDT *SDTData
	TOT *TOTData
}

// parsePSIData parses a PSI data
func parsePSIData(i *astikit.BytesIterator) (d *PSIData, err error) {
	// Init data
	d = &PSIData{}

	// Get next byte
	var b byte
	if b, err = i.NextByte(); err != nil {
		err = fmt.Errorf("astits: fetching next byte failed: %w", err)
		return
	}

	// Pointer field
	d.PointerField = int(b)

	// Pointer filler bytes
	i.Skip(d.PointerField)

	// Parse sections
	var s *PSISection
	var stop bool
	for i.HasBytesLeft() && !stop {
		if s, stop, err = parsePSISection(i); err != nil {
			err = fmt.Errorf("astits: parsing PSI table failed: %w", err)
			return
		}
		d.Sections = append(d.Sections, s)
	}
	return
}

// parsePSISection parses a PSI section
func parsePSISection(i *astikit.BytesIterator) (s *PSISection, stop bool, err error) {
	// Init section
	s = &PSISection{}

	// Parse header
	var offsetStart, offsetSectionsEnd, offsetEnd int
	if s.Header, offsetStart, _, offsetSectionsEnd, offsetEnd, err = parsePSISectionHeader(i); err != nil {
		err = fmt.Errorf("astits: parsing PSI section header failed: %w", err)
		return
	}

	// Check whether we need to stop the parsing
	if shouldStopPSIParsing(s.Header.TableType) {
		stop = true
		return
	}

	// Check whether there's a syntax section
	if s.Header.SectionLength > 0 {
		// Parse syntax
		if s.Syntax, err = parsePSISectionSyntax(i, s.Header, offsetSectionsEnd); err != nil {
			err = fmt.Errorf("astits: parsing PSI section syntax failed: %w", err)
			return
		}

		// Process CRC32
		if hasCRC32(s.Header.TableType) {
			// Seek to the end of the sections
			i.Seek(offsetSectionsEnd)

			// Parse CRC32
			if s.CRC32, err = parseCRC32(i); err != nil {
				err = fmt.Errorf("astits: parsing CRC32 failed: %w", err)
				return
			}

			// Get CRC32 data
			i.Seek(offsetStart)
			var crc32Data []byte
			if crc32Data, err = i.NextBytes(offsetSectionsEnd - offsetStart); err != nil {
				err = fmt.Errorf("astits: fetching next bytes failed: %w", err)
				return
			}

			// Compute CRC32
			var crc32 uint32
			if crc32, err = computeCRC32(crc32Data); err != nil {
				err = fmt.Errorf("astits: computing CRC32 failed: %w", err)
				return
			}

			// Check CRC32
			if crc32 != s.CRC32 {
				err = fmt.Errorf("astits: Table CRC32 %x != computed CRC32 %x", s.CRC32, crc32)
				return
			}
		}
	}

	// Seek to the end of the section
	i.Seek(offsetEnd)
	return
}

// parseCRC32 parses a CRC32
func parseCRC32(i *astikit.BytesIterator) (c uint32, err error) {
	var bs []byte
	if bs, err = i.NextBytes(4); err != nil {
		err = fmt.Errorf("astits: fetching next bytes failed: %w", err)
		return
	}
	c = uint32(bs[0])<<24 | uint32(bs[1])<<16 | uint32(bs[2])<<8 | uint32(bs[3])
	return
}

// computeCRC32 computes a CRC32
// https://stackoverflow.com/questions/35034042/how-to-calculate-crc32-in-psi-si-packet
func computeCRC32(bs []byte) (o uint32, err error) {
	o = uint32(0xffffffff)
	for _, b := range bs {
		for i := 0; i < 8; i++ {
			if (o >= uint32(0x80000000)) != (b >= uint8(0x80)) {
				o = (o << 1) ^ 0x04C11DB7
			} else {
				o = o << 1
			}
			b <<= 1
		}
	}
	return
}

// shouldStopPSIParsing checks whether the PSI parsing should be stopped
func shouldStopPSIParsing(tableType string) bool {
	return tableType == PSITableTypeNull || tableType == PSITableTypeUnknown
}

// parsePSISectionHeader parses a PSI section header
func parsePSISectionHeader(i *astikit.BytesIterator) (h *PSISectionHeader, offsetStart, offsetSectionsStart, offsetSectionsEnd, offsetEnd int, err error) {
	// Init
	h = &PSISectionHeader{}
	offsetStart = i.Offset()

	// Get next byte
	var b byte
	if b, err = i.NextByte(); err != nil {
		err = fmt.Errorf("astits: fetching next byte failed: %w", err)
		return
	}

	// Table ID
	h.TableID = int(b)

	// Table type
	h.TableType = psiTableType(h.TableID)

	// Check whether we need to stop the parsing
	if shouldStopPSIParsing(h.TableType) {
		return
	}

	// Get next bytes
	var bs []byte
	if bs, err = i.NextBytes(2); err != nil {
		err = fmt.Errorf("astits: fetching next bytes failed: %w", err)
		return
	}

	// Section syntax indicator
	h.SectionSyntaxIndicator = bs[0]&0x80 > 0

	// Private bit
	h.PrivateBit = bs[0]&0x40 > 0

	// Section length
	h.SectionLength = uint16(bs[0]&0xf)<<8 | uint16(bs[1])

	// Offsets
	offsetSectionsStart = i.Offset()
	offsetEnd = offsetSectionsStart + int(h.SectionLength)
	offsetSectionsEnd = offsetEnd
	if hasCRC32(h.TableType) {
		offsetSectionsEnd -= 4
	}
	return
}

// hasCRC32 checks whether the table has a CRC32
func hasCRC32(tableType string) bool {
	return tableType == PSITableTypePAT ||
		tableType == PSITableTypePMT ||
		tableType == PSITableTypeEIT ||
		tableType == PSITableTypeNIT ||
		tableType == PSITableTypeTOT ||
		tableType == PSITableTypeSDT
}

// psiTableType returns the psi table type based on the table id
// Page: 28 | https://www.dvb.org/resources/public/standards/a38_dvb-si_specification.pdf
func psiTableType(tableID int) string {
	switch {
	case tableID == 0x4a:
		return PSITableTypeBAT
	case tableID >= 0x4e && tableID <= 0x6f:
		return PSITableTypeEIT
	case tableID == 0x7e:
		return PSITableTypeDIT
	case tableID == 0x40, tableID == 0x41:
		return PSITableTypeNIT
	case tableID == 0xff:
		return PSITableTypeNull
	case tableID == 0:
		return PSITableTypePAT
	case tableID == 2:
		return PSITableTypePMT
	case tableID == 0x71:
		return PSITableTypeRST
	case tableID == 0x42, tableID == 0x46:
		return PSITableTypeSDT
	case tableID == 0x7f:
		return PSITableTypeSIT
	case tableID == 0x72:
		return PSITableTypeST
	case tableID == 0x70:
		return PSITableTypeTDT
	case tableID == 0x73:
		return PSITableTypeTOT
	default:
		return PSITableTypeUnknown
	}
}

// parsePSISectionSyntax parses a PSI section syntax
func parsePSISectionSyntax(i *astikit.BytesIterator, h *PSISectionHeader, offsetSectionsEnd int) (s *PSISectionSyntax, err error) {
	// Init
	s = &PSISectionSyntax{}

	// Header
	if hasPSISyntaxHeader(h.TableType) {
		if s.Header, err = parsePSISectionSyntaxHeader(i); err != nil {
			err = fmt.Errorf("astits: parsing PSI section syntax header failed: %w", err)
			return
		}
	}

	// Parse data
	if s.Data, err = parsePSISectionSyntaxData(i, h, s.Header, offsetSectionsEnd); err != nil {
		err = fmt.Errorf("astits: parsing PSI section syntax data failed: %w", err)
		return
	}
	return
}

// hasPSISyntaxHeader checks whether the section has a syntax header
func hasPSISyntaxHeader(tableType string) bool {
	return tableType == PSITableTypeEIT ||
		tableType == PSITableTypeNIT ||
		tableType == PSITableTypePAT ||
		tableType == PSITableTypePMT ||
		tableType == PSITableTypeSDT
}

// parsePSISectionSyntaxHeader parses a PSI section syntax header
func parsePSISectionSyntaxHeader(i *astikit.BytesIterator) (h *PSISectionSyntaxHeader, err error) {
	// Init
	h = &PSISectionSyntaxHeader{}

	// Get next 2 bytes
	var bs []byte
	if bs, err = i.NextBytes(2); err != nil {
		err = fmt.Errorf("astits: fetching next bytes failed: %w", err)
		return
	}

	// Table ID extension
	h.TableIDExtension = uint16(bs[0])<<8 | uint16(bs[1])

	// Get next byte
	var b byte
	if b, err = i.NextByte(); err != nil {
		err = fmt.Errorf("astits: fetching next byte failed: %w", err)
		return
	}

	// Version number
	h.VersionNumber = uint8(b&0x3f) >> 1

	// Current/Next indicator
	h.CurrentNextIndicator = b&0x1 > 0

	// Get next byte
	if b, err = i.NextByte(); err != nil {
		err = fmt.Errorf("astits: fetching next byte failed: %w", err)
		return
	}

	// Section number
	h.SectionNumber = uint8(b)

	// Get next byte
	if b, err = i.NextByte(); err != nil {
		err = fmt.Errorf("astits: fetching next byte failed: %w", err)
		return
	}

	// Last section number
	h.LastSectionNumber = uint8(b)
	return
}

// parsePSISectionSyntaxData parses a PSI section data
func parsePSISectionSyntaxData(i *astikit.BytesIterator, h *PSISectionHeader, sh *PSISectionSyntaxHeader, offsetSectionsEnd int) (d *PSISectionSyntaxData, err error) {
	// Init
	d = &PSISectionSyntaxData{}

	// Switch on table type
	switch h.TableType {
	case PSITableTypeBAT:
		// TODO Parse BAT
	case PSITableTypeDIT:
		// TODO Parse DIT
	case PSITableTypeEIT:
		if d.EIT, err = parseEITSection(i, offsetSectionsEnd, sh.TableIDExtension); err != nil {
			err = fmt.Errorf("astits: parsing EIT section failed: %w", err)
			return
		}
	case PSITableTypeNIT:
		if d.NIT, err = parseNITSection(i, sh.TableIDExtension); err != nil {
			err = fmt.Errorf("astits: parsing NIT section failed: %w", err)
			return
		}
	case PSITableTypePAT:
		if d.PAT, err = parsePATSection(i, offsetSectionsEnd, sh.TableIDExtension); err != nil {
			err = fmt.Errorf("astits: parsing PAT section failed: %w", err)
			return
		}
	case PSITableTypePMT:
		if d.PMT, err = parsePMTSection(i, offsetSectionsEnd, sh.TableIDExtension); err != nil {
			err = fmt.Errorf("astits: parsing PMT section failed: %w", err)
			return
		}
	case PSITableTypeRST:
		// TODO Parse RST
	case PSITableTypeSDT:
		if d.SDT, err = parseSDTSection(i, offsetSectionsEnd, sh.TableIDExtension); err != nil {
			err = fmt.Errorf("astits: parsing PMT section failed: %w", err)
			return
		}
	case PSITableTypeSIT:
		// TODO Parse SIT
	case PSITableTypeST:
		// TODO Parse ST
	case PSITableTypeTOT:
		if d.TOT, err = parseTOTSection(i); err != nil {
			err = fmt.Errorf("astits: parsing TOT section failed: %w", err)
			return
		}
	case PSITableTypeTDT:
		// TODO Parse TDT
	}
	return
}

// toData parses the PSI tables and returns a set of Data
func (d *PSIData) toData(firstPacket *Packet, pid uint16) (ds []*Data) {
	// Loop through sections
	for _, s := range d.Sections {
		// Switch on table type
		switch s.Header.TableType {
		case PSITableTypeEIT:
			ds = append(ds, &Data{EIT: s.Syntax.Data.EIT, FirstPacket: firstPacket, PID: pid})
		case PSITableTypeNIT:
			ds = append(ds, &Data{FirstPacket: firstPacket, NIT: s.Syntax.Data.NIT, PID: pid})
		case PSITableTypePAT:
			ds = append(ds, &Data{FirstPacket: firstPacket, PAT: s.Syntax.Data.PAT, PID: pid})
		case PSITableTypePMT:
			ds = append(ds, &Data{FirstPacket: firstPacket, PID: pid, PMT: s.Syntax.Data.PMT})
		case PSITableTypeSDT:
			ds = append(ds, &Data{FirstPacket: firstPacket, PID: pid, SDT: s.Syntax.Data.SDT})
		case PSITableTypeTOT:
			ds = append(ds, &Data{FirstPacket: firstPacket, PID: pid, TOT: s.Syntax.Data.TOT})
		}
	}
	return
}

func (d *PSIData) Serialise(b []byte) (int, error) {
	if len(b) <= 1 {
		return 0, ErrNoRoomInBuffer
	}

	//TODO take care of pointer field
	if d.PointerField != 0 {
		return 0, errors.New("Error pointer field muxing unimplemented")
	}
	b[0] = uint8(d.PointerField)
	idx := 1
	for i := range d.Sections {
		n, err := d.Sections[i].Serialise(b[idx:])
		if err != nil {
			return idx, err
		}
		idx += n
	}
	//TODO Handle Section.TableID=255 as stuffing bytes, but for now this works
	//Stuff the rest with 0xff
	for ; idx < len(b); idx++ {
		b[idx] = 0xff
	}
	return idx, nil
}

func (s *PSISection) Serialise(b []byte) (int, error) {

	if s.Header.TableID == 255 {
		return 0, nil
	}
	if len(b) <= 3 {
		return 0, ErrNoRoomInBuffer
	}
	idx := 3 // Skip 3 byte header we put in afterward

	if s.Syntax != nil {
		n, err := s.Syntax.Serialise(b[idx:])
		if err != nil {
			return idx, err
		}
		idx += n
	}

	s.Header.SectionLength = uint16(idx + 4 - 3) // Add CRC32 field subtract initial 3 bytes

	//Serialise header afterward so we ensure the section length is accurate
	if s.Header != nil {
		_, err := s.Header.Serialise(b[0:])
		if err != nil {
			return idx, err
		}
	}

	if hasCRC32(s.Header.TableType) {

		i := astikit.NewBytesIterator(b)
		// Get CRC32 data
		var crc32Data []byte
		var err error
		if crc32Data, err = i.NextBytes(idx); err != nil {
			return idx, fmt.Errorf("astits: fetching next bytes failed: %w", err)
		}

		// Compute CRC32
		var crc32 uint32
		if crc32, err = computeCRC32(crc32Data); err != nil {
			return idx, fmt.Errorf("astits: computing CRC32 failed: %w", err)
		}

		// Check CRC32
		// TODO emit a warning here if it is not valid
		// if crc32 != s.CRC32 {
		// 	return idx, fmt.Errorf("astits: Table CRC32 %x != computed CRC32 %x", s.CRC32, crc32)
		// }
		b[idx] = uint8(crc32 >> 24)
		b[idx+1] = uint8(crc32 >> 16)
		b[idx+2] = uint8(crc32 >> 8)
		b[idx+3] = uint8(crc32)
		idx += 4
	}

	return idx, nil

}

func (h *PSISectionHeader) Serialise(b []byte) (int, error) {
	if h.TableID == 255 {
		return 0, nil
	}
	if len(b) < 3 {
		return 0, ErrNoRoomInBuffer
	}
	b[0] = uint8(h.TableID)
	b[1] = Btou8(h.SectionSyntaxIndicator)<<7 | Btou8(h.PrivateBit)<<6 | 3<<4 | uint8(0xf&(h.SectionLength>>8))
	b[2] = uint8(0xff & h.SectionLength) // TODO how do we calculate this without having done the whole section?
	return 3, nil
	// TableType              string
}

func (s *PSISectionSyntax) Serialise(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, ErrNoRoomInBuffer
	}
	idx := 0
	if s.Header != nil {
		n, err := s.Header.Serialise(b[idx:])
		if err != nil {
			return idx, err
		}
		idx += n
	}
	if s.Data != nil {
		n, err := s.Data.Serialise(b[idx:])
		if err != nil {
			return idx, err
		}
		idx += n
	}

	return idx, nil
}
func (sh *PSISectionSyntaxHeader) Serialise(b []byte) (int, error) {
	if len(b) < 5 {
		return 0, ErrNoRoomInBuffer
	}
	b[0], b[1] = U16toU8s(sh.TableIDExtension)
	reservedBits := uint8(3 << 6) //TODO figure out if reserved are always set
	b[2] = uint8((0x1f&sh.VersionNumber)<<1) | Btou8(sh.CurrentNextIndicator) | reservedBits
	b[3] = sh.SectionNumber
	b[4] = sh.LastSectionNumber
	return 5, nil
}

func (sd *PSISectionSyntaxData) Serialise(b []byte) (int, error) {

	if sd.PAT != nil {
		return sd.PAT.Serialise(b)
	}
	if sd.PMT != nil {
		return sd.PMT.Serialise(b)
	}
	//TODO implement serialisation of other packets
	// 	sd.EIT.Serialise(b)
	// 	sd.NIT.Serialise(b)
	// 	sd.SDT.Serialise(b)
	// 	sd.TOT.Serialise(b)
	return 0, nil
}

func Btou8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func U16toU8s(a uint16) (uint8, uint8) {
	return uint8(0xff & (a >> 8)), uint8(0xff & a)
}
