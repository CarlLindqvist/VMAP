package vmap

import (
	"strconv"
)

// MarshalVmap marshals a VMAP to XML, producing output identical to encoding/xml.Marshal.
func MarshalVmap(v *VMAP) ([]byte, error) {
	buf := make([]byte, 0, 8192)
	buf = appendVMAP(buf, v)
	return buf, nil
}

// MarshalVast marshals a VAST to XML, producing output identical to encoding/xml.Marshal.
func MarshalVast(v *VAST) ([]byte, error) {
	buf := make([]byte, 0, 4096)
	buf = appendVAST(buf, v)
	return buf, nil
}

// --- escape helpers ---

// escText escapes text content, matching encoding/xml.EscapeText.
func escText(buf []byte, s string) []byte {
	last := 0
	for i := 0; i < len(s); i++ {
		var esc string
		switch s[i] {
		case '&':
			esc = "&amp;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		case '"':
			esc = "&#34;"
		case '\t':
			esc = "&#x9;"
		case '\n':
			esc = "&#xA;"
		case '\r':
			esc = "&#xD;"
		default:
			continue
		}
		buf = append(buf, s[last:i]...)
		buf = append(buf, esc...)
		last = i + 1
	}
	return append(buf, s[last:]...)
}

// escAttr escapes attribute values, matching encoding/xml attribute escaping.
func escAttr(buf []byte, s string) []byte {
	last := 0
	for i := 0; i < len(s); i++ {
		var esc string
		switch s[i] {
		case '&':
			esc = "&amp;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		case '"':
			esc = "&#34;"
		case '\t':
			esc = "&#x9;"
		case '\n':
			esc = "&#xA;"
		case '\r':
			esc = "&#xD;"
		default:
			continue
		}
		buf = append(buf, s[last:i]...)
		buf = append(buf, esc...)
		last = i + 1
	}
	return append(buf, s[last:]...)
}

// --- duration / time offset helpers (allocation-free) ---

func append2dig(buf []byte, n int) []byte {
	return append(buf, byte('0'+n/10), byte('0'+n%10))
}

func append3dig(buf []byte, n int) []byte {
	return append(buf, byte('0'+n/100), byte('0'+(n/10)%10), byte('0'+n%10))
}

func appendDuration(buf []byte, d Duration) []byte {
	dur := d.Duration
	if dur == 0 {
		return append(buf, "00:00:00"...)
	}
	h := int(dur.Hours())
	m := int(dur.Minutes()) % 60
	s := int(dur.Seconds()) % 60
	ms := int(dur.Milliseconds()) % 1000
	buf = append2dig(buf, h)
	buf = append(buf, ':')
	buf = append2dig(buf, m)
	buf = append(buf, ':')
	buf = append2dig(buf, s)
	if ms > 0 {
		buf = append(buf, '.')
		buf = append3dig(buf, ms)
	}
	return buf
}

func appendTimeOffset(buf []byte, to TimeOffset) []byte {
	if to.Duration != nil {
		return appendDuration(buf, *to.Duration)
	}
	if to.Position != 0 {
		buf = append(buf, '#')
		return strconv.AppendInt(buf, int64(to.Position), 10)
	}
	if to.Percent != 0 {
		buf = strconv.AppendFloat(buf, float64(to.Percent*100), 'f', 6, 32)
		return append(buf, '%')
	}
	return buf
}

// --- struct encoders ---
// Field and attribute order matches encoding/xml.Marshal exactly.

func appendVMAP(buf []byte, v *VMAP) []byte {
	// XMLName tag is xml:"VMAP" (name only) — xml.Marshal does not output xmlns
	buf = append(buf, `<VMAP vmap="`...)
	buf = escAttr(buf, v.Vmap)
	buf = append(buf, `" version="`...)
	buf = escAttr(buf, v.Version)
	buf = append(buf, '"', '>')

	// chardata (Text field, before child elements, matching xml.Marshal field order)
	buf = escText(buf, v.Text)

	for i := range v.AdBreaks {
		buf = appendAdBreak(buf, &v.AdBreaks[i])
	}
	buf = append(buf, "</VMAP>"...)
	return buf
}

func appendAdBreak(buf []byte, ab *AdBreak) []byte {
	// attrs: breakId, breakType, timeOffset
	buf = append(buf, `<AdBreak breakId="`...)
	buf = escAttr(buf, ab.Id)
	buf = append(buf, `" breakType="`...)
	buf = escAttr(buf, ab.BreakType)
	buf = append(buf, `" timeOffset="`...)
	buf = appendTimeOffset(buf, ab.TimeOffset)
	buf = append(buf, '"', '>')

	// child elements in field order: AdSource, TrackingEvents
	if ab.AdSource != nil {
		buf = appendAdSource(buf, ab.AdSource)
	}
	// Wrapper always emitted for nested path xml:"TrackingEvents>Tracking"
	buf = append(buf, "<TrackingEvents>"...)
	for i := range ab.TrackingEvents {
		buf = appendTracking(buf, &ab.TrackingEvents[i])
	}
	buf = append(buf, "</TrackingEvents>"...)
	buf = append(buf, "</AdBreak>"...)
	return buf
}

func appendAdSource(buf []byte, as *AdSource) []byte {
	buf = append(buf, "<AdSource>"...)
	if as.VASTData != nil {
		buf = append(buf, "<VASTAdData>"...)
		if as.VASTData.VAST != nil {
			buf = appendVAST(buf, as.VASTData.VAST)
		}
		buf = append(buf, "</VASTAdData>"...)
	}
	buf = append(buf, "</AdSource>"...)
	return buf
}

func appendVAST(buf []byte, v *VAST) []byte {
	// attrs: xsi, noNamespaceSchemaLocation, version
	buf = append(buf, `<VAST xsi="`...)
	buf = escAttr(buf, v.Xsi)
	buf = append(buf, `" noNamespaceSchemaLocation="`...)
	buf = escAttr(buf, v.NoNamespaceSchemaLocation)
	buf = append(buf, `" version="`...)
	buf = escAttr(buf, v.Version)
	buf = append(buf, '"', '>')

	// chardata
	buf = escText(buf, v.Text)

	for i := range v.Ad {
		buf = appendAd(buf, &v.Ad[i])
	}
	buf = append(buf, "</VAST>"...)
	return buf
}

func appendAd(buf []byte, ad *Ad) []byte {
	buf = append(buf, `<Ad id="`...)
	buf = escAttr(buf, ad.Id)
	buf = append(buf, `" sequence="`...)
	buf = strconv.AppendInt(buf, int64(ad.Sequence), 10)
	buf = append(buf, '"', '>')

	if ad.InLine != nil {
		buf = appendInLine(buf, ad.InLine)
	}
	buf = append(buf, "</Ad>"...)
	return buf
}

func appendInLine(buf []byte, il *InLine) []byte {
	buf = append(buf, "<InLine>"...)

	// field order: AdSystem, AdTitle, Impression, Creatives, Extensions, Error
	buf = append(buf, "<AdSystem>"...)
	buf = escText(buf, il.AdSystem)
	buf = append(buf, "</AdSystem>"...)

	buf = append(buf, "<AdTitle>"...)
	buf = escText(buf, il.AdTitle)
	buf = append(buf, "</AdTitle>"...)

	for i := range il.Impression {
		buf = appendImpression(buf, &il.Impression[i])
	}

	// Wrappers always emitted for nested paths
	buf = append(buf, "<Creatives>"...)
	for i := range il.Creatives {
		buf = appendCreative(buf, &il.Creatives[i])
	}
	buf = append(buf, "</Creatives>"...)

	buf = append(buf, "<Extensions>"...)
	for i := range il.Extensions {
		buf = appendExtension(buf, &il.Extensions[i])
	}
	buf = append(buf, "</Extensions>"...)

	if il.Error != nil {
		buf = append(buf, "<Error>"...)
		buf = escText(buf, il.Error.Value)
		buf = append(buf, "</Error>"...)
	}

	buf = append(buf, "</InLine>"...)
	return buf
}

func appendImpression(buf []byte, imp *Impression) []byte {
	buf = append(buf, `<Impression id="`...)
	buf = escAttr(buf, imp.Id)
	buf = append(buf, '"', '>')
	buf = escText(buf, imp.Text)
	buf = append(buf, "</Impression>"...)
	return buf
}

func appendCreative(buf []byte, c *Creative) []byte {
	buf = append(buf, `<Creative id="`...)
	buf = escAttr(buf, c.Id)
	buf = append(buf, `" adId="`...)
	buf = escAttr(buf, c.AdId)
	buf = append(buf, '"', '>')

	if c.UniversalAdId != nil {
		buf = append(buf, `<UniversalAdId idRegistry="`...)
		buf = escAttr(buf, c.UniversalAdId.IdRegistry)
		buf = append(buf, '"', '>')
		buf = escText(buf, c.UniversalAdId.Id)
		buf = append(buf, "</UniversalAdId>"...)
	}

	if c.Linear != nil {
		buf = appendLinear(buf, c.Linear)
	}

	buf = append(buf, "</Creative>"...)
	return buf
}

func appendLinear(buf []byte, l *Linear) []byte {
	buf = append(buf, "<Linear>"...)

	// Duration
	buf = append(buf, "<Duration>"...)
	buf = appendDuration(buf, l.Duration)
	buf = append(buf, "</Duration>"...)

	// Wrappers always emitted for nested paths
	buf = append(buf, "<TrackingEvents>"...)
	for i := range l.TrackingEvents {
		buf = appendTracking(buf, &l.TrackingEvents[i])
	}
	buf = append(buf, "</TrackingEvents>"...)

	buf = append(buf, "<MediaFiles>"...)
	for i := range l.MediaFiles {
		buf = appendMediaFile(buf, &l.MediaFiles[i])
	}
	buf = append(buf, "</MediaFiles>"...)

	// VideoClicks (shared wrapper for ClickThrough, ClickTracking, CustomClick)
	buf = append(buf, "<VideoClicks>"...)
	if l.ClickThrough != nil {
		buf = append(buf, `<ClickThrough id="`...)
		buf = escAttr(buf, l.ClickThrough.Id)
		buf = append(buf, '"', '>')
		buf = escText(buf, l.ClickThrough.Text)
		buf = append(buf, "</ClickThrough>"...)
	}
	for i := range l.ClickTracking {
		buf = append(buf, `<ClickTracking id="`...)
		buf = escAttr(buf, l.ClickTracking[i].Id)
		buf = append(buf, '"', '>')
		buf = escText(buf, l.ClickTracking[i].Text)
		buf = append(buf, "</ClickTracking>"...)
	}
	for i := range l.CustomClick {
		buf = append(buf, `<CustomClick id="`...)
		buf = escAttr(buf, l.CustomClick[i].Id)
		buf = append(buf, '"', '>')
		buf = escText(buf, l.CustomClick[i].Text)
		buf = append(buf, "</CustomClick>"...)
	}
	buf = append(buf, "</VideoClicks>"...)

	buf = append(buf, "</Linear>"...)
	return buf
}

func appendTracking(buf []byte, t *TrackingEvent) []byte {
	buf = append(buf, `<Tracking event="`...)
	buf = escAttr(buf, t.Event)
	buf = append(buf, '"', '>')
	buf = escText(buf, t.Text)
	buf = append(buf, "</Tracking>"...)
	return buf
}

func appendMediaFile(buf []byte, m *MediaFile) []byte {
	// attr order: bitrate, width, height, delivery, type, codec
	buf = append(buf, `<MediaFile bitrate="`...)
	buf = strconv.AppendInt(buf, int64(m.Bitrate), 10)
	buf = append(buf, `" width="`...)
	buf = strconv.AppendInt(buf, int64(m.Width), 10)
	buf = append(buf, `" height="`...)
	buf = strconv.AppendInt(buf, int64(m.Height), 10)
	buf = append(buf, `" delivery="`...)
	buf = escAttr(buf, m.Delivery)
	buf = append(buf, `" type="`...)
	buf = escAttr(buf, m.MediaType)
	buf = append(buf, `" codec="`...)
	buf = escAttr(buf, m.Codec)
	buf = append(buf, '"', '>')
	buf = escText(buf, m.Text)
	buf = append(buf, "</MediaFile>"...)
	return buf
}

func appendExtension(buf []byte, ext *Extension) []byte {
	buf = append(buf, `<Extension type="`...)
	buf = escAttr(buf, ext.ExtensionType)
	buf = append(buf, '"', '>')

	buf = append(buf, "<CreativeParameters>"...)
	for i := range ext.CreativeParameters {
		buf = appendCreativeParameter(buf, &ext.CreativeParameters[i])
	}
	buf = append(buf, "</CreativeParameters>"...)

	buf = append(buf, "</Extension>"...)
	return buf
}

func appendCreativeParameter(buf []byte, cp *CreativeParameter) []byte {
	// attr order: creativeId, name, type (Value is chardata)
	buf = append(buf, `<CreativeParameter creativeId="`...)
	buf = escAttr(buf, cp.CreativeId)
	buf = append(buf, `" name="`...)
	buf = escAttr(buf, cp.Name)
	buf = append(buf, `" type="`...)
	buf = escAttr(buf, cp.CreativeParameterType)
	buf = append(buf, '"', '>')
	buf = escText(buf, cp.Value)
	buf = append(buf, "</CreativeParameter>"...)
	return buf
}
