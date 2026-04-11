package sheetapp

// PDFRequest carries the parameters an sheet supplies when generating a sheet.
type PDFRequest struct {
	RodeoID int
	// Future: round filter, event filter, layout options, etc.
}

// PDFResponse holds the rendered PDF bytes and a suggested filename.
type PDFResponse struct {
	Filename string
	Data     []byte
}
