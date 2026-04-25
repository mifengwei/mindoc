package conf

import (
	"strings"
)

type SmtpConf struct {
	EnableMail   bool
	MailNumber   int
	SmtpUserName string
	SmtpHost     string
	SmtpPassword string
	SmtpPort     int
	FormUserName string
	MailExpired  int
	Secure       string
}

func GetMailConfig() *SmtpConf {
	user_name, _ := GetString("smtp_user_name")
	password, _ := GetString("smtp_password")
	smtp_host, _ := GetString("smtp_host")
	smtp_port := GetDefaultInt("smtp_port", 25)
	form_user_name, _ := GetString("form_user_name")
	enable_mail, _ := GetString("enable_mail")
	mail_number := GetDefaultInt("mail_number", 5)
	secure := GetDefaultString("secure", "NONE")

	if secure != "NONE" && secure != "LOGIN" && secure != "SSL" {
		secure = "NONE"
	}
	c := &SmtpConf{
		EnableMail:   strings.EqualFold(enable_mail, "true"),
		MailNumber:   mail_number,
		SmtpUserName: user_name,
		SmtpHost:     smtp_host,
		SmtpPassword: password,
		FormUserName: form_user_name,
		SmtpPort:     smtp_port,
		Secure:       secure,
	}
	return c
}
