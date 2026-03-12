package models

type EmailType string

const (
	EmailTypeOTP EmailType = "otp"
	EmailTypeTestPaper EmailType = "test_paper"
	EmailTypeGeneral EmailType = "general"
)

type EmailJob struct {
	Type EmailType `json:"type"`
	Data interface{} `json:"data"`
}

type OTPData struct {
	RecipientEmail string `json:"email"`
	OTP   string `json:"otp"`
}

type TestPaperData struct {
	RecipientEmail string `json:"email"`
	UserName string `json:"user_name"`
	TestPaperTitle string `json:"test_paper_title"`
	MarksObtained int    `json:"marks_obtained"`
	FeedbackURL   string `json:"feedback_url"`
}

type GeneralEmail struct {
	RecipientEmail string `json:"email"`
	Subject string `json:"subject"`
	Body string `json:"body"`
}