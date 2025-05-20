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
	"strconv"
	"time"
)

type (
	// Для приёма количества строк
	CntStrT struct {
		CntStr string `json:"cntstr"`
	}

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

	// Для хранения всех запрошенных частей
	AllDataDB struct {
		AllData []PartDataDB
	}
	PartDataDB struct {
		NumbReq int      `json:"numbreq"`
		Data    []DataEl `json:"data"`
	}
)

// Получение статуса сервера. Возвращается ошибка. Возвращаются данные сервера и ошибка.
//
// Парметры:
//
// token - токен пользователя.
// name - имя пользователя.
// u - URL.
// client - указатель на созданный https клиент.
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
// token - токен пользователя.
// name - имя пользователя.
// startDate - дата для выполнения экспорта данных.
// u - URL.
// client - указатель на созданный https клиент.
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

// Получение количества записей в БД по указанной дате. Возвращаются количество строк и ошибку.
//
// Парметры:
//
// token - токен пользователя.
// name - имя пользователя.
// startDate - дата для выполнения экспорта данных.
// u - URL.
// client - указатель на созданный https клиент.
func ReqCntStrByDateDB(token, name, startDate, u string, client *http.Client) (cntStr int, err error) {

	// Проверка аргементов
	if token == "" {
		return 0, errors.New("запрос количества строк -> пустое значение аргумента token")
	}
	if name == "" {
		return 0, errors.New("запрос количества строк -> пустое значение аргумента name")
	}
	if client == nil {
		return 0, errors.New("запрос количества строк -> нет ссылки на https клиент")
	}
	_, err = time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0, errors.New("запрос количества строк -> дата экспорта не в формате (YYYY-MM-DD)")
	}
	if u == "" {
		return 0, errors.New("запрос количества строк -> пустое значение аргумента URL")
	}

	parseU, err := url.Parse(u)
	if err != nil {
		return 0, fmt.Errorf("запрос количества строк -> ошибка парсинга URL при запросе архивных данных БД: {%v}", err)
	}

	rawQ := url.Values{}
	rawQ.Set("date", startDate)
	rawQ.Set("name", name)

	parseU.RawQuery = rawQ.Encode()

	req, err := http.NewRequest(http.MethodGet, parseU.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("запрос количества строк -> ошибка формирования запроса: {%v}", err)
	}

	req.Header.Set("authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("запрос количества строк -> ошибка выполнения запроса к серверу: {%v}", err)
	}

	// Проверка статус кода ответа сервера
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("запрос количества строк-> сервер вернул не код 200")
	}

	dataResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("запрос количества строк -> ошибка чтения тела ответа: {%v}", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	rxJson := CntStrT{}

	err = json.Unmarshal(dataResp, &rxJson)
	if err != nil {
		return 0, fmt.Errorf("запрос количества строк -> ошибка при десериализации принятых данных от сервера: {%v}", err)
	}

	cntStr, err = strconv.Atoi(rxJson.CntStr)
	if err != nil {
		return 0, fmt.Errorf("запрос количества строк -> принятое значение {%s} не является числов", rxJson.CntStr)
	}

	return cntStr, nil
}

// Регистрация на сервере. Возвращается ошибка.
//
// Параметры:
//
// name - имя пользователя.
// password - пароль пользователя.
// u - URL.
// client - указатель на https клиент.
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

// Функция реализует очередь запросов на сервер для выгрузки исходных данных. Возвращает ошибку.
//
// Параметры:
//
// startDate - дата экспорта данных.
// token - токен регистрации.
// name - имя пользователя.
// u - URL.
// cntStr - количество запрашиваемых строк.
// client - указатель на https клиента.
func QueReqPartDataDB(startDate, token, name, u string, cntStr int, client *http.Client) (rxRataDB []PartDataDB, err error) {

	// Проверка аргументов
	t, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return []PartDataDB{}, fmt.Errorf("запрос строк -> ошибка в содержимом даты: {%s}", t)
	}
	if token == "" {
		return []PartDataDB{}, errors.New("запрос строк -> пустое значение token")
	}
	if name == "" {
		return []PartDataDB{}, errors.New("запрос строк -> пустое значение name")
	}
	if u == "" {
		return []PartDataDB{}, errors.New("запрос строк -> пустое значение URL")
	}
	if cntStr < 0 {
		return []PartDataDB{}, fmt.Errorf("запрос строк -> в количестве строк отрицательное число: {%d}", cntStr)
	}
	if client == nil {
		return []PartDataDB{}, errors.New("запрос строк -> нет указателя на https клиента")
	}

	collectRxDataDB := make([]PartDataDB, 0)

	// Вычисление количества необходимых запросов
	iter := cntStr / 100

	// Запросы
	if iter == 0 {

		rxData, err := ReqPartDataDB(0, 100, 0, startDate, token, name, u, client)
		if err != nil {
			return []PartDataDB{}, errors.New("ошибка при выполнении запроса при количестве строк < 100")
		}
		collectRxDataDB = append(collectRxDataDB, rxData)

	} else {

		for i := 0; i < iter; i++ {

			// отображение процентов выполнения получения данных от сервера
			percentage := float64(i+1) / float64(iter) * 100
			fmt.Printf("Загрузка данных: %.1f%%\r", percentage)

			rxData, err := ReqPartDataDB(i, 100, 100*i, startDate, token, name, u, client)
			if err != nil {
				return []PartDataDB{}, fmt.Errorf("ошибка при выполнении запроса на итерации {%d}, {%v}", i, err)
			}
			collectRxDataDB = append(collectRxDataDB, rxData)

			time.Sleep(10 * time.Millisecond) // установка небольшой паузы между очередным запросом
		}
	}
	fmt.Println()

	return collectRxDataDB, nil
}

// Частичный запрос строк БД по дате, количеству строк и смещению. Возвращается результат запроса и ошибка.
//
// Параметры:
//
// numbReq - номер запроса.
// strLimit - количество строк.
// strOffSet - смещение номеров строк.
// dataDB - дата.
// token - токен.
// name - имя пользователя.
// u - URL.
// client - указатель на https клиента.
func ReqPartDataDB(numbReg, strLimit, strOffSet int, dateDB, token, name, u string, client *http.Client) (data PartDataDB, err error) {

	// Проверка значений аргументов
	if numbReg < 0 {
		return PartDataDB{}, errors.New("req-partdatadb -> значение аргумента numbReg, меньше нуля")
	}
	if strLimit < 0 {
		return PartDataDB{}, errors.New("req-partdatadb -> значение аргумента strLimit, меньше нуля")
	}
	if strOffSet < 0 {
		return PartDataDB{}, errors.New("req-partdatadb -> значение аргумента strOffSet, меньше нуля")
	}
	_, err = time.Parse("2006-01-02", dateDB)
	if err != nil {
		return PartDataDB{}, fmt.Errorf("req-partdatadb -> значение аргумента dataDB {%s}, не дата", dateDB)
	}
	if u == "" {
		return PartDataDB{}, errors.New("req-partdatadb -> пустое содержимое URL")
	}
	if client == nil {
		return PartDataDB{}, errors.New("req-partdatadb -> нет указателя на https клиент")
	}

	// Параметры запроса
	parseU, err := url.Parse(u)
	if err != nil {
		return PartDataDB{}, errors.New("req-partdatadb -> ошибка парсинга URL")
	}
	qP := url.Values{}
	qP.Set("numbReg", fmt.Sprintf("%d", numbReg))
	qP.Set("strLimit", fmt.Sprintf("%d", strLimit))
	qP.Set("strOffSet", fmt.Sprintf("%d", strOffSet))
	qP.Set("date", dateDB)
	qP.Set("name", name)

	parseU.RawQuery = qP.Encode()

	// Формирование запроса
	req, err := http.NewRequest(http.MethodGet, parseU.String(), nil)
	if err != nil {
		return PartDataDB{}, fmt.Errorf("req-partdatadb -> ошибка формирования запроса {%v}", err)
	}

	req.Header.Set("authorization", token)

	// Запрос к серверу
	resp, err := client.Do(req)
	if err != nil {
		return PartDataDB{}, fmt.Errorf("req-partdatadb -> ошибка выполнения запроса {%v}", err)
	}

	// Проверка статус-кода ответа сервера на 200
	if resp.StatusCode != http.StatusOK {
		return PartDataDB{}, fmt.Errorf("req-partdatadb -> сервер вернул код {%d}", resp.StatusCode)
	}

	// Обработка ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PartDataDB{}, fmt.Errorf("req-partdatadb -> ошибка чтения тела ответа {%v}", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	err = json.Unmarshal(body, &data)
	if err != nil {
		return PartDataDB{}, fmt.Errorf("req-partdatadb -> ошибка десиарелизации ответа {%v}", err)
	}

	return data, nil
}
