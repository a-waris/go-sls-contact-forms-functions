package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"text/template"
)

type FormType struct {
	DBTable      string
	TeamTemplate string
	UserTemplate string
	TeamSubject  string
	UserSubject  string
}

func arrayToString(array map[string]interface{}) string {
	var values []string
	for key, value := range array {
		values = append(values, fmt.Sprintf("%s: %v", key, value))
	}
	return strings.Join(values, ", ")
}

func extractFormFromArgs(args map[string]interface{}) (TaggedOfferContactForm, error) {
	// Extract and validate required fields
	name, nameOk := args["name"].(string)
	email, emailOk := args["email"].(string)
	phone, phoneOk := args["phone"].(string)

	if !nameOk || !emailOk || !phoneOk {
		return TaggedOfferContactForm{}, errors.New("Missing required fields")
	}

	// Process tags
	var tag string
	if tagValue, ok := args["tag"].(string); ok {
		tag = tagValue
	}

	var tags []string
	if tagsValue, ok := args["tags"].([]interface{}); ok {
		for _, tagItem := range tagsValue {
			if tagStr, ok := tagItem.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	if len(tags) > 0 {
		if tag != "" {
			tags = append([]string{tag}, tags...)
		}
		tag = strings.Join(tags, ", ")
	} else if tag == "" {
		tag = "untagged"
	}

	// Process additional fields
	var additional string
	if additionalMap, ok := args["additional"].(map[string]interface{}); ok {
		additional = arrayToString(additionalMap)
	}

	return TaggedOfferContactForm{
		Name:       name,
		Email:      email,
		Phone:      phone,
		Tag:        tag,
		Additional: additional,
	}, nil
}

func insertFormIntoDB(form TaggedOfferContactForm) error {
	db, err := GetDBConnection()
	if err != nil {
		return err
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	_, err = db.Exec("INSERT INTO tagged_offer_contact_forms (name, email, phone, tag, additional) VALUES (?, ?, ?, ?, ?)",
		form.Name, form.Email, form.Phone, form.Tag, form.Additional)
	return err
}

func sendEmail(templatePath, subject, to string, cc []string, form TaggedOfferContactForm) error {
	// Read the template directly from the embedded filesystem
	templateContent, err := templatesFS.ReadFile(templatePath)
	if err != nil {
		fmt.Println("Failed to read embedded template:", err)
		return err
	}

	//fmt.Println(string(templateContent))

	fromAddress := os.Getenv("MAIL_FROM_ADDRESS")
	fromName := os.Getenv("MAIL_FROM_NAME")
	password := os.Getenv("MAIL_PASSWORD")
	smtpHost := os.Getenv("MAIL_HOST")
	smtpPort := os.Getenv("MAIL_PORT")

	auth := smtp.PlainAuth("", fromAddress, password, smtpHost)

	// Parse and execute the HTML template
	tmpl, err := template.New("emailTemplate").Parse(string(templateContent))
	if err != nil {
		return err
	}

	var tmplBytes bytes.Buffer
	if err := tmpl.Execute(&tmplBytes, form); err != nil {
		return err
	}

	emailBody := tmplBytes.String()

	ccHeader := ""
	if len(cc) > 0 {
		ccHeader = "Cc: " + strings.Join(cc, ", ") + "\n"
	}

	from := fmt.Sprintf("%s <%s>", fromName, fromAddress)

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		ccHeader +
		"Subject: " + subject + "\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		emailBody

	recipients := append([]string{to}, cc...)
	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, fromAddress, recipients, []byte(msg))
	if err != nil {
		return err
	}

	return nil
}
