package clientapi

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

type (
	// JSON для приёма данных состояния сервера
	RxStatusSrv struct {
		TimeStart string          `json:"timeStart"`
		MbRTU     []InfoModbusRTU `json:"mbRTU"`
		MbTCP     []InfoModbusTCP `json:"mbTCP"`
		SizeF     SizeFiles       `json:"sizeFiles"`
	}
	InfoModbusRTU struct {
		ConName   string
		Con       string
		ConParams struct {
			BaudRate int
			DataBits int
			Parity   string
			StopBits int
		}
	}
	InfoModbusTCP struct {
		ConName string
		Con     string
	}
	SizeFiles struct {
		I int64
		W int64
		E int64
	}

	// JSON для приёма архивных данных БД
	RxDataDB struct {
		StartDate string   `json:"startdate"`
		Data      []DataEl `json:"datadb"`
	}
	DataEl struct {
		Name      string
		Value     string
		Qual      string
		TimeStamp string
	}
)

// Получение статуса сервера. Возвращается ошибка.
func (rx *RxStatusSrv) ReqStatusServer() error {

	u := "https://" + os.Getenv("HTTPS_SERVER_IP") + ":" + os.Getenv("HTTPS_SERVER_PORT") + "/status"

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %v", err)
	}

	client, err := createHttpsClient()
	if err != nil {
		return fmt.Errorf("ошибка создания клиента при запросе состояния: {%v}", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка Get запроса: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка при чтении тела ответа: %v", err)
	}

	err = json.Unmarshal(respBody, rx)
	if err != nil {
		return fmt.Errorf("ошибка обработки данных ответа: %v", err)
	}

	return nil
}

// Получение архивных данных БД. Возвращается ошибка
func (rx *RxDataDB) ReqDataDB() error {

	u := fmt.Sprintf("https://%s:%s/datadb", os.Getenv("HTTPS_SERVER_IP"), os.Getenv("HTTPS_SERVER_PORT"))

	parseU, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("ошибка парсинга URL при запросе архивных данных БД: {%v}", err)
	}

	rawQ := url.Values{}
	rawQ.Add("startdate", rx.StartDate)

	parseU.RawQuery = rawQ.Encode()

	req, err := http.NewRequest(http.MethodGet, parseU.String(), nil)
	if err != nil {
		return fmt.Errorf("ошибка формирования запроса: {%v}", err)
	}

	client, err := createHttpsClient()
	if err != nil {
		return fmt.Errorf("ошибка создания клиента при запросе данных: {%v}", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса к серверу: {%v}", err)
	}

	dataResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения тела ответа: {%v}", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	err = json.Unmarshal(dataResp, &rx)
	if err != nil {
		return fmt.Errorf("ошибка при десериализации принятых данных от сервера: {%v}", err)
	}

	return nil
}

// Создание HTTPS клиента. Функция возвращает клиента и ошибку
func createHttpsClient() (client *http.Client, err error) {

	// Загрузка сертификатов
	certPool := x509.NewCertPool()

	// Путь к сертификату CA
	cacert, err := os.ReadFile(os.Getenv("HTTPS_SERVER_KEY_PUBLIC"))
	if err != nil {
		log.Fatalf("ошибка при чтении CA-сертификата: %v", err)
	}

	// Добавляем CA-сертификат в пул доверенных сертификатов
	if ok := certPool.AppendCertsFromPEM(cacert); !ok {
		log.Fatal("не удалось добавить CA-сертификат в пул")
	}

	// Настройка клиент с TLS конфигурацией
	client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool, // Указываем пул сертификатов CA
			},
		},
	}

	return client, nil
}
