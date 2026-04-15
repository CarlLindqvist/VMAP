package vmap

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestUnmarshalVMAP(t *testing.T) {
	is := is.New(t)
	f, err := os.Open("sample-vmap/testVmap.xml")
	is.NoErr(err)
	defer f.Close()

	var vmap VMAP
	xmlBytes, err := io.ReadAll(f)
	is.NoErr(err)

	err = xml.Unmarshal(xmlBytes, &vmap)
	is.NoErr(err)

	is.Equal(len(vmap.AdBreaks), 3)
	firstBreak := vmap.AdBreaks[0]
	is.Equal(firstBreak.Id, "midroll.ad-1")
	is.Equal(firstBreak.BreakType, "linear")
	is.True(firstBreak.TimeOffset.Duration == nil)
	is.Equal(firstBreak.TimeOffset.Position, OffsetStart)
	is.True(firstBreak.AdSource.VASTData.VAST != nil)
	is.Equal(len(firstBreak.TrackingEvents), 1)

	secondBreak := vmap.AdBreaks[1]
	is.Equal(secondBreak.Id, "midroll.ad-2")
	is.Equal(secondBreak.BreakType, "linear")
	is.Equal(*secondBreak.TimeOffset.Duration, Duration{5 * time.Minute})
	is.True(firstBreak.AdSource.VASTData.VAST != nil)
	is.Equal(len(secondBreak.TrackingEvents), 1)

	thirdBreak := vmap.AdBreaks[2]
	is.Equal(thirdBreak.Id, "midroll.ad-3")
	is.Equal(thirdBreak.BreakType, "linear")
	is.Equal(*thirdBreak.TimeOffset.Duration, Duration{7 * time.Minute})
	is.True(thirdBreak.AdSource.VASTData.VAST != nil)
	is.Equal(len(thirdBreak.TrackingEvents), 1)
}

func TestDecodeEmptyVmap(t *testing.T) {
	is := is.New(t)
	doc, err := os.ReadFile("sample-vmap/testVmap2.xml")
	is.NoErr(err)

	vmap, err := DecodeVmap(doc)
	is.NoErr(err)

	is.Equal(len(vmap.AdBreaks), 0)
}

func TestDecodeVmapEmptyVast(t *testing.T) {
	is := is.New(t)
	doc, err := os.ReadFile("sample-vmap/testVmapEmptyVast.xml")
	is.NoErr(err)

	vmap, err := DecodeVmap(doc)
	is.NoErr(err)

	is.Equal(len(vmap.AdBreaks), 1)
	is.Equal(len(vmap.AdBreaks[0].AdSource.VASTData.VAST.Ad), 0)
}

func TestDecodeEmptyVast(t *testing.T) {
	is := is.New(t)
	doc, err := os.ReadFile("sample-vmap/testVast3.xml")
	is.NoErr(err)

	vast, err := DecodeVast(doc)
	is.NoErr(err)

	is.Equal(len(vast.Ad), 0)
}

func TestDecodeVmap(t *testing.T) {
	is := is.New(t)
	f, err := os.Open("sample-vmap/testVmap.xml")
	is.NoErr(err)
	defer f.Close()

	var vmap VMAP
	xmlBytes, err := io.ReadAll(f)
	is.NoErr(err)

	vmap, err = DecodeVmap(xmlBytes)
	is.NoErr(err)

	is.Equal(len(vmap.AdBreaks), 3)
	firstBreak := vmap.AdBreaks[0]
	is.Equal(firstBreak.Id, "midroll.ad-1")
	is.Equal(firstBreak.BreakType, "linear")
	is.True(firstBreak.TimeOffset.Duration == nil)
	is.Equal(firstBreak.TimeOffset.Position, OffsetStart)
	is.True(firstBreak.AdSource.VASTData.VAST != nil)
	is.Equal(len(firstBreak.TrackingEvents), 1)

	secondBreak := vmap.AdBreaks[1]
	is.Equal(secondBreak.Id, "midroll.ad-2")
	is.Equal(secondBreak.BreakType, "linear")
	is.Equal(*secondBreak.TimeOffset.Duration, Duration{5 * time.Minute})
	is.True(firstBreak.AdSource.VASTData.VAST != nil)
	is.Equal(len(secondBreak.TrackingEvents), 1)

	thirdBreak := vmap.AdBreaks[2]
	is.Equal(thirdBreak.Id, "midroll.ad-3")
	is.Equal(thirdBreak.BreakType, "linear")
	is.Equal(*thirdBreak.TimeOffset.Duration, Duration{7 * time.Minute})
	is.True(thirdBreak.AdSource.VASTData.VAST != nil)
	is.Equal(len(thirdBreak.TrackingEvents), 1)
}

func TestUnmarshalVast(t *testing.T) {
	is := is.New(t)
	f, err := os.Open("sample-vmap/testVast.xml")
	is.NoErr(err)
	defer f.Close()

	var vast VAST
	xmlBytes, err := io.ReadAll(f)
	is.NoErr(err)
	err = xml.Unmarshal(xmlBytes, &vast)
	is.NoErr(err)

	is.Equal(len(vast.Ad), 2)
	firstAd := vast.Ad[0]
	is.Equal(firstAd.Id, "POD_AD-ID_001")
	firstAdInLine := firstAd.InLine
	is.Equal(firstAdInLine.AdSystem, "Test Adserver")
	is.Equal(firstAdInLine.AdTitle, "Ad That Test-Adserver Wants Player To See #1")

	// Error validation
	firstAdError := firstAdInLine.Error
	is.True(firstAdError != nil)
	is.Equal(firstAdError.Value, "https://error-url/code")
	// Extension validation
	firstAdExtensions := firstAdInLine.Extensions
	is.Equal(len(firstAdExtensions), 1)
	firstAdExtension := firstAdExtensions[0]
	is.Equal(firstAdExtension.ExtensionType, "FreeWheel")
	firstAdExtensionCParams := firstAdExtension.CreativeParameters[0]
	is.Equal(firstAdExtensionCParams.CreativeId, "132285420")
	is.Equal(firstAdExtensionCParams.Name, "AdType")
	is.Equal(firstAdExtensionCParams.Value, "bumper")
	is.Equal(firstAdExtensionCParams.CreativeParameterType, "Linear")
	// Impression validation
	firstAdImpression := firstAdInLine.Impression
	is.Equal(len(firstAdImpression), 1)
	// Creatives validation
	firstAdCreatives := firstAdInLine.Creatives
	is.Equal(len(firstAdCreatives), 1)
	firstCreative := firstAdCreatives[0]
	is.Equal(firstCreative.Id, "CRETIVE-ID_001")
	is.Equal(firstCreative.AdId, "alvedon-10s")
	is.Equal(len(firstCreative.Linear.TrackingEvents), 5)
	is.Equal(firstCreative.Linear.Duration, Duration{10 * time.Second})
	is.Equal(len(firstCreative.Linear.MediaFiles), 1)
	is.True(firstCreative.Linear.ClickThrough != nil)
	is.Equal(len(firstCreative.Linear.ClickTracking), 0)
	is.Equal(len(firstCreative.Linear.CustomClick), 0)
	// MediaFile validation
	mediaFile := firstCreative.Linear.MediaFiles[0]
	is.Equal(mediaFile.Width, 718)
	is.Equal(mediaFile.Height, 404)
	is.Equal(mediaFile.MediaType, "video/mp4")
	is.Equal(mediaFile.Delivery, "progressive")
	is.Equal(mediaFile.Bitrate, 1300)
	is.Equal(mediaFile.Codec, "H.264")
}

func TestDecodeVast(t *testing.T) {
	is := is.New(t)
	f, err := os.Open("sample-vmap/testVast.xml")
	is.NoErr(err)
	defer f.Close()

	var vast VAST
	xmlBytes, err := io.ReadAll(f)
	is.NoErr(err)
	vast, err = DecodeVast(xmlBytes)
	is.NoErr(err)

	is.Equal(len(vast.Ad), 2)
	firstAd := vast.Ad[0]
	is.Equal(firstAd.Id, "POD_AD-ID_001")
	firstAdInLine := firstAd.InLine
	is.Equal(firstAdInLine.AdSystem, "Test Adserver")
	is.Equal(firstAdInLine.AdTitle, "Ad That Test-Adserver Wants Player To See #1")

	// Error validation
	firstAdError := firstAdInLine.Error
	is.True(firstAdError != nil)
	is.Equal(firstAdError.Value, "https://error-url/code")
	// Extension validation
	firstAdExtensions := firstAdInLine.Extensions
	is.Equal(len(firstAdExtensions), 1)
	firstAdExtension := firstAdExtensions[0]
	is.Equal(firstAdExtension.ExtensionType, "FreeWheel")
	firstAdExtensionCParams := firstAdExtension.CreativeParameters[0]
	is.Equal(firstAdExtensionCParams.CreativeId, "132285420")
	is.Equal(firstAdExtensionCParams.Name, "AdType")
	is.Equal(firstAdExtensionCParams.Value, "bumper")
	is.Equal(firstAdExtensionCParams.CreativeParameterType, "Linear")
	// Impression validation
	firstAdImpression := firstAdInLine.Impression
	is.Equal(len(firstAdImpression), 1)
	// Creatives validation
	firstAdCreatives := firstAdInLine.Creatives
	is.Equal(len(firstAdCreatives), 1)
	firstCreative := firstAdCreatives[0]
	is.Equal(firstCreative.Id, "CRETIVE-ID_001")
	is.Equal(firstCreative.AdId, "alvedon-10s")
	is.Equal(len(firstCreative.Linear.TrackingEvents), 5)
	is.Equal(firstCreative.Linear.Duration, Duration{10 * time.Second})
	is.Equal(len(firstCreative.Linear.MediaFiles), 1)
	is.True(firstCreative.Linear.ClickThrough != nil)
	is.Equal(len(firstCreative.Linear.ClickTracking), 0)
	is.Equal(len(firstCreative.Linear.CustomClick), 0)
	// MediaFile validation
	mediaFile := firstCreative.Linear.MediaFiles[0]
	is.Equal(mediaFile.Width, 718)
	is.Equal(mediaFile.Height, 404)
	is.Equal(mediaFile.MediaType, "video/mp4")
	is.Equal(mediaFile.Delivery, "progressive")
	is.Equal(mediaFile.Bitrate, 1300)
	is.Equal(mediaFile.Codec, "H.264")
}

func TestUnmarshalDuration(t *testing.T) {
	is := is.New(t)
	d := Duration{}
	err := d.UnmarshalText([]byte("00:00:10"))

	is.NoErr(err)
	is.Equal(d.Duration, 10*time.Second)
	err = d.UnmarshalText([]byte("00:01:00"))
	is.Equal(d.Duration, 1*time.Minute)
	is.NoErr(err)

	err = d.UnmarshalText([]byte("00:01:00.300"))
	is.NoErr(err)
	is.Equal(d.Duration, 1*time.Minute+300*time.Millisecond)

	err = d.UnmarshalText([]byte("04:01:12.345"))
	is.NoErr(err)
	is.Equal(d.Duration, 4*time.Hour+1*time.Minute+12*time.Second+345*time.Millisecond)

	err = d.UnmarshalText([]byte("01:04:01:12.345"))
	is.True(err != nil)

	err = d.UnmarshalText([]byte("01:04"))
	is.True(err != nil)
}

func TestMarshalJson(t *testing.T) {
	is := is.New(t)
	f, err := os.Open("sample-vmap/testVmap.xml")
	is.NoErr(err)
	defer f.Close()

	var vmap VMAP
	xmlBytes, err := io.ReadAll(f)
	is.NoErr(err)
	err = xml.Unmarshal(xmlBytes, &vmap)
	is.NoErr(err)
	jsonBytes, err := json.Marshal(vmap)
	is.NoErr(err)
	is.True(json.Valid(jsonBytes))

	var vmap2 VMAP
	err = json.Unmarshal(jsonBytes, &vmap2)
	is.NoErr(err)
	is.Equal(vmap, vmap2)
}

func BenchmarkUnmarshal(b *testing.B) {
	doc, err := os.ReadFile("sample-vmap/testVmap.xml")
	if err != nil {
		panic(err)
	}

	var vmap VMAP

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = xml.Unmarshal(doc, &vmap)
	}
}

func BenchmarkFasterDecode(b *testing.B) {
	doc, err := os.ReadFile("sample-vmap/testVmap.xml")
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeVmap(doc)
	}
}

func BenchmarkScanDecode(b *testing.B) {
	doc, err := os.ReadFile("sample-vmap/testVmap.xml")
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeVmapScan(doc)
	}
}

func TestDecodeVmapScan(t *testing.T) {
	is := is.New(t)
	doc, err := os.ReadFile("sample-vmap/testVmap.xml")
	is.NoErr(err)

	vmap1, err := DecodeVmap(doc)
	is.NoErr(err)
	vmap2, err := DecodeVmapScan(doc)
	is.NoErr(err)

	is.Equal(vmap1.Version, vmap2.Version)
	is.Equal(vmap1.Vmap, vmap2.Vmap)
	is.Equal(vmap1.XMLName.Local, vmap2.XMLName.Local)
	is.Equal(vmap1.XMLName.Space, vmap2.XMLName.Space)

	is.Equal(len(vmap1.AdBreaks), len(vmap2.AdBreaks))
	for i := range vmap1.AdBreaks {
		a := vmap1.AdBreaks[i]
		b := vmap2.AdBreaks[i]
		is.Equal(a.Id, b.Id)
		is.Equal(a.BreakType, b.BreakType)
		is.Equal(a.TimeOffset, b.TimeOffset)
		is.Equal(len(a.TrackingEvents), len(b.TrackingEvents))
		for j := range a.TrackingEvents {
			is.Equal(strings.TrimSpace(a.TrackingEvents[j].Text), strings.TrimSpace(b.TrackingEvents[j].Text))
			is.Equal(a.TrackingEvents[j].Event, b.TrackingEvents[j].Event)
		}

		v1 := a.AdSource.VASTData.VAST
		v2 := b.AdSource.VASTData.VAST
		is.True(v1 != nil)
		is.True(v2 != nil)
		is.Equal(v1.Version, v2.Version)
		is.Equal(len(v1.Ad), len(v2.Ad))
		for j := range v1.Ad {
			is.Equal(v1.Ad[j].Id, v2.Ad[j].Id)
			is.Equal(v1.Ad[j].Sequence, v2.Ad[j].Sequence)
			if v1.Ad[j].InLine != nil {
				is.True(v2.Ad[j].InLine != nil)
				is.Equal(strings.TrimSpace(v1.Ad[j].InLine.AdSystem), strings.TrimSpace(v2.Ad[j].InLine.AdSystem))
				is.Equal(strings.TrimSpace(v1.Ad[j].InLine.AdTitle), strings.TrimSpace(v2.Ad[j].InLine.AdTitle))
				is.Equal(v1.Ad[j].InLine.Error, v2.Ad[j].InLine.Error)
				is.Equal(len(v1.Ad[j].InLine.Creatives), len(v2.Ad[j].InLine.Creatives))
			}
		}
	}
}

func TestDecodeVastScan(t *testing.T) {
	is := is.New(t)
	doc, err := os.ReadFile("sample-vmap/testVast.xml")
	is.NoErr(err)

	vast1, err := DecodeVast(doc)
	is.NoErr(err)
	vast2, err := DecodeVastScan(doc)
	is.NoErr(err)

	is.Equal(vast1.Version, vast2.Version)
	is.Equal(len(vast1.Ad), len(vast2.Ad))
	for i := range vast1.Ad {
		a := vast1.Ad[i]
		b := vast2.Ad[i]
		is.Equal(a.Id, b.Id)
		is.Equal(a.Sequence, b.Sequence)
		if a.InLine != nil {
			is.True(b.InLine != nil)
			is.Equal(strings.TrimSpace(a.InLine.AdSystem), strings.TrimSpace(b.InLine.AdSystem))
			is.Equal(strings.TrimSpace(a.InLine.AdTitle), strings.TrimSpace(b.InLine.AdTitle))
			is.Equal(a.InLine.Error, b.InLine.Error)
			is.Equal(len(a.InLine.Impression), len(b.InLine.Impression))
			is.Equal(len(a.InLine.Creatives), len(b.InLine.Creatives))
			for j := range a.InLine.Creatives {
				c1 := a.InLine.Creatives[j]
				c2 := b.InLine.Creatives[j]
				is.Equal(c1.Id, c2.Id)
				is.Equal(c1.AdId, c2.AdId)
				is.Equal(c1.Linear.Duration, c2.Linear.Duration)
				is.Equal(len(c1.Linear.TrackingEvents), len(c2.Linear.TrackingEvents))
				is.Equal(len(c1.Linear.MediaFiles), len(c2.Linear.MediaFiles))
				for k := range c1.Linear.MediaFiles {
					is.Equal(c1.Linear.MediaFiles[k].Width, c2.Linear.MediaFiles[k].Width)
					is.Equal(c1.Linear.MediaFiles[k].Height, c2.Linear.MediaFiles[k].Height)
					is.Equal(c1.Linear.MediaFiles[k].Bitrate, c2.Linear.MediaFiles[k].Bitrate)
					is.Equal(c1.Linear.MediaFiles[k].MediaType, c2.Linear.MediaFiles[k].MediaType)
					is.Equal(c1.Linear.MediaFiles[k].Codec, c2.Linear.MediaFiles[k].Codec)
				}
			}
			is.Equal(len(a.InLine.Extensions), len(b.InLine.Extensions))
		}
	}
}

func TestSpecialCharactersScan(t *testing.T) {
	is := is.New(t)
	doc, err := os.ReadFile("sample-vmap/testVastSpecialChars.xml")
	is.NoErr(err)

	vastDecoded, err := DecodeVast(doc)
	is.NoErr(err)
	vastScanned, err := DecodeVastScan(doc)
	is.NoErr(err)

	is.Equal(vastDecoded.Ad[0].InLine.AdTitle, vastScanned.Ad[0].InLine.AdTitle)
	is.Equal(vastScanned.Ad[0].InLine.AdTitle, "Hej&ö\n<>\"")
}

func TestSpecialCharacters(t *testing.T) {
	is := is.New(t)
	doc, err := os.ReadFile("sample-vmap/testVastSpecialChars.xml")
	if err != nil {
		panic(err)
	}
	var vastUnmarshal VAST
	_ = xml.Unmarshal(doc, &vastUnmarshal)
	vastDecoded, _ := DecodeVast(doc)

	is.Equal(vastUnmarshal.Ad[0].InLine.AdTitle, vastDecoded.Ad[0].InLine.AdTitle)
	is.Equal(vastDecoded.Ad[0].InLine.AdTitle, "Hej&ö\n<>\"")
}

func TestDecodeCompliance(t *testing.T) {
	wg := sync.WaitGroup{}
	//Check for race conditions
	for range 1000 {
		wg.Add(1)
		go func(wg *sync.WaitGroup, t *testing.T) {
			defer wg.Done()
			is := is.New(t)
			doc, err := os.ReadFile("sample-vmap/testVmap.xml")
			is.NoErr(err)

			var vmap1 VMAP
			err = xml.Unmarshal(doc, &vmap1)
			is.NoErr(err)

			vmap2, err := DecodeVmap(doc)
			is.NoErr(err)

			is.Equal(vmap1.Version, vmap2.Version)
			is.Equal(vmap1.Vmap, vmap2.Vmap)
			is.Equal(vmap1.XMLName.Local, vmap2.XMLName.Local)
			is.Equal(vmap1.XMLName.Space, vmap2.XMLName.Space)

			is.Equal(len(vmap1.AdBreaks), len(vmap2.AdBreaks))
			for i := range vmap1.AdBreaks {
				adb1 := vmap1.AdBreaks[i]
				adb2 := vmap2.AdBreaks[i]
				is.Equal(adb1.BreakType, adb2.BreakType)
				is.Equal(adb1.Id, adb2.Id)
				is.Equal(adb1.TimeOffset, adb2.TimeOffset)
				is.Equal(adb1.TimeOffset.Duration, adb2.TimeOffset.Duration)
				is.Equal(adb1.TimeOffset.Position, adb2.TimeOffset.Position)

				if adb1.TrackingEvents != nil {
					te1 := adb1.TrackingEvents
					te2 := adb2.TrackingEvents

					for j := range te1 {
						abt1 := te1[j]
						abt2 := te2[j]
						is.Equal(abt1.Event, abt2.Event)
						//Decode trims spaces, so not checking whitespace
						is.Equal(strings.TrimSpace(abt1.Text), strings.TrimSpace(abt2.Text))
					}
				}

				is.True(adb1.AdSource.VASTData.VAST != nil)
				is.True(adb2.AdSource.VASTData.VAST != nil)
				v1 := *adb1.AdSource.VASTData.VAST
				v2 := *adb2.AdSource.VASTData.VAST
				is.Equal(v1.Version, v2.Version)
				is.Equal(v1.NoNamespaceSchemaLocation, v2.NoNamespaceSchemaLocation)
				is.Equal(v1.Xsi, v2.Xsi)

				for j := range v1.Ad {
					ad1 := v1.Ad[j]
					ad2 := v2.Ad[j]
					is.Equal(ad1.Id, ad2.Id)
					is.Equal(ad1.Sequence, ad2.Sequence)
					if ad1.InLine != nil {
						is.Equal(strings.TrimSpace(ad1.InLine.AdSystem), strings.TrimSpace(ad2.InLine.AdSystem))
						is.Equal(strings.TrimSpace(ad1.InLine.AdTitle), strings.TrimSpace(ad2.InLine.AdTitle))
						is.Equal(ad1.InLine.Error, ad2.InLine.Error)
						if ad1.InLine.Error != nil {
							is.Equal(ad1.InLine.Error.Value, ad2.InLine.Error.Value)
						}
						if ad1.InLine.Creatives != nil {
							for i := range ad1.InLine.Creatives {
								for j := range ad1.InLine.Creatives[i].Linear.TrackingEvents {
									is.Equal(
										strings.TrimSpace(ad1.InLine.Creatives[i].Linear.TrackingEvents[j].Text),
										strings.TrimSpace(ad2.InLine.Creatives[i].Linear.TrackingEvents[j].Text),
									)
								}
								for j := range ad1.InLine.Creatives[i].Linear.ClickTracking {
									fmt.Println(strings.TrimSpace(ad1.InLine.Creatives[i].Linear.ClickTracking[j].Text))
									is.Equal(
										strings.TrimSpace(ad1.InLine.Creatives[i].Linear.ClickTracking[j].Text),
										strings.TrimSpace(ad2.InLine.Creatives[i].Linear.ClickTracking[j].Text),
									)
								}
							}

						}
					}
				}
			}
		}(&wg, t)
	}
	wg.Wait()
}
