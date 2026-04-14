package sheetapp

// ContestantEntry pairs a contestant ID with any session notes the user typed.
type ContestantEntry struct {
	ID    string `json:"id"`
	Notes string `json:"notes"`
}

// PDFRequest is the JSON body sent by the browser when generating a PDF.
type PDFRequest struct {
	RodeoName   string            `json:"rodeoName"`
	RodeoDate   string            `json:"rodeoDate"`
	Contestants []ContestantEntry `json:"contestants"`
}
