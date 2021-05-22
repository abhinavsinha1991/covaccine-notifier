package main

import (
	"net/smtp"
	"strings"
)

func sendMail(age, dose, district, id, pass, body string) error {
	msg := "From: " + id + "\n" +
		"To: " + id + "\n" +
		"Subject: " + strings.ToUpper(district) + " : DOSE" + dose + " Vaccination slots are available for age: " + age + "\n\n" +
		"Vaccination slots are available at the following centers:\n\n" +
		body

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", id, pass, "smtp.gmail.com"),
		id, []string{id}, []byte(msg))

	if err != nil {
		return err
	}
	return nil
}
