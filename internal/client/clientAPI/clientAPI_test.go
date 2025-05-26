package clientapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тест регистрации - успешность
func Test_ReqLoginServer_Success(t *testing.T) {

	// Подготовка данных
	name := "userTest"
	password := "123"
	token := "1234567890"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application-json")

		var dataTx TokenT
		dataTx.Token = token

		byteTx, err := json.Marshal(dataTx)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(byteTx)
		if err != nil {
			require.NoErrorf(t, err, "ошибка при передаче данных в writer:{%v}", err)
		}
	}))
	defer server.Close()

	// Вызом тестируемой функции
	user, err := ReqLoginServer(name, password, server.URL, server.Client())
	require.NoErrorf(t, err, "тестируемая функция вернула ошибку:{%v}", err)
	assert.Equal(t, token, user.Token, "нет соответствия в токене, ожидается {%s} а принято {%s}", token, user.Token)

}

// Тест регистрации - ошибки
func Test_ReqLoginServer_Error(t *testing.T) {

	dataTest := []struct {
		nameTest    string
		user        string
		password    string
		wantError   string
		emptyURL    string
		emptyClient string
	}{
		{
			nameTest:    "Ошибка пустого значения в name",
			user:        "",
			password:    "123",
			wantError:   "login -> нет содержимого в аргументе name",
			emptyURL:    "false",
			emptyClient: "false",
		},
		{
			nameTest:    "Ошибка пустого значения в password",
			user:        "user",
			password:    "",
			wantError:   "login -> нет содержимого в аргументе password",
			emptyURL:    "false",
			emptyClient: "false",
		},
		{
			nameTest:    "Ошибка пустого значения в u",
			user:        "user",
			password:    "123",
			wantError:   "login -> нет содержимого в аргументе u",
			emptyURL:    "true",
			emptyClient: "false",
		},
		{
			nameTest:    "Ошибка пустого значения в Client",
			user:        "user",
			password:    "123",
			wantError:   "login -> нет содержимого в указателе на Http клиент",
			emptyURL:    "false",
			emptyClient: "true",
		},
	}

	// Проверка
	for _, tt := range dataTest {
		t.Run(tt.nameTest, func(t *testing.T) {

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				w.Header().Set("Content-Type", "application-json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("1234567890"))
				if err != nil {
					require.NoErrorf(t, err, "ошибка при передаче данных в writer:{%v}", err)
				}
			}))
			defer server.Close()

			tempURL := server.URL
			tempClient := server.Client()

			if tt.emptyURL == "true" {
				tempURL = ""
			}
			if tt.emptyClient == "true" {
				tempClient = nil
			}

			// Вызом тестируемой функции
			_, err := ReqLoginServer(tt.user, tt.password, tempURL, tempClient)
			e := fmt.Sprintf("%s", err)
			assert.Equalf(t, tt.wantError, e, "ожидалась ошибка: {%v}, а принято {%v}", tt.wantError, e)
		})
	}
}

// Статусные данные сервера - успешность
func Test_ReqStatusServer_Success(t *testing.T) {

	// Подготовка данных
	token := "1234567890"
	name := "test"

	infoMbRTU := InfoModbusRTU{
		ConName: "Con2",
		Con:     "/dev/ttyUSB0",
		ConParams: struct {
			BaudRate int
			DataBits int
			Parity   string
			StopBits int
		}{
			BaudRate: 9600,
			DataBits: 8,
			Parity:   "N",
			StopBits: 1,
		},
	}
	infoMbTCP := InfoModbusTCP{
		ConName: "Con1",
		Con:     "192.168.122.1",
	}
	infoSize := SizeFiles{
		I: 1,
		W: 2,
		E: 3,
	}

	statusSrv := RxStatusSrv{
		TimeStart: "",
		MbRTU:     []InfoModbusRTU{},
		MbTCP:     []InfoModbusTCP{},
		SizeF:     SizeFiles{},
	}

	statusSrv.TimeStart = "22-05-2025 02:18:15"
	statusSrv.MbRTU = append(statusSrv.MbRTU, infoMbRTU)
	statusSrv.MbTCP = append(statusSrv.MbTCP, infoMbTCP)
	statusSrv.SizeF = infoSize

	// Сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application-json")

		// Чтение заголовков звпроса
		token := r.Header.Get("authorization")
		if token == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Чтение тела запроса
		var rxBody NameT

		bytesBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = json.Unmarshal(bytesBody, &rxBody)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		bytesTx, err := json.Marshal(statusSrv)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// Проверка приёма имени
		if name != rxBody.Name {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Ответ
		w.WriteHeader(http.StatusOK)
		w.Write(bytesTx)
	}))
	defer server.Close()

	// Запрос
	rxData, err := ReqStatusServer(token, name, server.URL, server.Client())
	require.NoErrorf(t, err, "ошибка при запросе состояния сервера: {%v}", err)
	assert.Equal(t, statusSrv, rxData, "нет соответствия, ожидается {%v}, а принято {%v}", statusSrv, rxData)

}

// Статусные данные сервера - ошибки
func Test_ReqStatusServer_Error(t *testing.T) {

	// Подготовка данных
	token := "1234567890"
	name := "test"

	argData := []struct {
		nameTest  string
		token     string
		name      string
		useURL    string
		useClient string
		wantErr   string
	}{
		{
			nameTest:  "пустое значение токена",
			token:     "",
			name:      "test",
			useURL:    "true",
			useClient: "true",
			wantErr:   "req-status -> пустое значение аргумента token",
		},
		{
			nameTest:  "другой токен",
			token:     "123",
			name:      "test",
			useURL:    "true",
			useClient: "true",
			wantErr:   "req-status -> нет успешности запроса",
		},
		{
			nameTest:  "пустое значение имени пользователя",
			token:     "1234567890",
			name:      "",
			useURL:    "true",
			useClient: "true",
			wantErr:   "req-status -> пустое значение аргумента name",
		},
		{
			nameTest:  "пустое значение строки URL",
			token:     "1234567890",
			name:      "test",
			useURL:    "false",
			useClient: "true",
			wantErr:   "req-status -> пустое значение URL",
		},
		{
			nameTest:  "нет указателя на http клиент",
			token:     "1234567890",
			name:      "test",
			useURL:    "true",
			useClient: "false",
			wantErr:   "req-status -> нет ссылки на http клиент",
		},
	}

	// Состояние сервера
	infoMbRTU := InfoModbusRTU{
		ConName: "Con2",
		Con:     "/dev/ttyUSB0",
		ConParams: struct {
			BaudRate int
			DataBits int
			Parity   string
			StopBits int
		}{
			BaudRate: 9600,
			DataBits: 8,
			Parity:   "N",
			StopBits: 1,
		},
	}
	infoMbTCP := InfoModbusTCP{
		ConName: "Con1",
		Con:     "192.168.122.1",
	}
	infoSize := SizeFiles{
		I: 1,
		W: 2,
		E: 3,
	}

	statusSrv := RxStatusSrv{
		TimeStart: "",
		MbRTU:     []InfoModbusRTU{},
		MbTCP:     []InfoModbusTCP{},
		SizeF:     SizeFiles{},
	}

	statusSrv.TimeStart = "22-05-2025 02:18:15"
	statusSrv.MbRTU = append(statusSrv.MbRTU, infoMbRTU)
	statusSrv.MbTCP = append(statusSrv.MbTCP, infoMbTCP)
	statusSrv.SizeF = infoSize

	// Тестирование
	for _, tt := range argData {
		t.Run(tt.nameTest, func(t *testing.T) {

			// Сервер
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				w.Header().Set("Content-Type", "application-json")

				// Чтение заголовков звпроса
				tokenH := r.Header.Get("authorization")
				if token == "" {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}

				if tokenH != token {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}

				// Чтение тела запроса
				var rxBody NameT

				bytesBody, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
				defer r.Body.Close()

				err = json.Unmarshal(bytesBody, &rxBody)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				bytesTx, err := json.Marshal(statusSrv)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				// Проверка приёма имени
				if name != rxBody.Name {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}

				// Ответ
				w.WriteHeader(http.StatusOK)
				w.Write(bytesTx)
			}))
			defer server.Close()

			u := server.URL
			if tt.useURL == "false" {
				u = ""
			}
			client := server.Client()
			if tt.useClient == "false" {
				client = nil
			}
			// Запрос
			_, err := ReqStatusServer(tt.token, tt.name, u, client)
			rxErr := fmt.Sprintf("%v", err)
			assert.Equal(t, tt.wantErr, rxErr, "нет соответствия ошибки, ожидается {%s}, а принято {%s}", tt.wantErr, rxErr)

		})
	}

}

// Запрос количество строк по дате - успешность
func Test_ReqCntStrByDateDB_Success(t *testing.T) {

	token := "1234567890"
	name := "test"
	date := "2025-05-05"
	cntStr := 100

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application-json")

		// Чтение заголовка
		tokenH := r.Header.Get("authorization")
		if tokenH == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if tokenH != token {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Тело запроса
		var reqBoddy DateNameT

		bytesBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		err = json.Unmarshal(bytesBody, &reqBoddy)
		if err != nil {

		}

		// Проверка данных тела запроса
		_, err = time.Parse("2006-01-02", reqBoddy.Date)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if reqBoddy.Name == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Ответ
		str := strconv.Itoa(cntStr)
		cntInfo := CntStrT{
			CntStr: str,
		}
		bTx, err := json.Marshal(cntInfo)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(bTx)
	}))

	rxCntStr, err := ReqCntStrByDateDB(token, name, date, server.URL, server.Client())
	require.NoErrorf(t, err, "принята ошибка: {%s}", fmt.Sprintf("%v", err))
	assert.Equalf(t, cntStr, rxCntStr, "нет соответствия в количестве строк. Ожидалось:{%d}, а принято:{%d}", cntStr, rxCntStr)
}

// Запрос количество строк по дате - ошибки
func Test_ReqCntStrByDateDB_Error(t *testing.T) {

	token := "1234567890"
	cntStr := 100

	argData := []struct {
		nameTest  string
		token     string
		name      string
		date      string
		useURL    string
		useClient string
		wantErr   string
	}{
		{
			nameTest:  "пустое значение токена",
			token:     "",
			name:      "test",
			date:      "2025-05-05",
			useURL:    "true",
			useClient: "true",
			wantErr:   "req-cntStr -> пустое значение аргумента token",
		},
		{
			nameTest:  "другой токен",
			token:     "123",
			name:      "test",
			date:      "2025-05-05",
			useURL:    "true",
			useClient: "true",
			wantErr:   "req-cntStr -> сервер вернул не код 200",
		},
		{
			nameTest:  "пустое значение имени пользователя",
			token:     "1234567890",
			name:      "",
			date:      "2025-05-05",
			useURL:    "true",
			useClient: "true",
			wantErr:   "req-cntStr -> пустое значение аргумента name",
		},
		{
			nameTest:  "пустое значение даты",
			token:     "1234567890",
			name:      "test",
			date:      "",
			useURL:    "true",
			useClient: "true",
			wantErr:   "req-cntStr -> пустое значение даты",
		},
		{
			nameTest:  "дата не в формате YYYY-MM-DD",
			token:     "1234567890",
			name:      "test",
			date:      "01-02-2025",
			useURL:    "true",
			useClient: "true",
			wantErr:   "req-cntStr -> принятая дата не в формате YYYY-MM-DD",
		},
		{
			nameTest:  "пустое значение строки URL",
			token:     "1234567890",
			name:      "test",
			date:      "2025-05-05",
			useURL:    "false",
			useClient: "true",
			wantErr:   "req-cntStr -> пустое значение аргумента URL",
		},
		{
			nameTest:  "нет указателя на http клиент",
			token:     "1234567890",
			name:      "test",
			date:      "2025-05-05",
			useURL:    "true",
			useClient: "false",
			wantErr:   "req-cntStr -> нет ссылки на https клиент",
		},
	}

	for _, tt := range argData {
		t.Run(tt.nameTest, func(t *testing.T) {

			// Сервер
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				w.Header().Set("Content-Type", "application-json")

				// Чтение заголовков звпроса
				tokenH := r.Header.Get("authorization")
				if token == "" {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}

				if tokenH != token {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}

				// Чтение тела запроса
				var rxBody DateNameT

				bytesBody, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
				defer r.Body.Close()

				err = json.Unmarshal(bytesBody, &rxBody)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				// Проверка
				_, err = time.Parse("2006-01-02", rxBody.Date)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}

				// Ответ
				cntStr := strconv.Itoa(cntStr)

				cntInfo := CntStrT{
					CntStr: cntStr,
				}
				bTx, err := json.Marshal(cntInfo)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				// Ответ
				w.WriteHeader(http.StatusOK)
				w.Write(bTx)
			}))
			defer server.Close()

			u := server.URL
			if tt.useURL == "false" {
				u = ""
			}
			client := server.Client()
			if tt.useClient == "false" {
				client = nil
			}
			// Запрос
			_, err := ReqCntStrByDateDB(tt.token, tt.name, tt.date, u, client)
			rxErr := fmt.Sprintf("%v", err)
			assert.Equal(t, tt.wantErr, rxErr, "нет соответствия ошибки, ожидается {%s}, а принято {%s}", tt.wantErr, rxErr)
		})
	}
}

// Запрос архивных данных сервера - успешность
func Test_ReqPartDataDB_Success(t *testing.T) {

	numbReq := 0
	strLimit := 34
	strOffSet := 2
	dateDB := "2025-01-02"
	token := "1234567890"
	name := "test"

	simDataDB := make([]DataEl, 0)

	for i := 0; i < 60; i++ {

		tStamp := fmt.Sprintf("2025-05-18T03:01:%d.391321+07:00", i)
		str := DataEl{
			Name:      "Dev3. HR. Тестовая переменная ShortInt",
			Value:     strconv.Itoa(i),
			Qual:      "1",
			TimeStamp: tStamp,
		}

		simDataDB = append(simDataDB, str)
	}

	// Сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application-json")

		// Чтение параметров запроса. Проверка.
		qP := r.URL.Query()

		RxNumbReg := qP.Get("numbReg")
		if RxNumbReg == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		numbReq, err := strconv.Atoi(RxNumbReg)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		rxStrLimit := qP.Get("strLimit")
		if rxStrLimit == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		limit, err := strconv.Atoi(rxStrLimit)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		rxStrOffSet := qP.Get("strOffSet")
		if rxStrOffSet == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		OffSet, err := strconv.Atoi(rxStrOffSet)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Формирование данных для ответа
		rdDataDB := make([]DataEl, 0)
		for i := OffSet; i < limit; i++ {
			rdDataDB = append(rdDataDB, simDataDB[i])
		}

		// Ответ
		dataForTx := PartDataDB{
			NumbReq: numbReq,
			Data:    rdDataDB,
		}

		txByte, err := json.Marshal(dataForTx)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(txByte)
	}))

	rxData, err := ReqPartDataDB(numbReq, strLimit, strOffSet, dateDB, token, name, server.URL, server.Client())
	require.NoErrorf(t, err, "ожидалось отсутствие ошибки, а принято: {%v}", err)
	assert.Equalf(t, numbReq, rxData.NumbReq, "ожидался номер запроса: {%d}, а принят: {%d}", numbReq, rxData.NumbReq)

	for i, v := range rxData.Data {
		strRx := fmt.Sprintf("%s  %s  %s  %s", v.Name, v.Value, v.Qual, v.TimeStamp)
		strData := fmt.Sprintf("%s  %s  %s  %s", simDataDB[i+strOffSet].Name, simDataDB[i+strOffSet].Value, simDataDB[i+strOffSet].Qual, simDataDB[i+strOffSet].TimeStamp)
		assert.Equalf(t, strData, strRx, "ожидалась строка: {%s}, а принята {%s}", strData, strRx)
	}
}

// Запрос архивных данных сервера - ошибки
func Test_ReqPartDataDB_Error(t *testing.T) {

	userName := "test"

	argData := []struct {
		nameTest  string
		numbReq   int
		strLimit  int
		strOffSet int
		dateDB    string
		token     string
		name      string
		useURL    string
		useClient string
		wantError string
	}{
		{
			nameTest:  "номер запроса меньше нуля",
			numbReq:   -1,
			strLimit:  10,
			strOffSet: 0,
			dateDB:    "2025-01-02",
			token:     "1234567890",
			name:      "test",
			useURL:    "true",
			useClient: "true",
			wantError: "req-partdatadb -> значение аргумента numbReg, меньше нуля",
		},
		{
			nameTest:  "лимита запроса меньше нуля",
			numbReq:   0,
			strLimit:  -1,
			strOffSet: 0,
			dateDB:    "2025-01-02",
			token:     "1234567890",
			name:      "test",
			useURL:    "true",
			useClient: "true",
			wantError: "req-partdatadb -> значение аргумента strLimit, меньше нуля",
		},
		{
			nameTest:  "смещение запроса меньше нуля",
			numbReq:   0,
			strLimit:  10,
			strOffSet: -1,
			dateDB:    "2025-01-02",
			token:     "1234567890",
			name:      "test",
			useURL:    "true",
			useClient: "true",
			wantError: "req-partdatadb -> значение аргумента strOffSet, меньше нуля",
		},
		{
			nameTest:  "пустое значение даты",
			numbReq:   0,
			strLimit:  10,
			strOffSet: 0,
			dateDB:    "",
			token:     "1234567890",
			name:      "test",
			useURL:    "true",
			useClient: "true",
			wantError: "req-partdatadb -> пустое значение даты",
		},
		{
			nameTest:  "значение даты не в формате YYYY-MM-DD",
			numbReq:   0,
			strLimit:  10,
			strOffSet: 0,
			dateDB:    "02-01-2025",
			token:     "1234567890",
			name:      "test",
			useURL:    "true",
			useClient: "true",
			wantError: "req-partdatadb -> значение даты не в формате YYYY-MM-DD",
		},
		{
			nameTest:  "пустое значение токена",
			numbReq:   0,
			strLimit:  10,
			strOffSet: 0,
			dateDB:    "2025-01-02",
			token:     "",
			name:      "test",
			useURL:    "true",
			useClient: "true",
			wantError: "req-partdatadb -> пустое значение токена",
		},
		{
			nameTest:  "пустое значение имени",
			numbReq:   0,
			strLimit:  10,
			strOffSet: 0,
			dateDB:    "2025-01-02",
			token:     "1234567890",
			name:      "",
			useURL:    "true",
			useClient: "true",
			wantError: "req-partdatadb -> пустое значение имени",
		},
		{
			nameTest:  "пустое значение URL",
			numbReq:   0,
			strLimit:  10,
			strOffSet: 0,
			dateDB:    "2025-01-02",
			token:     "1234567890",
			name:      "test",
			useURL:    "false",
			useClient: "true",
			wantError: "req-partdatadb -> пустое содержимое URL",
		},
		{
			nameTest:  "нет указателя на http клиент",
			numbReq:   0,
			strLimit:  10,
			strOffSet: 0,
			dateDB:    "2025-01-02",
			token:     "1234567890",
			name:      "test",
			useURL:    "true",
			useClient: "false",
			wantError: "req-partdatadb -> нет указателя на https клиент",
		},
		{
			nameTest:  "нет пользователя в БД",
			numbReq:   0,
			strLimit:  10,
			strOffSet: 0,
			dateDB:    "2025-01-02",
			token:     "1234567890",
			name:      "test2",
			useURL:    "true",
			useClient: "true",
			wantError: "req-partdatadb -> сервер не вернул код 200",
		},
	}

	// Логика
	for _, tt := range argData {
		t.Run(tt.nameTest, func(t *testing.T) {

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				w.Header().Set("Content-Type", "application-json")

				// Чтение заголовков
				token := r.Header.Get("authorization")
				if token == "" {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}

				// Тело запроса
				var reqBody DateNameT

				bytesBody, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				defer r.Body.Close()

				err = json.Unmarshal(bytesBody, &reqBody)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				// Проверка имени
				if reqBody.Name != userName {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}

				// Ответ (данных нет)
				dataForTx := PartDataDB{}

				txByte, err := json.Marshal(dataForTx)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write(txByte)

			}))
			defer server.Close()

			u := server.URL
			client := server.Client()
			if tt.useURL == "false" {
				u = ""
			}
			if tt.useClient == "false" {
				client = nil
			}

			_, err := ReqPartDataDB(tt.numbReq, tt.strLimit, tt.strOffSet, tt.dateDB, tt.token, tt.name, u, client)
			errRx := fmt.Sprintf("%v", err)
			assert.Equal(t, tt.wantError, errRx, "ожидалась ошибка:{%s}, а принята {%s}", tt.wantError, errRx)
		})
	}

}

// Очередь запросов на загрузку архивных данных
func Test_QueReqPartDataDB_Success(t *testing.T) {
	usrName := "test"
	usrToken := "1234567890"

	// Имитация набора архивных данных БД (3600 записей)
	simDataDB3600 := make([]DataEl, 0)
	for i := 0; i < 60; i++ {
		for ii := 0; ii < 60; ii++ {
			tStamp := fmt.Sprintf("2025-05-18T03:%d:%d.391321+07:00", i, ii)
			str := DataEl{
				Name:      "Dev3600. HR. Тестовая переменная ShortInt",
				Value:     strconv.Itoa(i * ii),
				Qual:      "1",
				TimeStamp: tStamp,
			}
			simDataDB3600 = append(simDataDB3600, str)
		}
	}

	// Данные для тесов
	argData := []struct {
		nameTest   string
		startDate  string
		token      string
		name       string
		cntStr     int
		wantRxSize int
	}{
		{
			nameTest:   "количество строк = 0",
			startDate:  "2025-01-01",
			token:      usrToken,
			name:       usrName,
			cntStr:     0,
			wantRxSize: 0,
		},
		{
			nameTest:   "количество строк = 100",
			startDate:  "2025-01-01",
			token:      usrToken,
			name:       usrName,
			cntStr:     100,
			wantRxSize: 100,
		},
		{
			nameTest:   "количество строк = 1000",
			startDate:  "2025-01-01",
			token:      usrToken,
			name:       usrName,
			cntStr:     1000,
			wantRxSize: 1000,
		},
	}

	// Сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application-json")

		// Чтение параметров запроса. Проверка.
		qP := r.URL.Query()

		RxNumbReg := qP.Get("numbReg")
		if RxNumbReg == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		numbReq, err := strconv.Atoi(RxNumbReg)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		rxStrLimit := qP.Get("strLimit")
		if rxStrLimit == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		limit, err := strconv.Atoi(rxStrLimit)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		rxStrOffSet := qP.Get("strOffSet")
		if rxStrOffSet == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		OffSet, err := strconv.Atoi(rxStrOffSet)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Формирование данных для ответа
		rdDataDB := make([]DataEl, 0)
		for i := OffSet; i < len(simDataDB3600); i++ {

			if i < (OffSet + limit) {
				rdDataDB = append(rdDataDB, simDataDB3600[i])
				continue
			}
			break
		}

		// Ответ
		dataForTx := PartDataDB{
			NumbReq: numbReq,
			Data:    rdDataDB,
		}

		txByte, err := json.Marshal(dataForTx)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(txByte)
	}))

	for _, tt := range argData {
		rxData, err := QueReqPartDataDB(tt.startDate, tt.token, tt.name, server.URL, tt.cntStr, server.Client())
		require.NoErrorf(t, err, "ожидалось отутствие ошибки, а принято: {%s}", fmt.Sprintf("%s", err))

		saveData := make([]DataEl, 0)
		for _, v := range rxData {
			saveData = append(saveData, v.Data...)
		}
		rxCnt := len(saveData)
		assert.Equalf(t, tt.wantRxSize, rxCnt, "ожидалось %d записей, а принято: %d", tt.wantRxSize, rxCnt)
	}
}

// Очередь запросов на загрузку архивных данных (если строк меньше 100)
func Test_QueReqPartDataDB_Error(t *testing.T) {

	usrName := "test"
	usrToken := "1234567890"

	// Имитация набора архивных данных БД (60 записей)
	simDataDB60 := make([]DataEl, 0)
	for i := 0; i < 60; i++ {
		tStamp := fmt.Sprintf("2025-05-18T03:00:%d.391321+07:00", i)
		str := DataEl{
			Name:      "Dev60. HR. Тестовая переменная ShortInt",
			Value:     strconv.Itoa(i),
			Qual:      "1",
			TimeStamp: tStamp,
		}
		simDataDB60 = append(simDataDB60, str)
	}
	sizeDB60 := len(simDataDB60)

	// Имитация набора архивных данных БД (3600 записей)
	simDataDB3600 := make([]DataEl, 0)
	for i := 0; i < 60; i++ {
		for ii := 0; ii < 60; ii++ {
			tStamp := fmt.Sprintf("2025-05-18T03:%d:%d.391321+07:00", i, ii)
			str := DataEl{
				Name:      "Dev3600. HR. Тестовая переменная ShortInt",
				Value:     strconv.Itoa(i * ii),
				Qual:      "1",
				TimeStamp: tStamp,
			}
			simDataDB3600 = append(simDataDB3600, str)
		}
	}
	sizeDB3600 := len(simDataDB3600)
	_ = sizeDB3600

	// Данные для тесов
	argData := []struct {
		nameTest  string
		startDate string
		token     string
		name      string
		cntStr    int
		useURL    string
		useClient string
		simData   []DataEl
		wantErr   string
	}{
		{
			nameTest:  "пустое значение даты",
			startDate: "",
			token:     usrToken,
			name:      usrName,
			cntStr:    sizeDB60,
			useURL:    "true",
			useClient: "true",
			simData:   simDataDB60,
			wantErr:   "queReq -> пустое значение даты",
		},
		{
			nameTest:  "значение даты не в формате YYYY-MM-DD",
			startDate: "02-01-2025",
			token:     usrToken,
			name:      usrName,
			cntStr:    sizeDB60,
			useURL:    "true",
			useClient: "true",
			simData:   simDataDB60,
			wantErr:   "queReq -> значение даты не в формате YYYY-MM-DD",
		},
		{
			nameTest:  "пустое значение token",
			startDate: "2025-01-02",
			token:     "",
			name:      usrName,
			cntStr:    sizeDB60,
			useURL:    "true",
			useClient: "true",
			simData:   simDataDB60,
			wantErr:   "queReq -> пустое значение token",
		},
		{
			nameTest:  "пустое значение name",
			startDate: "2025-01-02",
			token:     usrToken,
			name:      "",
			cntStr:    sizeDB60,
			useURL:    "true",
			useClient: "true",
			simData:   simDataDB60,
			wantErr:   "queReq -> пустое значение name",
		},
		{
			nameTest:  "пустое значение URL",
			startDate: "2025-01-02",
			token:     usrToken,
			name:      usrName,
			cntStr:    sizeDB60,
			useURL:    "false",
			useClient: "true",
			simData:   simDataDB60,
			wantErr:   "queReq -> пустое значение URL",
		},
		{
			nameTest:  "в количестве строк отрицательное число",
			startDate: "2025-01-02",
			token:     usrToken,
			name:      usrName,
			cntStr:    -1,
			useURL:    "true",
			useClient: "true",
			simData:   simDataDB60,
			wantErr:   "queReq -> в количестве строк отрицательное число",
		},
		{
			nameTest:  "нет указателя на https клиента",
			startDate: "2025-01-02",
			token:     usrToken,
			name:      usrName,
			cntStr:    0,
			useURL:    "true",
			useClient: "false",
			simData:   simDataDB60,
			wantErr:   "queReq -> нет указателя на https клиента",
		},
	}

	// Сервер
	simDataDB := make([]DataEl, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application-json")

		// Чтение параметров запроса. Проверка.
		qP := r.URL.Query()

		RxNumbReg := qP.Get("numbReg")
		if RxNumbReg == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		numbReq, err := strconv.Atoi(RxNumbReg)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		rxStrLimit := qP.Get("strLimit")
		if rxStrLimit == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		limit, err := strconv.Atoi(rxStrLimit)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		rxStrOffSet := qP.Get("strOffSet")
		if rxStrOffSet == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		OffSet, err := strconv.Atoi(rxStrOffSet)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Формирование данных для ответа
		rdDataDB := make([]DataEl, 0)
		for i := OffSet; i < len(simDataDB); i++ {

			if i < (OffSet + limit) {
				rdDataDB = append(rdDataDB, simDataDB[i])
				continue
			}
			break
		}

		// Ответ
		dataForTx := PartDataDB{
			NumbReq: numbReq,
			Data:    rdDataDB,
		}

		txByte, err := json.Marshal(dataForTx)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(txByte)
	}))

	// Тесты
	for _, tt := range argData {
		t.Run(tt.nameTest, func(t *testing.T) {

			simDataDB = tt.simData

			u := server.URL
			if tt.useURL == "false" {
				u = ""
			}
			client := server.Client()
			if tt.useClient == "false" {
				client = nil
			}
			_, err := QueReqPartDataDB(tt.startDate, tt.token, tt.name, u, tt.cntStr, client)
			rxErr := fmt.Sprintf("%v", err)
			assert.Equalf(t, tt.wantErr, rxErr, "ожидалась ошибка: {%s}, а принята: {%s}", tt.wantErr, rxErr)
		})
	}

}
