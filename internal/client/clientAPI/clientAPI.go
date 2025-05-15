package clientapi

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
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

	// Регистрация на сервере
	UserLogin struct {
		Token    string
		Name     string
		Password string
	}
)

// Получение статуса сервера. Возвращается ошибка.
func (rx *RxStatusSrv) ReqStatusServer(token string) error {

	u := "https://" + os.Getenv("HTTPS_SERVER_IP") + ":" + os.Getenv("HTTPS_SERVER_PORT") + "/status"

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("authorization", token)

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
func (rx *RxDataDB) ReqDataDB(token string) error {

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

	req.Header.Set("authorization", token)

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

// Регистрация на сервере
func (user *UserLogin) LoginHttpsServer() error {

	u := "https://" + os.Getenv("HTTPS_SERVER_IP") + ":" + os.Getenv("HTTPS_SERVER_PORT") + "/registration"
	body := bytes.NewBuffer([]byte(fmt.Sprintf("%s %s", user.Name, user.Password)))

	// Формирование запроса
	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return errors.New("login -> ошибка при создании запроса регистрации на сервере")
	}

	// Создание https клиента.
	client, err := createHttpsClient()
	if err != nil {
		return fmt.Errorf("login -> ошибка создания клиента при запросе данных: {%v}", err)
	}

	// Запрос
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("login -> ошибка при выполнении запроса к https серверу")
	}

	// Ответ
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.New("login -> ошибка при чтении тела ответа сервера")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Проверка результата и фиксация данных
	if resp.StatusCode == http.StatusOK {
		user.Token = string(respBody)
		return nil
	}

	user.Token = ""
	return errors.New("login -> регистрация пользователя не выполнена")
}
