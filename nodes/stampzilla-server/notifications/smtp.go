package notifications

import (
	"crypto/tls"

	log "github.com/cihub/seelog"
	"gopkg.in/gomail.v2"
)

type Smtp struct {
	Server             string `default:localhost`
	Username           string
	Password           string
	Port               int    `default:25`
	Sender             string `default:stampzilla`
	To                 string `default:stampzilla`
	InsecureSkipVerify bool
}

func (self *Smtp) Start() {
}

func (self *Smtp) Dispatch(note Notification) {
	msg := gomail.NewMessage()
	msg.SetHeader("From", self.Sender)
	msg.SetHeader("To", self.To)
	msg.SetHeader("Subject", note.Level.String()+" - "+note.Message)
	msg.SetBody("text/html", "<p>Notification from <b>"+note.Source+"</b>("+note.SourceUuid+")!</p><p>"+note.Level.String()+" - "+note.Message+"</p>")

	mailer := gomail.NewPlainDialer(self.Server, self.Port, self.Username, self.Password)
	mailer.TLSConfig = &tls.Config{InsecureSkipVerify: self.InsecureSkipVerify}
	if err := mailer.DialAndSend(msg); err != nil {
		log.Error("Failed to send mail - ", err)
	}
}
func (self *Smtp) Stop() {
}
