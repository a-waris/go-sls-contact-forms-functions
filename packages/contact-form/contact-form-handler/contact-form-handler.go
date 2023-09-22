package main

import (
	"embed"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net/smtp"
	"sync"
)

//go:embed templates/*
var templatesFS embed.FS

var smtpClient *smtp.Client

func Main(args map[string]interface{}) map[string]interface{} {
	response := make(map[string]interface{})

	form, err := extractFormFromArgs(args)
	if err != nil {
		response["error"] = err.Error()
		return response
	}

	if err := insertFormIntoDB(form); err != nil {
		response["error"] = "Failed to insert into database"
		return response
	}

	var wg sync.WaitGroup
	var teamEmailErr, userEmailErr error

	wg.Add(2) // We're going to run two goroutines

	go func() {
		defer wg.Done()
		teamEmailErr = sendTeamEmail(form)
	}()

	go func() {
		defer wg.Done()
		userEmailErr = sendUserEmail(form)
	}()

	wg.Wait() // Wait for both goroutines to finish

	if teamEmailErr != nil {
		response["error"] = "Failed to send email to the team"
		return response
	}

	if userEmailErr != nil {
		response["error"] = "Failed to send reply email to the user"
		return response
	}

	response["message"] = "Thank you for contacting"
	return response
}

func sendTeamEmail(form TaggedOfferContactForm) error {
	teamSubject := fmt.Sprintf("%s | Offer Contact Form | Resourceinn | Ad", form.Tag)
	cc := []string{"geekinntech@gmail.com", "gogglebakers@gmail.com", "noman@resourceinn.com", "ahsan@resourceinn.com", "marketing@resourceinn.com"}
	to := "sales@resourceinn.com"
	teamTemplatePath := getTemplatePath("tagged_offer_contact_form.html")
	return sendEmail(teamTemplatePath, teamSubject, to, cc, form)
}

func sendUserEmail(form TaggedOfferContactForm) error {
	userSubject := "Resourceinn (HRM) | Thank You!"
	userTemplatePath := getTemplatePath("offer_contact_form_sender_reply.html")
	return sendEmail(userTemplatePath, userSubject, form.Email, nil, form)
}

func getTemplatePath(filename string) string {
	return "templates/" + filename
}
