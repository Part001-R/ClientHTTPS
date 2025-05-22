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

	// Для передачи даты и имени
	DateNameT struct {
		Date string `json:"date"`
		Name string `json:"name"`
	}

	// Для передачи имени
	NameT struct {
		Name string `json:"name"`
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
		return RxStatusSrv{}, errors.New("req-status -> пустое значение аргумента token")
	}
	if name == "" {
		return RxStatusSrv{}, errors.New("req-status -> пустое значение аргумента name")
	}
	if client == nil {
		return RxStatusSrv{}, errors.New("req-status -> нет ссылки на http клиент")
	}
	if u == "" {
		return RxStatusSrv{}, errors.New("req-status -> пустое значение URL")
	}

	// Тело запроса
	infoTx := NameT{
		Name: name,
	}

	bytesBody, err := json.Marshal(infoTx)
	if err != nil {
		return RxStatusSrv{}, fmt.Errorf("req-status -> ошибка маршалинга данных: {%v}", err)
	}

	reqBody := bytes.NewBuffer(bytesBody)

	// Формирование запроса
	req, err := http.NewRequest(http.MethodPost, u, reqBody)
	if err != nil {
		return RxStatusSrv{}, fmt.Errorf("req-status -> ошибка создания запроса: %v", err)
	}

	req.Header.Set("authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		return RxStatusSrv{}, fmt.Errorf("req-status -> ошибка запроса: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return RxStatusSrv{}, fmt.Errorf("req-status -> нет успешности запроса")
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return RxStatusSrv{}, fmt.Errorf("req-status -> ошибка при чтении тела ответа: %v", err)
	}

	err = json.Unmarshal(respBody, &dataRx)
	if err != nil {
		return RxStatusSrv{}, fmt.Errorf("req-status -> ошибка обработки данных ответа: %v", err)
	}

	return dataRx, nil
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
		return 0, errors.New("req-cntStr -> пустое значение аргумента token")
	}
	if name == "" {
		return 0, errors.New("req-cntStr -> пустое значение аргумента name")
	}
	if client == nil {
		return 0, errors.New("req-cntStr -> нет ссылки на https клиент")
	}
	if startDate == "" {
		return 0, errors.New("req-cntStr -> пустое значение даты")
	}
	_, err = time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0, errors.New("req-cntStr -> принятая дата не в формате YYYY-MM-DD")
	}
	if u == "" {
		return 0, errors.New("req-cntStr -> пустое значение аргумента URL")
	}

	// Тело запроса
	infoTx := DateNameT{
		Date: startDate,
		Name: name,
	}

	bytesBody, err := json.Marshal(infoTx)
	if err != nil {
		return 0, fmt.Errorf("req-cntStr -> ошибка маршалинга данных: {%v}", err)
	}

	reqBody := bytes.NewBuffer(bytesBody)

	// Формирование запроса
	req, err := http.NewRequest(http.MethodPost, u, reqBody)
	if err != nil {
		return 0, fmt.Errorf("req-cntStr -> ошибка формирования запроса: {%v}", err)
	}

	req.Header.Set("authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("req-cntStr -> ошибка выполнения запроса к серверу: {%v}", err)
	}

	// Статус ответа
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("req-cntStr -> сервер вернул не код 200")
	}

	// Ответ
	dataResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("req-cntStr -> ошибка чтения тела ответа: {%v}", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	rxJson := CntStrT{}

	err = json.Unmarshal(dataResp, &rxJson)
	if err != nil {
		return 0, fmt.Errorf("req-cntStr -> ошибка при десериализации принятых данных от сервера: {%v}", err)
	}

	cntStr, err = strconv.Atoi(rxJson.CntStr)
	if err != nil {
		return 0, fmt.Errorf("req-cntStr -> принятое значение {%s} не является числов", rxJson.CntStr)
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
	if dateDB == "" {
		return PartDataDB{}, errors.New("req-partdatadb -> пустое значение даты")
	}
	_, err = time.Parse("2006-01-02", dateDB)
	if err != nil {
		return PartDataDB{}, errors.New("req-partdatadb -> значение даты не в формате YYYY-MM-DD")
	}
	if token == "" {
		return PartDataDB{}, errors.New("req-partdatadb -> пустое значение токена")
	}
	if name == "" {
		return PartDataDB{}, errors.New("req-partdatadb -> пустое значение имени")
	}
	if u == "" {
		return PartDataDB{}, errors.New("req-partdatadb -> пустое содержимое URL")
	}
	if client == nil {
		return PartDataDB{}, errors.New("req-partdatadb -> нет указателя на https клиент")
	}

	// Тело запроса
	infoTx := DateNameT{
		Date: dateDB,
		Name: name,
	}

	bytesBody, err := json.Marshal(infoTx)
	if err != nil {
		return PartDataDB{}, fmt.Errorf("req-partdatadb -> ошибка маршалинга данных: {%v}", err)
	}

	reqBody := bytes.NewBuffer(bytesBody)

	// Параметры запроса
	parseU, err := url.Parse(u)
	if err != nil {
		return PartDataDB{}, errors.New("req-partdatadb -> ошибка парсинга URL")
	}
	qP := url.Values{}
	qP.Set("numbReg", fmt.Sprintf("%d", numbReg))
	qP.Set("strLimit", fmt.Sprintf("%d", strLimit))
	qP.Set("strOffSet", fmt.Sprintf("%d", strOffSet))

	parseU.RawQuery = qP.Encode()

	// Формирование запроса
	req, err := http.NewRequest(http.MethodPost, parseU.String(), reqBody)
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
		return PartDataDB{}, errors.New("req-partdatadb -> сервер не вернул код 200")
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
	if startDate == "" {
		return []PartDataDB{}, errors.New("queReq -> пустое значение даты")
	}
	_, err = time.Parse("2006-01-02", startDate)
	if err != nil {
		return []PartDataDB{}, errors.New("queReq -> значение даты не в формате YYYY-MM-DD")
	}
	if token == "" {
		return []PartDataDB{}, errors.New("queReq -> пустое значение token")
	}
	if name == "" {
		return []PartDataDB{}, errors.New("queReq -> пустое значение name")
	}
	if u == "" {
		return []PartDataDB{}, errors.New("queReq -> пустое значение URL")
	}
	if cntStr < 0 {
		return []PartDataDB{}, errors.New("queReq -> в количестве строк отрицательное число")
	}
	if client == nil {
		return []PartDataDB{}, errors.New("queReq -> нет указателя на https клиента")
	}

	// Если количество строк в запросе = 0
	if cntStr == 0 {
		return []PartDataDB{}, nil
	}

	collectRxDataDB := make([]PartDataDB, 0)

	// Вычисление количества необходимых запросов
	iter := cntStr / 100

	// Запросы
	if iter == 0 {

		rxData, err := ReqPartDataDB(0, 100, 0, startDate, token, name, u, client)
		if err != nil {
			return []PartDataDB{}, errors.New("queReq -> ошибка при выполнении запроса при количестве строк < 100")
		}
		collectRxDataDB = append(collectRxDataDB, rxData)

	} else {

		for i := 0; i < iter; i++ {

			// отображение процентов выполнения получения данных от сервера
			percentage := float64(i+1) / float64(iter) * 100
			fmt.Printf("Загрузка данных: %.1f%%\r", percentage)

			rxData, err := ReqPartDataDB(i, 100, 100*i, startDate, token, name, u, client)
			if err != nil {
				return []PartDataDB{}, fmt.Errorf("queReq -> ошибка при выполнении запроса при количестве строк >= 100, на итерации {%d}, {%v}", i, err)
			}
			collectRxDataDB = append(collectRxDataDB, rxData)

			time.Sleep(10 * time.Millisecond) // установка небольшой паузы между очередным запросом
		}
	}
	fmt.Println()

	return collectRxDataDB, nil
}
