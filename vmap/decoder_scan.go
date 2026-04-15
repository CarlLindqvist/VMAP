package vmap

import (
	"bytes"
	"errors"
	"strconv"
	"unsafe"
)

// byteStr converts b to a string without copying. The returned string
// shares memory with b; b must not be modified while the string is in use.
func byteStr(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// decodeXMLStr converts XML text bytes to a Go string, decoding entities.
// Zero-copy when no entities are present.
func decodeXMLStr(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if bytes.IndexByte(b, '&') < 0 {
		return byteStr(b)
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return string(xmlStringToString(cp))
}

// scan is a minimal byte scanner for VMAP/VAST XML.
type scan struct {
	data []byte
	pos  int
}

// next finds the next XML tag. Returns the tag name as a slice of the
// input, whether it is an end tag, and whether it is self-closing.
// After return, pos is right after the tag name (before attrs and '>').
// For end tags, pos is advanced past '>'.
func (s *scan) next() (name []byte, isEnd, selfClose bool) {
	for {
		i := bytes.IndexByte(s.data[s.pos:], '<')
		if i < 0 {
			s.pos = len(s.data)
			return nil, false, false
		}
		s.pos += i + 1
		if s.pos >= len(s.data) {
			return nil, false, false
		}

		c := s.data[s.pos]
		if c == '?' || c == '!' {
			j := bytes.IndexByte(s.data[s.pos:], '>')
			if j < 0 {
				s.pos = len(s.data)
				return nil, false, false
			}
			s.pos += j + 1
			continue
		}

		if c == '/' {
			isEnd = true
			s.pos++
		}

		start := s.pos
		for s.pos < len(s.data) {
			c = s.data[s.pos]
			if c == ' ' || c == '>' || c == '/' || c == '\t' || c == '\n' || c == '\r' {
				break
			}
			s.pos++
		}
		name = s.data[start:s.pos]
		// Strip namespace prefix (e.g., "vmap:VMAP" -> "VMAP")
		if colon := bytes.IndexByte(name, ':'); colon >= 0 {
			name = name[colon+1:]
		}

		if isEnd {
			j := bytes.IndexByte(s.data[s.pos:], '>')
			if j >= 0 {
				s.pos += j + 1
			}
			return name, true, false
		}

		j := bytes.IndexByte(s.data[s.pos:], '>')
		if j >= 0 && j > 0 && s.data[s.pos+j-1] == '/' {
			selfClose = true
		}
		return name, false, selfClose
	}
}

// attr finds the value of the named attribute in the current tag,
// matching on the local name (after any namespace prefix).
// Must be called after next() and before endAttrs().
func (s *scan) attr(name string) []byte {
	gt := bytes.IndexByte(s.data[s.pos:], '>')
	if gt < 0 {
		return nil
	}
	end := s.pos + gt
	region := s.data[s.pos:end]

	// Try ' name="' (no namespace prefix)
	var buf [64]byte
	buf[0] = ' '
	n := 1 + copy(buf[1:], name)
	buf[n] = '='
	buf[n+1] = '"'

	i := bytes.Index(region, buf[:n+2])
	if i >= 0 {
		valStart := i + n + 2
		valEnd := bytes.IndexByte(region[valStart:], '"')
		if valEnd >= 0 {
			return s.data[s.pos+valStart : s.pos+valStart+valEnd]
		}
	}

	// Try ':name="' (namespace-prefixed, e.g. xmlns:vmap="...")
	buf[0] = ':'
	i = bytes.Index(region, buf[:n+2])
	if i >= 0 {
		valStart := i + n + 2
		valEnd := bytes.IndexByte(region[valStart:], '"')
		if valEnd >= 0 {
			return s.data[s.pos+valStart : s.pos+valStart+valEnd]
		}
	}

	return nil
}

// endAttrs advances past the '>' of the current start tag.
func (s *scan) endAttrs() {
	j := bytes.IndexByte(s.data[s.pos:], '>')
	if j >= 0 {
		s.pos += j + 1
	}
}

// text extracts text or CDATA content from the current position until the
// next '<'. Returns the raw bytes and whether it was CDATA.
func (s *scan) text() (content []byte, wasCDATA bool) {
	if s.pos >= len(s.data) {
		return nil, false
	}

	// Skip whitespace before checking for CDATA
	p := s.pos
	for p < len(s.data) && (s.data[p] == ' ' || s.data[p] == '\t' || s.data[p] == '\n' || s.data[p] == '\r') {
		p++
	}

	const cdataOpen = "<![CDATA["
	const cdataClose = "]]>"
	if p+len(cdataOpen) <= len(s.data) && string(s.data[p:p+len(cdataOpen)]) == cdataOpen {
		start := p + len(cdataOpen)
		end := bytes.Index(s.data[start:], []byte(cdataClose))
		if end < 0 {
			return nil, false
		}
		s.pos = start + end + len(cdataClose)
		return s.data[start : start+end], true
	}

	i := bytes.IndexByte(s.data[s.pos:], '<')
	if i < 0 {
		return nil, false
	}
	content = bytes.TrimSpace(s.data[s.pos : s.pos+i])
	s.pos += i
	if len(content) == 0 {
		return nil, false
	}
	return content, false
}

// textStr extracts text content and returns it as a decoded string.
func (s *scan) textStr() string {
	content, wasCDATA := s.text()
	if content == nil {
		return ""
	}
	if wasCDATA {
		return byteStr(content)
	}
	return decodeXMLStr(content)
}

// --- Top-level decoders ---

// DecodeVmapScan decodes a VMAP document using direct byte scanning.
// String fields in the returned struct may reference the input slice;
// the input must not be modified while the result is in use.
func DecodeVmapScan(input []byte) (VMAP, error) {
	var vmap VMAP
	s := scan{data: input}
	found := false

	for {
		name, isEnd, _ := s.next()
		if name == nil {
			break
		}
		if isEnd {
			continue
		}

		switch string(name) {
		case "VMAP":
			found = true
			if v := s.attr("version"); v != nil {
				vmap.Version = byteStr(v)
			}
			if v := s.attr("vmap"); v != nil {
				vmap.Vmap = byteStr(v)
				vmap.XMLName.Space = byteStr(v)
			}
			vmap.XMLName.Local = "VMAP"
			s.endAttrs()
		case "AdBreak":
			vmap.AdBreaks = append(vmap.AdBreaks, scanAdBreak(&s))
		}
	}

	if !found {
		return vmap, errors.New("no VMAP token found in document")
	}
	return vmap, nil
}

// DecodeVastScan decodes a VAST document using direct byte scanning.
func DecodeVastScan(input []byte) (VAST, error) {
	var vast VAST
	s := scan{data: input}
	found := false

	for {
		name, isEnd, selfClose := s.next()
		if name == nil {
			break
		}
		if isEnd {
			continue
		}
		if string(name) == "VAST" {
			found = true
			if selfClose {
				break
			}
			vast = scanVast(&s)
		}
	}

	if !found {
		return vast, errors.New("no VAST token found in document")
	}
	return vast, nil
}

// --- Per-element scanners ---

func scanAdBreak(s *scan) AdBreak {
	var ab AdBreak
	ab.AdSource = &AdSource{VASTData: &VASTData{}}

	if v := s.attr("breakId"); v != nil {
		ab.Id = byteStr(v)
	}
	if v := s.attr("breakType"); v != nil {
		ab.BreakType = byteStr(v)
	}
	if v := s.attr("timeOffset"); v != nil {
		_ = ab.TimeOffset.UnmarshalText(v)
	}
	s.endAttrs()

	for {
		name, isEnd, selfClose := s.next()
		if name == nil {
			break
		}
		if isEnd {
			if string(name) == "AdBreak" {
				break
			}
			continue
		}
		switch string(name) {
		case "VAST":
			if selfClose {
				ab.AdSource.VASTData.VAST = &VAST{}
				continue
			}
			vast := scanVast(s)
			ab.AdSource.VASTData.VAST = &vast
		case "Tracking":
			if ab.TrackingEvents == nil {
				ab.TrackingEvents = []TrackingEvent{}
			}
			var t TrackingEvent
			if v := s.attr("event"); v != nil {
				t.Event = byteStr(v)
			}
			s.endAttrs()
			t.Text = s.textStr()
			ab.TrackingEvents = append(ab.TrackingEvents, t)
		}
	}
	return ab
}

func scanVast(s *scan) VAST {
	var vast VAST
	if v := s.attr("version"); v != nil {
		vast.Version = byteStr(v)
	}
	s.endAttrs()

	for {
		name, isEnd, _ := s.next()
		if name == nil {
			break
		}
		if isEnd {
			if string(name) == "VAST" {
				break
			}
			continue
		}
		if string(name) == "Ad" {
			vast.Ad = append(vast.Ad, scanAd(s))
		}
	}
	return vast
}

func scanAd(s *scan) Ad {
	var ad Ad
	if v := s.attr("id"); v != nil {
		ad.Id = byteStr(v)
	}
	if v := s.attr("sequence"); v != nil {
		ad.Sequence, _ = strconv.Atoi(byteStr(v))
	}
	s.endAttrs()

	for {
		name, isEnd, _ := s.next()
		if name == nil {
			break
		}
		if isEnd {
			if string(name) == "Ad" {
				break
			}
			continue
		}
		if string(name) == "InLine" {
			inline := scanInLine(s)
			ad.InLine = &inline
		}
	}
	return ad
}

func scanInLine(s *scan) InLine {
	var inline InLine
	s.endAttrs()

	for {
		name, isEnd, _ := s.next()
		if name == nil {
			break
		}
		if isEnd {
			if string(name) == "InLine" {
				break
			}
			continue
		}
		switch string(name) {
		case "Creative":
			inline.Creatives = append(inline.Creatives, scanCreative(s))
		case "Impression":
			var imp Impression
			if v := s.attr("id"); v != nil {
				imp.Id = byteStr(v)
			}
			s.endAttrs()
			imp.Text = s.textStr()
			inline.Impression = append(inline.Impression, imp)
		case "AdSystem":
			s.endAttrs()
			inline.AdSystem = s.textStr()
		case "AdTitle":
			s.endAttrs()
			inline.AdTitle = s.textStr()
		case "Extension":
			inline.Extensions = append(inline.Extensions, scanExtension(s))
		case "Error":
			s.endAttrs()
			inline.Error = &Error{Value: s.textStr()}
		}
	}
	return inline
}

func scanCreative(s *scan) Creative {
	var c Creative
	if v := s.attr("id"); v != nil {
		c.Id = byteStr(v)
	}
	if v := s.attr("adId"); v != nil {
		c.AdId = byteStr(v)
	}
	s.endAttrs()

	for {
		name, isEnd, _ := s.next()
		if name == nil {
			break
		}
		if isEnd {
			if string(name) == "Creative" {
				break
			}
			continue
		}
		switch string(name) {
		case "UniversalAdId":
			var uaid UniversalAdId
			if v := s.attr("idRegistry"); v != nil {
				uaid.IdRegistry = byteStr(v)
			}
			s.endAttrs()
			uaid.Id = s.textStr()
			c.UniversalAdId = &uaid
		case "Tracking":
			if c.Linear == nil {
				c.Linear = &Linear{}
			}
			var t TrackingEvent
			if v := s.attr("event"); v != nil {
				t.Event = byteStr(v)
			}
			s.endAttrs()
			t.Text = s.textStr()
			c.Linear.TrackingEvents = append(c.Linear.TrackingEvents, t)
		case "ClickThrough":
			if c.Linear == nil {
				c.Linear = &Linear{}
			}
			c.Linear.ClickThrough = &ClickThrough{}
			if v := s.attr("id"); v != nil {
				c.Linear.ClickThrough.Id = byteStr(v)
			}
			s.endAttrs()
			c.Linear.ClickThrough.Text = s.textStr()
		case "ClickTracking":
			if c.Linear == nil {
				c.Linear = &Linear{}
			}
			var ct ClickTracking
			if v := s.attr("id"); v != nil {
				ct.Id = byteStr(v)
			}
			s.endAttrs()
			ct.Text = s.textStr()
			c.Linear.ClickTracking = append(c.Linear.ClickTracking, ct)
		case "Duration":
			if c.Linear == nil {
				c.Linear = &Linear{}
			}
			s.endAttrs()
			content, wasCDATA := s.text()
			if content != nil {
				if wasCDATA || bytes.IndexByte(content, '&') < 0 {
					_ = c.Linear.Duration.UnmarshalText(content)
				} else {
					cp := make([]byte, len(content))
					copy(cp, content)
					_ = c.Linear.Duration.UnmarshalText(xmlStringToString(cp))
				}
			}
		case "MediaFile":
			if c.Linear == nil {
				c.Linear = &Linear{}
			}
			var m MediaFile
			if v := s.attr("bitrate"); v != nil {
				m.Bitrate, _ = strconv.Atoi(byteStr(v))
			}
			if v := s.attr("height"); v != nil {
				m.Height, _ = strconv.Atoi(byteStr(v))
			}
			if v := s.attr("width"); v != nil {
				m.Width, _ = strconv.Atoi(byteStr(v))
			}
			if v := s.attr("delivery"); v != nil {
				m.Delivery = byteStr(v)
			}
			if v := s.attr("type"); v != nil {
				m.MediaType = byteStr(v)
			}
			if v := s.attr("codec"); v != nil {
				m.Codec = byteStr(v)
			}
			s.endAttrs()
			m.Text = s.textStr()
			c.Linear.MediaFiles = append(c.Linear.MediaFiles, m)
		}
	}
	return c
}

func scanExtension(s *scan) Extension {
	var ext Extension
	if v := s.attr("type"); v != nil {
		ext.ExtensionType = byteStr(v)
	}
	s.endAttrs()

	for {
		name, isEnd, _ := s.next()
		if name == nil {
			break
		}
		if isEnd {
			if string(name) == "Extension" {
				break
			}
			continue
		}
		if string(name) == "CreativeParameter" {
			var par CreativeParameter
			if v := s.attr("creativeId"); v != nil {
				par.CreativeId = byteStr(v)
			}
			if v := s.attr("name"); v != nil {
				par.Name = byteStr(v)
			}
			if v := s.attr("type"); v != nil {
				par.CreativeParameterType = byteStr(v)
			}
			s.endAttrs()
			par.Value = s.textStr()
			ext.CreativeParameters = append(ext.CreativeParameters, par)
		}
	}
	return ext
}
