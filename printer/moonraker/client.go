package moonraker

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// UploadAndPrint загружает G-Code на принтер и сразу запускает печать
func UploadAndPrint(printerIP, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}
	io.Copy(part, file)

	// Флаг 'print=true' заставляет Moonraker начать печать сразу после загрузки
	_ = writer.WriteField("print", "true")
	writer.Close()

	url := fmt.Sprintf("http://%s/server/files/upload", printerIP)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Moonraker error: %d", resp.StatusCode)
	}
	return nil
}

// SendCommand отправляет команды управления: pause, resume, stop, estop
func SendCommand(printerIP, action string) error {
	var endpoint string
	switch action {
	case "pause":
		endpoint = "/printer/print/pause"
	case "resume":
		endpoint = "/printer/print/resume"
	case "stop":
		endpoint = "/printer/print/cancel"
	case "estop":
		endpoint = "/printer/emergency_stop"
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	url := fmt.Sprintf("http://%s%s", printerIP, endpoint)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("printer rejected command: %d", resp.StatusCode)
	}
	return nil
}
