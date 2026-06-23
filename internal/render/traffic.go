// Package render turns aggregated data into PNG images for the bot to send as
// photos. Pure rendering: input is plain data, output is image bytes.
package render

import (
	"bytes"
	"fmt"
	"image/color"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/goregular"

	"xui-tg-admin/internal/helpers"
)

// Layout (logical pixels, designed at ~2x for crisp display in Telegram).
const (
	width   = 1080
	marginX = 48
	marginY = 44

	headerH    = 116
	rowH       = 132
	sectionH   = 70 // "By inbound" sub-header height
	inbRowH    = 96 // compact per-inbound row
	footerGapH = 24 // space above the total divider
	footerH    = 96 // total block height
	legendH    = 64 // colour legend at the very bottom

	dotRadius = 9
	nameX     = marginX + 42

	barH = 14

	inbSwatch = 18 // side of the inbound colour swatch
	inbBarH   = 12

	nearExpiryDays = 7
)

// Palette — GitHub-dark inspired.
const (
	colBG     = "#0D1117"
	colText   = "#E6EDF3"
	colDim    = "#7D8590"
	colGreen  = "#3FB950"
	colGrey   = "#6E7681"
	colRed    = "#F85149"
	colAmber  = "#D29922"
	colTrack  = "#21262D"
	colRule   = "#272D36" // hairline dividers under headers
	colBarA   = "#2F6FEB" // bar gradient start
	colBarB   = "#4D9DF7" // bar gradient end (subtle same-hue sheen)
	colAccent = "#4D9DF7" // inbound swatch
)

// hexColor parses a "#RRGGBB" string into an opaque color. All callers pass
// compile-time constants, so the fallback to black is only a safety net.
func hexColor(s string) color.Color {
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return color.Black
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// face builds a font face of the given size from embedded TTF bytes.
func face(ttf []byte, size float64) font.Face {
	f, err := truetype.Parse(ttf)
	if err != nil {
		// gofont bytes are compiled in and always valid; this can't fail in practice.
		panic(fmt.Sprintf("render: parse font: %v", err))
	}
	return truetype.NewFace(f, &truetype.Options{Size: size})
}

// TrafficReport renders the aggregated traffic report as a PNG.
func TrafficReport(report helpers.TrafficReport, generatedAt time.Time) ([]byte, error) {
	height := marginY*2 + headerH + len(report.Users)*rowH + footerGapH + footerH + legendH
	if len(report.Inbounds) > 0 {
		height += sectionH + len(report.Inbounds)*inbRowH
	}

	dc := gg.NewContext(width, height)
	dc.SetHexColor(colBG)
	dc.Clear()

	var (
		fontTitle = face(gobold.TTF, 44)
		fontName  = face(gomedium.TTF, 34)
		fontNum   = face(gobold.TTF, 36)
		fontSub   = face(goregular.TTF, 26)
		fontLabel = face(gomedium.TTF, 24)
		fontFoot  = face(gobold.TTF, 40)
	)

	// Largest totals drive proportional bar lengths within each section.
	var maxUser int64 = 1
	for _, u := range report.Users {
		if u.Total() > maxUser {
			maxUser = u.Total()
		}
	}
	var maxInbound int64 = 1
	for _, in := range report.Inbounds {
		if in.Total() > maxInbound {
			maxInbound = in.Total()
		}
	}

	// ---- Header ----
	dc.SetFontFace(fontTitle)
	dc.SetHexColor(colText)
	dc.DrawString("Traffic usage", marginX, marginY+44)

	dc.SetFontFace(fontSub)
	dc.SetHexColor(colDim)
	subtitle := fmt.Sprintf("since last reset  ·  %d users  ·  %d online  ·  %s",
		len(report.Users), report.OnlineCount, generatedAt.Format("02 Jan 2006, 15:04"))
	dc.DrawString(subtitle, marginX, marginY+82)

	// Hairline under the header.
	hairline(dc, float64(marginY+headerH)-20)

	y := float64(marginY + headerH)

	// ---- User rows ----
	for _, u := range report.Users {
		drawUserRow(dc, u, y, maxUser, generatedAt, fontName, fontNum, fontSub)
		y += rowH
	}

	// ---- By-inbound section ----
	if len(report.Inbounds) > 0 {
		drawSectionHeader(dc, "BY INBOUND", y, fontLabel)
		y += sectionH
		for _, in := range report.Inbounds {
			drawInboundRow(dc, in, y, maxInbound, fontName, fontNum)
			y += inbRowH
		}
	}

	// ---- Footer total ----
	y += footerGapH
	dc.SetHexColor(colTrack)
	dc.DrawRectangle(marginX, y, width-2*marginX, 2)
	dc.Fill()

	dc.SetFontFace(fontFoot)
	dc.SetHexColor(colText)
	dc.DrawString("Total", marginX, y+62)

	dc.SetFontFace(fontSub)
	dc.SetHexColor(colDim)
	totalStr := fmt.Sprintf("downloaded %s   ·   uploaded %s",
		helpers.FormatBytes(report.TotalDown), helpers.FormatBytes(report.TotalUp))
	dc.DrawStringAnchored(totalStr, width-marginX, y+44, 1, 0.5)

	// ---- Legend ----
	drawLegend(dc, y+float64(footerH)+24, fontLabel)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("render traffic report: %w", err)
	}
	return buf.Bytes(), nil
}

func drawLegend(dc *gg.Context, cy float64, fontLabel font.Face) {
	items := []struct {
		color, label string
	}{
		{colGreen, "online"},
		{colGrey, "offline"},
		{colAmber, "expiring soon"},
		{colRed, "disabled"},
	}

	dc.SetFontFace(fontLabel)
	const dotR = 8.0
	const gapDotText = 16.0
	const gapItems = 40.0

	// Measure total width to centre the legend.
	total := 0.0
	for i, it := range items {
		w, _ := dc.MeasureString(it.label)
		total += dotR*2 + gapDotText + w
		if i < len(items)-1 {
			total += gapItems
		}
	}

	x := (float64(width) - total) / 2
	for _, it := range items {
		dc.SetHexColor(it.color)
		dc.DrawCircle(x+dotR, cy, dotR)
		dc.Fill()
		x += dotR * 2

		dc.SetHexColor(colDim)
		dc.DrawStringAnchored(it.label, x+gapDotText, cy, 0, 0.5)
		w, _ := dc.MeasureString(it.label)
		x += gapDotText + w + gapItems
	}
}

func drawUserRow(
	dc *gg.Context,
	u helpers.UserTraffic,
	y float64,
	maxTotal int64,
	now time.Time,
	fontName, fontNum, fontSub font.Face,
) {
	line1 := y + 24
	line2 := y + 64

	// Status dot.
	dc.SetHexColor(statusColor(u, now))
	dc.DrawCircle(marginX+dotRadius+4, line1, dotRadius)
	dc.Fill()

	// Name.
	dc.SetFontFace(fontName)
	dc.SetHexColor(nameColor(u))
	dc.DrawStringAnchored(u.Name, nameX, line1, 0, 0.5)

	// Total (big, right-aligned).
	dc.SetFontFace(fontNum)
	dc.SetHexColor(colText)
	dc.DrawStringAnchored(helpers.FormatBytes(u.Total()), width-marginX, line1, 1, 0.5)

	// Secondary line: down/up on the left, expiry on the right.
	dc.SetFontFace(fontSub)
	dc.SetHexColor(colDim)
	dc.DrawStringAnchored(
		fmt.Sprintf("↓ %s    ↑ %s", helpers.FormatBytes(u.Down), helpers.FormatBytes(u.Up)),
		nameX, line2, 0, 0.5)
	dc.SetHexColor(expiryColor(u, now))
	dc.DrawStringAnchored(expiryLabel(u), width-marginX, line2, 1, 0.5)

	// Proportional usage bar spanning the row width, blue gradient.
	drawBar(dc, float64(marginX), y+float64(rowH)-30, float64(width-2*marginX), barH,
		float64(u.Total())/float64(maxTotal), gradientFill)
}

func drawInboundRow(
	dc *gg.Context,
	in helpers.InboundTraffic,
	y float64,
	maxTotal int64,
	fontName, fontNum font.Face,
) {
	line1 := y + 24

	// Small accent swatch (uniform — no rainbow).
	dc.SetHexColor(colAccent)
	dc.DrawRoundedRectangle(marginX, line1-float64(inbSwatch)/2, inbSwatch, inbSwatch, 4)
	dc.Fill()

	// Name + total.
	dc.SetFontFace(fontName)
	dc.SetHexColor(colText)
	dc.DrawStringAnchored(in.Name, marginX+inbSwatch+18, line1, 0, 0.5)

	dc.SetFontFace(fontNum)
	dc.DrawStringAnchored(helpers.FormatBytes(in.Total()), width-marginX, line1, 1, 0.5)

	// Same blue bar as the user rows, for a cohesive look.
	drawBar(dc, float64(marginX), y+float64(inbRowH)-34, float64(width-2*marginX), inbBarH,
		float64(in.Total())/float64(maxTotal), gradientFill)
}

// hairline draws a full-width 2px divider rule.
func hairline(dc *gg.Context, y float64) {
	dc.SetHexColor(colRule)
	dc.DrawRectangle(marginX, y, width-2*marginX, 2)
	dc.Fill()
}

// drawSectionHeader draws a dim uppercase label followed by a rule filling the
// rest of the width.
func drawSectionHeader(dc *gg.Context, label string, y float64, fontLabel font.Face) {
	cy := y + 36
	dc.SetFontFace(fontLabel)
	dc.SetHexColor(colDim)
	dc.DrawStringAnchored(label, marginX, cy, 0, 0.5)
	w, _ := dc.MeasureString(label)
	ruleX := marginX + w + 24
	dc.SetHexColor(colRule)
	dc.DrawRectangle(ruleX, cy-1, float64(width-marginX)-ruleX, 2)
	dc.Fill()
}

// barFiller paints the filled portion of a bar over [x, x+w].
type barFiller func(dc *gg.Context, x, w float64)

func gradientFill(dc *gg.Context, x, w float64) {
	grad := gg.NewLinearGradient(x, 0, x+w, 0)
	grad.AddColorStop(0, hexColor(colBarA))
	grad.AddColorStop(1, hexColor(colBarB))
	dc.SetFillStyle(grad)
}

func drawBar(dc *gg.Context, x, y, w, h, frac float64, fill barFiller) {
	radius := h / 2
	dc.SetHexColor(colTrack)
	dc.DrawRoundedRectangle(x, y, w, h, radius)
	dc.Fill()
	if frac <= 0 {
		return
	}
	fillW := w * frac
	if fillW < h { // keep tiny bars visible and round-cap friendly
		fillW = h
	}
	fill(dc, x, w)
	dc.DrawRoundedRectangle(x, y, fillW, h, radius)
	dc.Fill()
}

func statusColor(u helpers.UserTraffic, now time.Time) string {
	switch {
	case !u.Enabled:
		return colRed
	case u.Online:
		return colGreen
	case isNearExpiry(u, now):
		return colAmber
	default:
		return colGrey
	}
}

func nameColor(u helpers.UserTraffic) string {
	if !u.Enabled {
		return colDim
	}
	return colText
}

func expiryColor(u helpers.UserTraffic, now time.Time) string {
	if isNearExpiry(u, now) {
		return colAmber
	}
	return colDim
}

func expiryLabel(u helpers.UserTraffic) string {
	if u.ExpiryTime == 0 {
		return "∞"
	}
	return "until " + time.UnixMilli(u.ExpiryTime).Format("02 Jan 2006")
}

func isNearExpiry(u helpers.UserTraffic, now time.Time) bool {
	if u.ExpiryTime == 0 {
		return false
	}
	exp := time.UnixMilli(u.ExpiryTime)
	return exp.After(now) && exp.Before(now.AddDate(0, 0, nearExpiryDays))
}
