package pdf

const (
	pageWidth    = 210.0 // A4 mm
	pageHeight   = 297.0 // A4 mm
	marginLeft   = 15.0
	marginRight  = 15.0
	marginTop    = 20.0
	marginBottom = 20.0
	bodyWidth    = pageWidth - marginLeft - marginRight

	fontSizeH1   = 16.0
	fontSizeH2   = 14.0
	fontSizeH3   = 12.0
	fontSizeBody = 10.0
	fontSizeCode = 9.0
	fontSizeSm   = 8.0

	lineHeight     = 6.0
	lineHeightBody = 6.0

	// AppVersion is embedded in the cover page of generated PDFs.
	AppVersion = "0.5.0"
	appVersion = AppVersion
)

// classColor holds RGB values for a classification level banner.
type classColor struct {
	BgR, BgG, BgB     int // background
	TextR, TextG, TextB int // text (always white)
}

var classColors = map[string]classColor{
	"UNCLASSIFIED": {0, 128, 0, 255, 255, 255},
	"CONFIDENTIAL": {0, 0, 180, 255, 255, 255},
	"RESTRICTED":   {200, 100, 0, 255, 255, 255},
	"SECRET":       {180, 0, 0, 255, 255, 255},
	"TOP SECRET":   {100, 0, 0, 255, 255, 255},
}

func classColorFor(level string) classColor {
	if c, ok := classColors[level]; ok {
		return c
	}
	return classColor{0, 128, 0, 255, 255, 255} // default unclassified
}
