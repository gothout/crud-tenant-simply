package mailer

type Service interface {
	SendRaw(to, subject, body string) error
	SendTemplate(to, subject, tpl string, data interface{}) error
}
