// Branded PDF reports for a client evaluation (executive and technical).
//
// Everything printed comes from the same server-side data the portal shows
// (client-visible findings, effective remediation state, SLA policy); nothing is
// estimated, scored or written by an LLM here. Recommendations are fixed rules
// over those numbers. Brand assets (NEXUS logo, mark, Geist typeface — SIL OFL)
// are embedded in the binary so the document looks the same everywhere.
package server

import (
	"bytes"
	"embed"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/go-pdf/fpdf"

	"github.com/nexus-ia/nexus/db"
)

//go:embed reportassets/*.ttf reportassets/*.png
var reportAssets embed.FS

const (
	reportExecutive = "executive"
	reportTechnical = "technical"
)

type reportInput struct {
	Kind        string // executive | technical
	Lang        string // es | en
	OrgName     string
	Title       string
	Scope       string
	RequestID   int64
	RequestedAt time.Time
	GeneratedAt time.Time
	Draft       bool // operator preview of unreleased results
	Items       []*db.RemediationItem
	Reports     map[int64]string
}

type rgb struct{ r, g, b int }

var (
	cNavy       = rgb{11, 27, 58}
	cNavy2      = rgb{12, 52, 92}
	cCyan       = rgb{14, 116, 144}
	cCyanBright = rgb{34, 211, 238}
	cInk        = rgb{15, 23, 42}
	cText       = rgb{51, 65, 85}
	cMuted      = rgb{100, 116, 139}
	cOnNavy     = rgb{186, 200, 220}
	cLine       = rgb{226, 232, 240}
	cSoft       = rgb{241, 245, 249}
	cBg         = rgb{248, 250, 252}
	cWhite      = rgb{255, 255, 255}
	cRed        = rgb{190, 18, 60}
	cGreen      = rgb{4, 120, 87}
)

// Severity colours: text/accent and light tint, matching the portal badges.
var sevTone = map[string][2]rgb{
	"critical": {{190, 18, 60}, {255, 228, 230}},
	"high":     {{194, 65, 12}, {255, 237, 213}},
	"medium":   {{161, 98, 7}, {254, 243, 199}},
	"low":      {{3, 105, 161}, {224, 242, 254}},
	"unrated":  {{71, 85, 105}, {241, 245, 249}},
}

var reportSeverities = []string{"critical", "high", "medium", "low", "unrated"}

func sevKey(s string) string {
	if _, ok := sevTone[s]; ok && s != "unrated" {
		return s
	}
	return "unrated"
}

type rpt struct {
	f        *fpdf.Fpdf
	t        *reportText
	in       *reportInput
	pageW    float64
	pageH    float64
	ml       float64
	contentW float64
	bottom   float64 // y beyond which content must break to a new page
	curFont  string
}

// reportFonts maps the style names used by rpt.font to the embedded files.
var reportFonts = map[string]string{
	"R": "Geist-Regular.ttf", "SB": "Geist-SemiBold.ttf", "B": "Geist-Bold.ttf", "M": "GeistMono-Regular.ttf",
}

var (
	glyphSetsOnce sync.Once
	glyphSets     map[string]map[rune]bool // style -> code points with a real glyph
)

func reportGlyphSets() map[string]map[rune]bool {
	glyphSetsOnce.Do(func() {
		glyphSets = map[string]map[rune]bool{}
		for style, file := range reportFonts {
			if b, err := reportAssets.ReadFile("reportassets/" + file); err == nil {
				glyphSets[style] = ttfCoverage(b)
			}
		}
	})
	return glyphSets
}

// ttfCoverage reads a TrueType cmap (format 4 or 12, Unicode BMP only) and
// returns the code points that map to a glyph other than .notdef. fpdf draws
// missing glyphs as empty boxes and reports a fallback width for them, so the
// font file itself is the only reliable answer.
func ttfCoverage(b []byte) map[rune]bool {
	u16 := func(o int) int {
		if o < 0 || o+2 > len(b) {
			return 0
		}
		return int(b[o])<<8 | int(b[o+1])
	}
	u32 := func(o int) int {
		if o < 0 || o+4 > len(b) {
			return 0
		}
		return int(b[o])<<24 | int(b[o+1])<<16 | int(b[o+2])<<8 | int(b[o+3])
	}
	cmap := -1
	for i, n := 0, u16(4); i < n; i++ {
		rec := 12 + 16*i
		if rec+16 <= len(b) && string(b[rec:rec+4]) == "cmap" {
			cmap = u32(rec + 8)
		}
	}
	out := map[rune]bool{}
	if cmap < 0 {
		return out
	}
	sub, best := -1, 0
	for i, n := 0, u16(cmap+2); i < n; i++ {
		rec := cmap + 4 + 8*i
		plat, enc, off := u16(rec), u16(rec+2), cmap+u32(rec+4)
		rank := 0
		switch {
		case u16(off) == 12 && (plat == 0 || (plat == 3 && enc == 10)):
			rank = 2
		case u16(off) == 4 && (plat == 0 || (plat == 3 && enc == 1)):
			rank = 1
		}
		if rank > best {
			sub, best = off, rank
		}
	}
	switch {
	case best == 2:
		for g, n := 0, u32(sub+12); g < n; g++ {
			grp := sub + 16 + 12*g
			start, end, glyph := u32(grp), u32(grp+4), u32(grp+8)
			for c := start; c <= end && c <= 0xFFFF; c++ {
				if glyph+(c-start) != 0 {
					out[rune(c)] = true
				}
			}
		}
	case best == 1:
		seg := u16(sub+6) / 2
		ends, starts := sub+14, sub+16+2*seg
		deltas, ranges := starts+2*seg, starts+4*seg
		for i := 0; i < seg; i++ {
			start, end := u16(starts+2*i), u16(ends+2*i)
			delta, ro := u16(deltas+2*i), u16(ranges+2*i)
			for c := start; c <= end && c < 0xFFFF; c++ {
				g := (c + delta) & 0xFFFF
				if ro != 0 {
					if g = u16(ranges + 2*i + ro + 2*(c-start)); g != 0 {
						g = (g + delta) & 0xFFFF
					}
				}
				if g != 0 {
					out[rune(c)] = true
				}
			}
		}
	}
	return out
}

const (
	reportMarginX   = 18.0
	reportMarginTop = 22.0
	reportMarginBot = 20.0
)

// renderReport builds the PDF and returns its bytes.
func renderReport(in *reportInput) ([]byte, error) {
	t := reportTexts[in.Lang]
	if t == nil {
		t = reportTexts["es"]
	}
	f := fpdf.New("P", "mm", "A4", "")
	p := &rpt{f: f, t: t, in: in, ml: reportMarginX}
	p.pageW, p.pageH = f.GetPageSize()
	p.contentW = p.pageW - 2*reportMarginX
	p.bottom = p.pageH - reportMarginBot
	f.SetMargins(reportMarginX, reportMarginTop, reportMarginX)
	f.SetAutoPageBreak(true, reportMarginBot)
	f.SetCellMargin(0)

	for name, file := range reportFonts {
		b, err := reportAssets.ReadFile("reportassets/" + file)
		if err != nil {
			return nil, err
		}
		f.AddUTF8FontFromBytes("geist"+name, "", b)
	}
	for name, file := range map[string]string{"logo": "nexus-logo.png", "mark": "nexus-mark.png"} {
		b, err := reportAssets.ReadFile("reportassets/" + file)
		if err != nil {
			return nil, err
		}
		f.RegisterImageOptionsReader(name, fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(b))
	}

	kindLabel := t.ExecTitle
	if in.Kind == reportTechnical {
		kindLabel = t.TechTitle
	}
	f.SetTitle(kindLabel+" — "+in.Title, true)
	f.SetSubject(in.OrgName, true)
	f.SetAuthor("NEXUS Cybersecurity Global", true)
	f.SetCreator("NEXUS Cybersecurity Global", true)
	f.SetLang(in.Lang)
	f.SetCreationDate(in.GeneratedAt)
	f.AliasNbPages("{nb}")

	f.SetHeaderFuncMode(func() {
		if in.Draft {
			p.watermark()
		}
		if f.PageNo() == 1 {
			return
		}
		p.fill(cNavy)
		f.Rect(0, 0, p.pageW, 13, "F")
		p.fill(cCyanBright)
		f.Rect(0, 13, p.pageW, 0.6, "F")
		f.ImageOptions("logo", p.ml, 2.4, 0, 8.4, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		p.font("SB", 7.5)
		p.text(cOnNavy)
		f.SetXY(p.ml, 0)
		f.CellFormat(p.contentW, 13, p.s(kindLabel+"  ·  "+in.OrgName), "", 0, "RM", false, 0, "")
	}, true)
	f.SetFooterFunc(func() {
		if f.PageNo() == 1 {
			return
		}
		f.SetY(-13)
		p.draw(cLine)
		f.SetLineWidth(0.2)
		f.Line(p.ml, f.GetY(), p.pageW-p.ml, f.GetY())
		f.SetY(-11)
		p.font("R", 7)
		p.text(cMuted)
		f.CellFormat(p.contentW/2, 5, p.s("NEXUS Cybersecurity Global  ·  "+t.Confidential), "", 0, "L", false, 0, "")
		f.CellFormat(p.contentW/2, 5, fmt.Sprintf(t.PageOf, f.PageNo(), "{nb}"), "", 0, "R", false, 0, "")
	})

	p.cover(kindLabel)
	if in.Kind == reportTechnical {
		p.technical()
	} else {
		p.executive()
	}
	p.methodology()

	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ---- primitives -------------------------------------------------------------

func (p *rpt) font(style string, size float64) {
	p.curFont = style
	p.f.SetFont("geist"+style, "", size)
}

// s prepares text for the CURRENT font: clean() plus "?" for any character the
// typeface has no glyph for (e.g. CJK), instead of an empty box.
func (p *rpt) s(txt string) string {
	cov := reportGlyphSets()[p.curFont]
	return strings.Map(func(r rune) rune {
		if r < 0x80 || unicode.IsSpace(r) || cov[r] {
			return r
		}
		return '?'
	}, clean(txt))
}
func (p *rpt) text(c rgb) { p.f.SetTextColor(c.r, c.g, c.b) }
func (p *rpt) fill(c rgb) { p.f.SetFillColor(c.r, c.g, c.b) }
func (p *rpt) draw(c rgb) { p.f.SetDrawColor(c.r, c.g, c.b) }

// need starts a new page when fewer than h mm are left, so headings never end up
// alone at the bottom of a page.
func (p *rpt) need(h float64) {
	if p.f.GetY()+h > p.bottom {
		p.f.AddPage()
	}
}

var reCtl = regexp.MustCompile(`[\x00-\x08\x0b-\x1f\x7f]`)

// clean makes text safe for the embedded font: the PDF writer indexes glyph
// widths by rune and only covers the Basic Multilingual Plane.
func clean(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "    ")
	s = reCtl.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		if r > 0xFFFF || (r >= 0xD800 && r <= 0xDFFF) || r == unicode.ReplacementChar {
			return -1
		}
		return r
	}, s)
}

func (p *rpt) watermark() {
	f := p.f
	f.TransformBegin()
	f.TransformRotate(35, p.pageW/2, p.pageH/2)
	f.SetAlpha(0.07, "Normal")
	p.font("B", 70)
	p.text(cRed)
	f.SetXY(0, p.pageH/2-15)
	f.CellFormat(p.pageW, 30, p.s(p.t.Draft), "", 0, "C", false, 0, "")
	f.SetAlpha(1, "Normal")
	f.TransformEnd()
}

func (p *rpt) h1(txt string) {
	p.need(26)
	f := p.f
	y := f.GetY() + 2
	p.fill(cCyan)
	f.Rect(p.ml, y+1, 1.4, 6.6, "F")
	p.font("B", 15)
	p.text(cInk)
	f.SetXY(p.ml+4.5, y)
	f.CellFormat(p.contentW-4.5, 8.6, p.s(txt), "", 1, "LM", false, 0, "")
	f.Ln(3.5)
}

func (p *rpt) h2(txt string) {
	p.need(16)
	p.font("SB", 11)
	p.text(cInk)
	p.f.SetX(p.ml)
	p.f.CellFormat(p.contentW, 6.5, p.s(txt), "", 1, "LM", false, 0, "")
	p.f.Ln(1.2)
}

func (p *rpt) para(txt string, size float64, c rgb) {
	p.font("R", size)
	p.text(c)
	p.f.SetX(p.ml)
	p.f.MultiCell(p.contentW, size*0.5, p.s(txt), "", "L", false)
	p.f.Ln(1.6)
}

func (p *rpt) bullet(txt string, indent float64) {
	p.font("R", 9.5)
	lines := p.f.SplitText(p.s(txt), p.contentW-indent-5)
	p.need(float64(len(lines))*4.8 + 1)
	y := p.f.GetY()
	p.fill(cCyan)
	p.f.Rect(p.ml+indent+0.6, y+1.7, 1.4, 1.4, "F")
	p.text(cText)
	for _, ln := range lines {
		p.f.SetX(p.ml + indent + 4.5)
		p.f.CellFormat(p.contentW-indent-4.5, 4.8, ln, "", 1, "L", false, 0, "")
	}
	p.f.Ln(0.8)
}

// chip draws a pill at (x,y) and returns its width.
func (p *rpt) chip(x, y float64, label string, fg, bg rgb) float64 {
	p.font("SB", 6.8)
	label = strings.ToUpper(p.s(label))
	w := p.f.GetStringWidth(label) + 4
	p.fill(bg)
	p.f.RoundedRect(x, y, w, 4.6, 2.3, "1234", "F")
	p.text(fg)
	p.f.SetXY(x, y)
	p.f.CellFormat(w, 4.6, label, "", 0, "CM", false, 0, "")
	return w
}

func (p *rpt) sevChip(x, y float64, sev string) float64 {
	k := sevKey(sev)
	return p.chip(x, y, p.t.Sev[k], sevTone[k][0], sevTone[k][1])
}

// ---- tables -------------------------------------------------------------------

type tcol struct {
	title string
	w     float64 // fraction of the content width
	align string
}

type tcell struct {
	text  string
	sev   string // when set, render a severity chip instead of text
	color *rgb
	bold  bool
}

func (p *rpt) table(cols []tcol, rows [][]tcell) {
	f := p.f
	widths := make([]float64, len(cols))
	for i, c := range cols {
		widths[i] = c.w * p.contentW
	}
	const lineH, padX, padY = 4.3, 2.2, 1.9
	header := func() {
		p.font("SB", 7.2)
		p.fill(cNavy)
		p.text(cWhite)
		x := p.ml
		y := f.GetY()
		f.Rect(p.ml, y, p.contentW, 7, "F")
		for i, c := range cols {
			f.SetXY(x+padX, y)
			f.CellFormat(widths[i]-2*padX, 7, strings.ToUpper(p.s(c.title)), "", 0, alignOr(c.align)+"M", false, 0, "")
			x += widths[i]
		}
		f.SetXY(p.ml, y+7)
	}
	p.need(7 + 9)
	header()
	for ri, row := range rows {
		p.font("R", 8.4)
		// Row height = tallest wrapped cell.
		cellLines := make([][]string, len(cols))
		maxLines := 1
		for i, c := range row {
			if c.sev != "" {
				cellLines[i] = nil
				continue
			}
			if c.bold {
				p.font("SB", 8.4)
			} else {
				p.font("R", 8.4)
			}
			ls := f.SplitText(p.s(c.text), widths[i]-2*padX)
			if len(ls) == 0 {
				ls = []string{""}
			}
			cellLines[i] = ls
			if len(ls) > maxLines {
				maxLines = len(ls)
			}
		}
		h := float64(maxLines)*lineH + 2*padY
		if f.GetY()+h > p.bottom {
			f.AddPage()
			header()
		}
		y := f.GetY()
		if ri%2 == 1 {
			p.fill(cBg)
			f.Rect(p.ml, y, p.contentW, h, "F")
		}
		x := p.ml
		for i, c := range row {
			if c.sev != "" {
				p.sevChip(x+padX, y+padY+(lineH-4.6)/2+0.2, c.sev)
			} else {
				if c.bold {
					p.font("SB", 8.4)
				} else {
					p.font("R", 8.4)
				}
				col := cText
				if c.color != nil {
					col = *c.color
				}
				p.text(col)
				for li, ln := range cellLines[i] {
					f.SetXY(x+padX, y+padY+float64(li)*lineH)
					f.CellFormat(widths[i]-2*padX, lineH, ln, "", 0, alignOr(cols[i].align)+"M", false, 0, "")
				}
			}
			x += widths[i]
		}
		p.draw(cLine)
		f.SetLineWidth(0.15)
		f.Line(p.ml, y+h, p.ml+p.contentW, y+h)
		f.SetXY(p.ml, y+h)
	}
	f.Ln(5)
}

func alignOr(a string) string {
	if a == "" {
		return "L"
	}
	return a
}

// ---- data ---------------------------------------------------------------------

type sevCount struct{ total, open, inProgress, fixed, risk int }

type reportStats struct {
	bySev          map[string]*sevCount
	total          sevCount
	overdue        int
	unassignedOpen int
	awaitingVerify int // being fixed but no verification requested yet
	exposure       string
}

func computeStats(items []*db.RemediationItem) reportStats {
	st := reportStats{bySev: map[string]*sevCount{}}
	for _, s := range reportSeverities {
		st.bySev[s] = &sevCount{}
	}
	for _, it := range items {
		c := st.bySev[sevKey(it.Severity)]
		c.total++
		st.total.total++
		switch it.Bucket {
		case "open":
			c.open++
			st.total.open++
		case "in_progress":
			c.inProgress++
			st.total.inProgress++
		case "remediated":
			c.fixed++
			st.total.fixed++
		case "risk_accepted":
			c.risk++
			st.total.risk++
		}
		active := it.Bucket == "open" || it.Bucket == "in_progress"
		if it.Overdue {
			st.overdue++
		}
		if active && it.AssigneeID == nil {
			st.unassignedOpen++
		}
		if it.State == db.RemInProgress {
			st.awaitingVerify++
		}
	}
	st.exposure = "none"
	for _, s := range reportSeverities {
		if c := st.bySev[s]; c.open+c.inProgress > 0 {
			st.exposure = s
			break
		}
	}
	return st
}

func (p *rpt) day(d string) string {
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return d
	}
	return p.date(t)
}

func (p *rpt) date(t time.Time) string {
	t = t.UTC()
	return fmt.Sprintf(p.t.DateFmt, t.Day(), p.t.Months[t.Month()-1], t.Year())
}

func (p *rpt) dueText(it *db.RemediationItem) (string, *rgb) {
	if it.DueDate == "" || it.Bucket == "remediated" || it.Bucket == "risk_accepted" {
		return "—", nil
	}
	if it.Overdue {
		return p.day(it.DueDate) + " · " + p.t.Overdue, &cRed
	}
	return p.day(it.DueDate), nil
}

func findingName(it *db.RemediationItem) string {
	if strings.TrimSpace(it.Name) != "" {
		return it.Name
	}
	return it.VulnClass
}

// ---- cover --------------------------------------------------------------------

func (p *rpt) cover(kindLabel string) {
	f := p.f
	t := p.t
	f.SetAutoPageBreak(false, 0)
	f.AddPage()
	f.LinearGradient(0, 0, p.pageW, 122, cNavy.r, cNavy.g, cNavy.b, cNavy2.r, cNavy2.g, cNavy2.b, 0, 0, 1, 1)
	// Large faded brand mark bleeding off the right edge.
	f.SetAlpha(0.16, "Normal")
	f.ImageOptions("mark", p.pageW-88, 12, 104, 0, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	f.SetAlpha(1, "Normal")
	f.ImageOptions("logo", p.ml, 20, 76, 0, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	p.fill(cCyanBright)
	f.Rect(p.ml, 58, 20, 1.1, "F")
	p.font("SB", 9.5)
	p.text(cCyanBright)
	f.SetXY(p.ml, 63)
	f.CellFormat(p.contentW, 6, strings.ToUpper(p.s(kindLabel)), "", 1, "L", false, 0, "")

	p.font("B", 23)
	p.text(cWhite)
	f.SetXY(p.ml, 71)
	lines := f.SplitText(p.s(p.in.Title), p.contentW-30)
	if len(lines) > 3 {
		lines = append(lines[:2], strings.TrimSpace(lines[2])+"…")
	}
	for _, ln := range lines {
		f.SetX(p.ml)
		f.CellFormat(p.contentW-30, 10, ln, "", 1, "L", false, 0, "")
	}
	p.font("R", 12)
	p.text(cOnNavy)
	f.SetX(p.ml)
	f.CellFormat(p.contentW, 8, p.s(p.in.OrgName), "", 1, "L", false, 0, "")

	// Document control block.
	y := 138.0
	p.font("SB", 8)
	p.text(cCyan)
	f.SetXY(p.ml, y)
	f.CellFormat(p.contentW, 5, strings.ToUpper(t.DocControl), "", 1, "L", false, 0, "")
	y += 8
	scope := strings.TrimSpace(p.in.Scope)
	if scope == "" {
		scope = t.NoScope
	}
	rows := [][2]string{
		{t.Organization, p.in.OrgName},
		{t.Evaluation, fmt.Sprintf("%s (#%d)", p.in.Title, p.in.RequestID)},
		{t.Scope, scope},
		{t.Requested, p.date(p.in.RequestedAt)},
		{t.ReportDate, p.date(p.in.GeneratedAt) + " · " + p.in.GeneratedAt.UTC().Format("15:04") + " UTC"},
		{t.ReportID, fmt.Sprintf("NX-%d-%s", p.in.RequestID, p.in.GeneratedAt.UTC().Format("20060102-1504"))},
		{t.Classification, t.Confidential},
		{t.PreparedBy, "NEXUS Cybersecurity Global"},
	}
	labelW := 42.0
	for _, r := range rows {
		p.font("R", 9)
		vl := f.SplitText(p.s(r[1]), p.contentW-labelW)
		if len(vl) > 4 {
			vl = append(vl[:3], strings.TrimSpace(vl[3])+"…")
		}
		h := float64(len(vl)) * 4.8
		p.font("SB", 8.5)
		p.text(cMuted)
		f.SetXY(p.ml, y)
		f.CellFormat(labelW, 4.8, p.s(r[0]), "", 0, "L", false, 0, "")
		p.font("R", 9)
		p.text(cInk)
		for i, ln := range vl {
			f.SetXY(p.ml+labelW, y+float64(i)*4.8)
			f.CellFormat(p.contentW-labelW, 4.8, ln, "", 0, "L", false, 0, "")
		}
		y += h + 2.2
		p.draw(cLine)
		f.SetLineWidth(0.15)
		f.Line(p.ml, y-1.1, p.ml+p.contentW, y-1.1)
	}

	if p.in.Draft {
		p.fill(rgb{255, 241, 242})
		f.RoundedRect(p.ml, y+3, p.contentW, 11, 2, "1234", "F")
		p.font("SB", 8.5)
		p.text(cRed)
		f.SetXY(p.ml+4, y+3)
		f.CellFormat(p.contentW-8, 11, p.s(t.DraftNote), "", 0, "LM", false, 0, "")
	}

	// Confidentiality notice.
	p.font("R", 7.4)
	p.text(cMuted)
	f.SetXY(p.ml, p.pageH-34)
	f.MultiCell(p.contentW, 3.6, p.s(t.Notice), "", "L", false)

	f.SetAutoPageBreak(true, reportMarginBot)
	f.AddPage()
}

// ---- executive ----------------------------------------------------------------

func (p *rpt) executive() {
	t := p.t
	items := p.in.Items
	st := computeStats(items)

	p.h1(t.ExecSummary)
	p.exposureBox(st)
	p.kpis(st)

	p.h2(t.BySeverity)
	p.severityTable(st)

	open := []*db.RemediationItem{}
	for _, it := range items {
		if it.Bucket == "open" || it.Bucket == "in_progress" {
			open = append(open, it)
		}
	}
	p.h2(t.Priorities)
	if len(open) == 0 {
		p.para(t.NoPriorities, 9.5, cGreen)
	} else {
		if len(open) > 8 {
			open = open[:8]
		}
		rows := [][]tcell{}
		for _, it := range open {
			due, dc := p.dueText(it)
			rows = append(rows, []tcell{
				{sev: sevKey(it.Severity)},
				{text: findingName(it), bold: true},
				{text: t.State[it.State]},
				{text: orDash(it.AssigneeEmail, t.Unassigned)},
				{text: due, color: dc},
			})
		}
		p.table([]tcol{
			{t.ColSeverity, 0.14, ""}, {t.ColFinding, 0.28, ""}, {t.ColState, 0.16, ""},
			{t.ColOwner, 0.24, ""}, {t.ColDue, 0.18, ""},
		}, rows)
	}

	p.h2(t.Recommendations)
	for _, rec := range p.recommendations(st) {
		p.bullet(rec, 0)
	}
	p.f.Ln(3)
}

func orDash(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func (p *rpt) exposureBox(st reportStats) {
	f := p.f
	t := p.t
	fg, bg := cGreen, rgb{236, 253, 245}
	level := t.ExposureNone
	if st.exposure != "none" {
		tone := sevTone[st.exposure]
		fg, bg = tone[0], tone[1]
		level = t.Exposure[st.exposure]
	}
	var line string
	switch {
	case st.total.total == 0:
		line = t.NoFindings
	case st.exposure == "none":
		line = t.AllClosed
	default:
		parts := []string{}
		for _, s := range reportSeverities {
			c := st.bySev[s]
			if n := c.open + c.inProgress; n > 0 {
				parts = append(parts, t.CountSev(n, s))
			}
		}
		line = t.ExposureLine(joinList(parts, t.And))
	}
	p.need(30)
	y := f.GetY()
	p.font("R", 9.5)
	lines := f.SplitText(p.s(line), p.contentW-14)
	h := 17 + float64(len(lines))*4.8
	p.fill(bg)
	f.RoundedRect(p.ml, y, p.contentW, h, 2.5, "1234", "F")
	p.fill(fg)
	f.Rect(p.ml, y, 1.6, h, "F")
	p.font("SB", 7.5)
	p.text(cMuted)
	f.SetXY(p.ml+7, y+4)
	f.CellFormat(p.contentW-14, 4, strings.ToUpper(p.s(t.ExposureTitle)), "", 1, "L", false, 0, "")
	p.font("B", 15)
	p.text(fg)
	f.SetX(p.ml + 7)
	f.CellFormat(p.contentW-14, 8, p.s(level), "", 1, "L", false, 0, "")
	p.font("R", 9.5)
	p.text(cText)
	for _, ln := range lines {
		f.SetX(p.ml + 7)
		f.CellFormat(p.contentW-14, 4.8, ln, "", 1, "L", false, 0, "")
	}
	f.SetY(y + h + 5)
}

func (p *rpt) kpis(st reportStats) {
	f := p.f
	t := p.t
	pct := "—"
	if d := st.total.open + st.total.inProgress + st.total.fixed; d > 0 {
		pct = fmt.Sprintf("%d%%", (st.total.fixed*100+d/2)/d)
	}
	tiles := []struct {
		label, value, hint string
		warn               bool
	}{
		{t.KpiTotal, fmt.Sprint(st.total.total), t.KpiTotalHint, false},
		{t.KpiOpen, fmt.Sprint(st.total.open + st.total.inProgress), t.InProgressN(st.total.inProgress), false},
		{t.KpiFixed, pct, t.FixedOf(st.total.fixed, st.total.open+st.total.inProgress+st.total.fixed), false},
		{t.KpiOverdue, fmt.Sprint(st.overdue), t.KpiOverdueHint, st.overdue > 0},
	}
	p.need(30)
	gap := 4.0
	w := (p.contentW - 3*gap) / 4
	y := f.GetY()
	for i, k := range tiles {
		x := p.ml + float64(i)*(w+gap)
		p.fill(cBg)
		p.draw(cLine)
		f.SetLineWidth(0.25)
		f.RoundedRect(x, y, w, 24, 2.2, "1234", "FD")
		p.font("SB", 7.2)
		p.text(cMuted)
		f.SetXY(x+4, y+3.5)
		f.CellFormat(w-8, 4, p.s(k.label), "", 0, "L", false, 0, "")
		p.font("B", 17)
		if k.warn {
			p.text(cRed)
		} else {
			p.text(cInk)
		}
		f.SetXY(x+4, y+8.5)
		f.CellFormat(w-8, 8, k.value, "", 0, "L", false, 0, "")
		p.font("R", 6.8)
		p.text(cMuted)
		f.SetXY(x+4, y+17.5)
		f.CellFormat(w-8, 4, p.s(k.hint), "", 0, "L", false, 0, "")
	}
	f.SetY(y + 24 + 7)
}

func (p *rpt) severityTable(st reportStats) {
	t := p.t
	rows := [][]tcell{}
	for _, s := range reportSeverities {
		c := st.bySev[s]
		if s == "unrated" && c.total == 0 {
			continue
		}
		rows = append(rows, []tcell{
			{sev: s},
			{text: fmt.Sprint(c.open), bold: c.open > 0},
			{text: fmt.Sprint(c.inProgress)},
			{text: fmt.Sprint(c.fixed)},
			{text: fmt.Sprint(c.risk)},
			{text: fmt.Sprint(c.total), bold: true},
		})
	}
	rows = append(rows, []tcell{
		{text: t.Total, bold: true},
		{text: fmt.Sprint(st.total.open), bold: true},
		{text: fmt.Sprint(st.total.inProgress), bold: true},
		{text: fmt.Sprint(st.total.fixed), bold: true},
		{text: fmt.Sprint(st.total.risk), bold: true},
		{text: fmt.Sprint(st.total.total), bold: true},
	})
	p.table([]tcol{
		{t.ColSeverity, 0.2, ""}, {t.BucketOpen, 0.16, "R"}, {t.BucketInProgress, 0.16, "R"},
		{t.BucketFixed, 0.16, "R"}, {t.BucketRisk, 0.18, "R"}, {t.Total, 0.14, "R"},
	}, rows)
}

// recommendations are fixed rules over the figures above — no free text.
func (p *rpt) recommendations(st reportStats) []string {
	t := p.t
	out := []string{}
	if c := st.bySev["critical"]; c.open+c.inProgress > 0 {
		out = append(out, t.RecSev(c.open+c.inProgress, "critical", db.RemediationSLADays["critical"]))
	}
	if c := st.bySev["high"]; c.open+c.inProgress > 0 {
		out = append(out, t.RecSev(c.open+c.inProgress, "high", db.RemediationSLADays["high"]))
	}
	if st.overdue > 0 {
		out = append(out, t.RecOverdue(st.overdue))
	}
	if st.unassignedOpen > 0 {
		out = append(out, t.RecUnassigned(st.unassignedOpen))
	}
	if st.awaitingVerify > 0 {
		out = append(out, t.RecVerify(st.awaitingVerify))
	}
	if st.total.risk > 0 {
		out = append(out, t.RecRisk(st.total.risk))
	}
	if c := st.bySev["medium"]; c.open+c.inProgress > 0 {
		out = append(out, t.RecPlan(c.open+c.inProgress+st.bySev["low"].open+st.bySev["low"].inProgress))
	}
	out = append(out, t.RecRetest)
	return out
}

func joinList(parts []string, and string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " " + and + " " + parts[len(parts)-1]
}

// ---- technical ----------------------------------------------------------------

func (p *rpt) technical() {
	t := p.t
	items := p.in.Items
	st := computeStats(items)

	p.h1(t.TechSummary)
	p.para(t.TechIntro(st.total.total), 9.5, cText)
	p.severityTable(st)

	if len(items) == 0 {
		p.para(t.NoFindings, 9.5, cGreen)
		return
	}

	p.h2(t.FindingIndex)
	rows := [][]tcell{}
	for i, it := range items {
		due, dc := p.dueText(it)
		rows = append(rows, []tcell{
			{text: findingRef(i), bold: true},
			{sev: sevKey(it.Severity)},
			{text: findingName(it)},
			{text: it.VulnClass},
			{text: t.State[it.State]},
			{text: due, color: dc},
		})
	}
	p.table([]tcol{
		{"ID", 0.08, ""}, {t.ColSeverity, 0.13, ""}, {t.ColFinding, 0.29, ""}, {t.ColClass, 0.15, ""},
		{t.ColState, 0.17, ""}, {t.ColDue, 0.18, ""},
	}, rows)

	for i, it := range items {
		p.findingDetail(i, it)
	}
}

func findingRef(i int) string { return fmt.Sprintf("F-%02d", i+1) }

func (p *rpt) findingDetail(i int, it *db.RemediationItem) {
	f := p.f
	t := p.t
	f.AddPage()
	if i == 0 {
		p.h1(t.FindingDetails)
	}
	y := f.GetY()
	w := p.sevChip(p.ml, y+1.6, it.Severity)
	p.font("B", 13)
	p.text(cInk)
	f.SetXY(p.ml+w+3, y)
	name := findingRef(i) + "  " + findingName(it)
	lines := f.SplitText(p.s(name), p.contentW-w-3)
	for _, ln := range lines {
		f.SetX(p.ml + w + 3)
		f.CellFormat(p.contentW-w-3, 7.5, ln, "", 1, "L", false, 0, "")
	}
	f.Ln(2.5)

	due, _ := p.dueText(it)
	if it.Overdue {
		due = p.day(it.DueDate) + " (" + t.Overdue + ")"
	}
	retest := t.NoRetest
	if lr := it.LastRetest; lr != nil {
		when := lr.CreatedAt
		if lr.FinishedAt != nil {
			when = *lr.FinishedAt
		}
		if lr.Status == "completed" {
			retest = t.Verdict[lr.Verdict] + " · " + p.date(when)
		} else {
			retest = t.RetestStatus[lr.Status] + " · " + p.date(when)
		}
	}
	meta := [][2]string{
		{t.ColClass, orDash(it.VulnClass, "—")},
		{t.ColState, t.State[it.State]},
		{t.Detected, p.date(it.DetectedAt)},
		{t.ColOwner, orDash(it.AssigneeEmail, t.Unassigned)},
		{t.ColDue, due},
		{t.LastRetest, retest},
	}
	p.metaGrid(meta)

	if it.State == db.RemRiskAccepted && it.RiskReason != "" {
		p.h2(t.RiskAccepted)
		p.para(t.RiskAcceptedBy(orDash(it.RiskAcceptedBy, "—"), p.day(it.RiskExpiresAt))+" "+it.RiskReason, 9, cText)
	}
	if strings.TrimSpace(it.Summary) != "" {
		p.h2(t.Description)
		p.para(it.Summary, 9.5, cText)
	}
	if rep := strings.TrimSpace(p.in.Reports[it.FindingID]); rep != "" {
		p.h2(t.TechnicalDetail)
		if len(rep) > 60000 {
			rep = rep[:60000] + "\n\n…"
		}
		p.markdown(rep)
	}
}

// metaGrid prints label/value pairs in two columns inside a soft panel.
func (p *rpt) metaGrid(pairs [][2]string) {
	f := p.f
	rows := (len(pairs) + 1) / 2
	h := float64(rows)*10 + 4
	p.need(h + 4)
	y := f.GetY()
	p.fill(cBg)
	p.draw(cLine)
	f.SetLineWidth(0.25)
	f.RoundedRect(p.ml, y, p.contentW, h, 2.2, "1234", "FD")
	colW := (p.contentW - 8) / 2
	for i, kv := range pairs {
		x := p.ml + 4 + float64(i%2)*colW
		yy := y + 3 + float64(i/2)*10
		p.font("SB", 6.8)
		p.text(cMuted)
		f.SetXY(x, yy)
		f.CellFormat(colW-4, 3.6, strings.ToUpper(p.s(kv[0])), "", 0, "L", false, 0, "")
		p.font("R", 8.8)
		p.text(cInk)
		f.SetXY(x, yy+3.8)
		v := p.s(kv[1])
		if ls := f.SplitText(v, colW-4); len(ls) > 1 {
			v = strings.TrimSpace(ls[0]) + "…"
		}
		f.CellFormat(colW-4, 4.6, v, "", 0, "L", false, 0, "")
	}
	f.SetY(y + h + 6)
}

// ---- markdown (subset) --------------------------------------------------------

var (
	reBold   = regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`)
	reCode   = regexp.MustCompile("`([^`]+)`")
	reLink   = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	reImg    = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	reOrder  = regexp.MustCompile(`^(\d+)[.)]\s+(.*)$`)
	reTblSep = regexp.MustCompile(`^\|?\s*:?-{3,}`)
)

func inlineMD(s string) string {
	s = reImg.ReplaceAllString(s, "$1")
	s = reLink.ReplaceAllString(s, "$1 ($2)")
	s = reBold.ReplaceAllString(s, "$1$2")
	s = reCode.ReplaceAllString(s, "$1")
	return s
}

// markdown renders the subset of Markdown the engine's reports use: headings,
// paragraphs, lists, quotes, rules, fenced code and pipe tables (as monospace).
func (p *rpt) markdown(md string) {
	lines := strings.Split(clean(md), "\n")
	var para, code, table []string
	inCode := false
	flushPara := func() {
		if len(para) > 0 {
			p.para(inlineMD(strings.Join(para, " ")), 9, cText)
			para = nil
		}
	}
	flushCode := func() {
		if len(code) > 0 {
			p.codeBlock(code)
			code = nil
		}
	}
	flushTable := func() {
		if len(table) > 0 {
			p.codeBlock(table)
			table = nil
		}
	}
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			flushPara()
			flushTable()
			if inCode {
				flushCode()
			}
			inCode = !inCode
			continue
		}
		if inCode {
			code = append(code, raw)
			continue
		}
		if !strings.HasPrefix(trimmed, "|") {
			flushTable()
		}
		indent := float64(len(raw)-len(strings.TrimLeft(raw, " "))) / 2 * 4
		if indent > 12 {
			indent = 12
		}
		switch {
		case trimmed == "":
			flushPara()
		case strings.HasPrefix(trimmed, "#"):
			flushPara()
			level := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
			txt := inlineMD(strings.TrimSpace(strings.TrimLeft(trimmed, "#")))
			size := 10.5
			if level <= 1 {
				size = 12
			} else if level >= 3 {
				size = 9.5
			}
			p.need(14)
			p.f.Ln(1.5)
			p.font("SB", size)
			p.text(cInk)
			p.f.SetX(p.ml)
			p.f.MultiCell(p.contentW, size*0.52, txt, "", "L", false)
			p.f.Ln(1.2)
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ "):
			flushPara()
			p.bullet(inlineMD(trimmed[2:]), indent)
		case reOrder.MatchString(trimmed):
			flushPara()
			m := reOrder.FindStringSubmatch(trimmed)
			p.numbered(m[1], inlineMD(m[2]), indent)
		case strings.HasPrefix(trimmed, ">"):
			flushPara()
			p.quote(inlineMD(strings.TrimSpace(strings.TrimLeft(trimmed, ">"))))
		case trimmed == "---" || trimmed == "***" || trimmed == "___":
			flushPara()
			p.need(6)
			y := p.f.GetY() + 2
			p.draw(cLine)
			p.f.SetLineWidth(0.3)
			p.f.Line(p.ml, y, p.ml+p.contentW, y)
			p.f.SetY(y + 3)
		case strings.HasPrefix(trimmed, "|"):
			flushPara()
			if !reTblSep.MatchString(trimmed) {
				table = append(table, trimmed)
			}
		default:
			para = append(para, trimmed)
		}
	}
	flushPara()
	flushTable()
	flushCode() // unterminated fence
}

func (p *rpt) numbered(n, txt string, indent float64) {
	p.font("R", 9.5)
	lines := p.f.SplitText(p.s(txt), p.contentW-indent-7)
	p.need(float64(len(lines))*4.8 + 1)
	p.font("SB", 9.5)
	p.text(cCyan)
	p.f.SetX(p.ml + indent)
	p.f.CellFormat(6, 4.8, n+".", "", 0, "L", false, 0, "")
	p.font("R", 9.5)
	p.text(cText)
	for _, ln := range lines {
		p.f.SetX(p.ml + indent + 6.5)
		p.f.CellFormat(p.contentW-indent-6.5, 4.8, ln, "", 1, "L", false, 0, "")
	}
	p.f.Ln(0.8)
}

func (p *rpt) quote(txt string) {
	p.font("R", 9)
	lines := p.f.SplitText(p.s(txt), p.contentW-8)
	h := float64(len(lines)) * 4.6
	p.need(h + 2)
	y := p.f.GetY()
	p.fill(cLine)
	p.f.Rect(p.ml+1, y, 1, h, "F")
	p.text(cMuted)
	for _, ln := range lines {
		p.f.SetX(p.ml + 5)
		p.f.CellFormat(p.contentW-8, 4.6, ln, "", 1, "L", false, 0, "")
	}
	p.f.Ln(1.5)
}

func (p *rpt) codeBlock(src []string) {
	f := p.f
	p.font("M", 7.4)
	const lh = 3.6
	wrapped := []string{}
	for _, l := range src {
		ls := f.SplitText(p.s(l), p.contentW-6)
		if len(ls) == 0 {
			ls = []string{""}
		}
		wrapped = append(wrapped, ls...)
	}
	if len(wrapped) > 400 {
		wrapped = append(wrapped[:400], "…")
	}
	pad := func() {
		p.fill(cSoft)
		f.Rect(p.ml, f.GetY(), p.contentW, 1.8, "F")
		f.SetY(f.GetY() + 1.8)
	}
	p.need(lh*2 + 4)
	pad()
	for _, ln := range wrapped {
		if f.GetY()+lh > p.bottom {
			f.AddPage()
			pad()
		}
		y := f.GetY()
		p.fill(cSoft)
		f.Rect(p.ml, y, p.contentW, lh, "F")
		p.font("M", 7.4)
		p.text(cInk)
		f.SetXY(p.ml+3, y)
		f.CellFormat(p.contentW-6, lh, ln, "", 0, "LM", false, 0, "")
		f.SetY(y + lh)
	}
	pad()
	f.Ln(2.5)
}

// ---- methodology (both reports) --------------------------------------------------

func (p *rpt) methodology() {
	t := p.t
	p.f.AddPage()
	p.h1(t.MethodTitle)
	p.h2(t.Scope)
	scope := strings.TrimSpace(p.in.Scope)
	if scope == "" {
		scope = t.NoScope
	}
	p.para(scope, 9.5, cText)
	p.h2(t.HowWeWork)
	p.para(t.MethodBody, 9.5, cText)
	p.h2(t.SeverityScale)
	rows := [][]tcell{}
	for _, s := range []string{"critical", "high", "medium", "low"} {
		rows = append(rows, []tcell{
			{sev: s},
			{text: t.SevDef[s]},
			{text: t.Days(db.RemediationSLADays[s]), bold: true},
		})
	}
	p.table([]tcol{{t.ColSeverity, 0.16, ""}, {t.Meaning, 0.6, ""}, {t.SLA, 0.24, ""}}, rows)
	p.h2(t.StatesTitle)
	for _, s := range []string{db.RemOpen, db.RemInProgress, db.RemRetestRequested, db.RemStillVulnerable, db.RemVerifiedFixed, db.RemRiskAccepted} {
		p.bullet(t.State[s]+": "+t.StateDef[s], 0)
	}
	p.f.Ln(2)
	p.h2(t.Limitations)
	p.para(t.LimitationsBody, 9.5, cText)
}
