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
	"time"
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

// Получение статуса сервера. Возвращается ошибка. Возвращаются данные сервера и ошибка.
//
// Парметры:
//
// token - токен пользователя
// name - имя пользователя
// u - URL
// client - указатель на созданный https клиент
func ReqStatusServer(token, name, u string, client *http.Client) (dataRx RxStatusSrv, err error) {

	// Проверка аргементов
	if token == "" {
		log.Fatal("Запрос статуса сервера -> пустое значение аргумента token")
	}
	if name == "" {
		log.Fatal("Запрос статуса сервера -> пустое значение аргумента name")
	}
	if client == nil {
		log.Fatal("Запрос статуса сервера -> нет ссылки на https клиент")
	}

	// Добавление имени пользователя в параметры запроса
	pURL, err := url.Parse(u)
	if err != nil {
		return RxStatusSrv{}, fmt.Errorf("ошибка при парсинге URL: %v", err)
	}
	qPrm := url.Values{}
	qPrm.Set("name", name)

	pURL.RawQuery = qPrm.Encode()

	// Формирование запроса
	req, err := http.NewRequest(http.MethodGet, pURL.String(), nil)
	if err != nil {
		return RxStatusSrv{}, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		return RxStatusSrv{}, fmt.Errorf("ошибка Get запроса: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return RxStatusSrv{}, fmt.Errorf("нет успешности запроса")
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return RxStatusSrv{}, fmt.Errorf("ошибка при чтении тела ответа: %v", err)
	}

	err = json.Unmarshal(respBody, &dataRx)
	if err != nil {
		return RxStatusSrv{}, fmt.Errorf("ошибка обработки данных ответа: %v", err)
	}

	return dataRx, nil
}

// Получение архивных данных БД. Возвращаются данные сервера и ошибка.
//
// Парметры:
//
// token - токен пользователя
// name - имя пользователя
// startDate - дата для выполнения экспорта данных
// u - URL
// client - указатель на созданный https клиент
func ReqDataDB(token, name, startDate, u string, client *http.Client) (dataRx RxDataDB, cntStr string, err error) {

	// Проверка аргементов
	if token == "" {
		return RxDataDB{}, "", errors.New("запрос архивных данных -> пустое значение аргумента token")
	}
	if name == "" {
		return RxDataDB{}, "", errors.New("запрос архивных данных -> пустое значение аргумента name")
	}
	if client == nil {
		return RxDataDB{}, "", errors.New("запрос архивных данных -> нет ссылки на https клиент")
	}
	_, err = time.Parse("2006-01-02", startDate)
	if err != nil {
		return RxDataDB{}, "", errors.New("запрос архивных данных -> дата экспорта не в формате (YYYY-MM-DD)")
	}
	if u == "" {
		return RxDataDB{}, "", errors.New("запрос архивных данных -> пустое значение аргумента URL")
	}

	parseU, err := url.Parse(u)
	if err != nil {
		return RxDataDB{}, "", fmt.Errorf("запрос архивных данных -> ошибка парсинга URL при запросе архивных данных БД: {%v}", err)
	}

	rawQ := url.Values{}
	rawQ.Set("startdate", startDate)
	rawQ.Set("name", name)

	parseU.RawQuery = rawQ.Encode()

	req, err := http.NewRequest(http.MethodGet, parseU.String(), nil)
	if err != nil {
		return RxDataDB{}, "", fmt.Errorf("запрос архивных данных -> ошибка формирования запроса: {%v}", err)
	}

	req.Header.Set("authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		return RxDataDB{}, "", fmt.Errorf("запрос архивных данных -> ошибка выполнения запроса к серверу: {%v}", err)
	}

	// Проверка статус кода ответа сервера
	if resp.StatusCode != http.StatusOK {
		return RxDataDB{}, "", fmt.Errorf("запрос архивных данных -> нет успешности запроса")
	}

	cntStr = resp.Header.Get("Count-Strings")
	if cntStr == "" {
		return RxDataDB{}, "", fmt.Errorf("запрос архивных данных -> нет данных о количестве записей")
	}

	dataResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return RxDataDB{}, "", fmt.Errorf("запрос архивных данных -> ошибка чтения тела ответа: {%v}", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	err = json.Unmarshal(dataResp, &dataRx)
	if err != nil {
		return RxDataDB{}, "", fmt.Errorf("запрос архивных данных -> ошибка при десериализации принятых данных от сервера: {%v}", err)
	}

	return dataRx, cntStr, nil
}

// Регистрация на сервере. Возвращается ошибка.
//
// Параметры:
//
// name - имя пользователя
// password - пароль пользователя
// u - URL
// client - указатель на https клиент
func ReqLoginServer(name, password, u string, client *http.Client) (user UserLogin, err error) {

	// Проверка аргументов
	if name == "" {
		return UserLogin{}, errors.New("login -> нет содержимого в аргументе name")
	}
	if password == "" {
		return UserLogin{}, errors.New("login -> нет содержимого в аргументе password")
	}
	if u == "" {
		return UserLogin{}, errors.New("login -> нет содержимого в аргументе u")
	}
	if client == nil {
		return UserLogin{}, errors.New("login -> нет содержимого в указателе на Http клиент")
	}

	body := bytes.NewBuffer([]byte(fmt.Sprintf("%s %s", name, password)))

	// Формирование запроса
	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return UserLogin{}, errors.New("login -> ошибка при создании запроса регистрации на сервере")
	}

	// Запрос
	resp, err := client.Do(req)
	if err != nil {
		return UserLogin{}, errors.New("login -> ошибка при выполнении запроса к https серверу")
	}

	// Проверка статус-кода ответа
	if resp.StatusCode != http.StatusOK {
		return UserLogin{}, errors.New("login -> ошибка, сервер не вернул код 200")
	}

	// Ответ
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserLogin{}, errors.New("login -> ошибка при чтении тела ответа сервера")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Фиксация данных
	user.Token = string(respBody)
	user.Name = name
	return user, nil

}

// Создание HTTPS клиента. Функция возвращает https клиент и ошибку
func CreateHttpsClient() (client *http.Client, err error) {

	// Загрузка сертификатов
	certPool := x509.NewCertPool()

	// Путь к сертификату
	cacert, err := os.ReadFile(os.Getenv("HTTPS_SERVER_KEY_PUBLIC"))
	if err != nil {
		log.Fatalf("ошибка при чтении CA-сертификата: %v", err)
	}

	// Добавление сертификатав пул доверенных сертификатов
	if ok := certPool.AppendCertsFromPEM(cacert); !ok {
		log.Fatal("не удалось добавить CA-сертификат в пул")
	}

	// Настройка клиент с TLS конфигурацией
	client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool, // пул сертификатов CA
			},
		},
	}

	return client, nil
}
