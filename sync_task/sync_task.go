package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: sync_task <directory_path> <organization>")
		return
	}
	directoryPath := os.Args[1]
	organization := os.Args[2]
	sendEmailWithAttachment(directoryPath, organization)
}

func sendEmailWithAttachment(directoryPath, organization string) {
	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error walking the path", path, err)
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".xlsx" || filepath.Ext(path) == ".csv") {
			sendEmail(path, organization)
		}
		return nil
	})

	if err != nil {
		fmt.Println("Error during directory walk:", err)
	}
}

func sendEmail(filePath, organization string) {
	from := "FROM"
	to := "TO"
	subject := organization
	body := "Please find attached the daily report."

	message := fmt.Sprintf("From: %s\r\n", from)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: multipart/mixed; boundary=boundary\r\n"
	message += "\r\n"
	message += "--boundary\r\n"
	message += "Content-Type: text/plain; charset=\"UTF-8\"\r\n"
	message += "Content-Transfer-Encoding: 7bit\r\n"
	message += "\r\n" + body + "\r\n"

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	encodedContent := encodeBase64(fileContent)

	message += "\r\n--boundary\r\n"
	message += fmt.Sprintf("Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet; name=\"%s\"\r\n", filepath.Base(filePath))
	message += "Content-Disposition: attachment; filename=\"" + filepath.Base(filePath) + "\"\r\n"
	message += "Content-Transfer-Encoding: base64\r\n"
	message += "\r\n" + encodedContent + "\r\n"
	message += "--boundary--\r\n"

	smtpHost := "smtpHost"
	smtpPort := "smtpPort"
	auth := smtp.PlainAuth("", from, "PASSWORD", smtpHost)

	for {
		if checkInternetConnection() {
			err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(message))
			if err != nil {
				fmt.Println("Error sending email:", err)
			} else {
				fmt.Println("Email sent successfully!")
			}
			break
		} else {
			fmt.Println("No internet connection. Retrying in 5 minutes...")
			time.Sleep(5 * time.Minute)
		}
	}
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func checkInternetConnection() bool {
	_, err := http.Get("http://www.google.com")
	return err == nil
}
