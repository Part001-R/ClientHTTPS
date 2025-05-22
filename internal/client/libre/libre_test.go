package libre

import (
	clientapi "clienthttps/internal/client/clientAPI"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// Создание файла (успешность)
func Test_createXlsx_Success(t *testing.T) {

	path := "./"
	name := "testFile"
	time := time.Now().Format("02.01.2006-15:04:05")
	typ := ".xlsx"

	wantName := path + name + "-" + time + typ

	nameFile, err := createXlsx(path, name, time, typ)
	require.NoErrorf(t, err, "создание файла - ожидалось отсутствие ошибки, а принята: {%s}", fmt.Sprintf("%v", err))
	assert.Equalf(t, wantName, nameFile, "ожидалось имя файла: {%s}, а принято {%s}", wantName, nameFile)

	_, err = os.Stat(nameFile)
	require.NoErrorf(t, err, "проверка существоания файла - ожидалось отсутствие ошибки, а принято: {%s}", fmt.Sprintf("%s", err))

	err = os.Remove(nameFile)
	require.NoErrorf(t, err, "удаление файла - ожидалось отсутствие ошибки, а принято: {%s}", fmt.Sprintf("%s", err))
}

// Создание файла (ошибки)
func Test_createXlsx_Error(t *testing.T) {

	argData := []struct {
		testName string
		path     string
		name     string
		time     string
		typ      string
		wantErr  string
	}{
		{
			testName: "пустое содержимое пути",
			path:     "",
			name:     "test",
			time:     "2025-01-01 12:12:12",
			typ:      ".xlsx",
			wantErr:  "создание xlsx -> нет содержимого в path",
		},
		{
			testName: "пустое содержимое имени",
			path:     "./",
			name:     "",
			time:     "2025-01-01 12:12:12",
			typ:      ".xlsx",
			wantErr:  "создание xlsx -> нет содержимого в name",
		},
		{
			testName: "нет содержимого в time",
			path:     "./",
			name:     "test",
			time:     "",
			typ:      ".xlsx",
			wantErr:  "создание xlsx -> нет содержимого в time",
		},
		{
			testName: "нет содержимого в typ",
			path:     "./",
			name:     "test",
			time:     "2025-01-01 12:12:12",
			typ:      "",
			wantErr:  "создание xlsx -> нет содержимого в typ",
		},
		{
			testName: "нет точки",
			path:     "./",
			name:     "test",
			time:     "2025-01-01 12:12:12",
			typ:      "xlsx",
			wantErr:  "создание xlsx -> нет точки",
		},
		{
			testName: "указан тип не xlsx",
			path:     "./",
			name:     "test",
			time:     "2025-01-01 12:12:12",
			typ:      ".txt",
			wantErr:  "создание xlsx -> указан тип не xlsx",
		},
	}

	for _, tt := range argData {
		t.Run(tt.testName, func(t *testing.T) {
			_, err := createXlsx(tt.path, tt.name, tt.time, tt.typ)
			rxErr := fmt.Sprintf("%v", err)
			assert.Equalf(t, tt.wantErr, rxErr, "ошидалась ошибка: {%s}, а принято: {%s}", tt.wantErr, rxErr)
		})
	}
}

// Сохранение данных в xlsx (успешность)
func Test_SaveDataXlsx_Success(t *testing.T) {

	simDataDB := clientapi.RxDataDB{
		StartDate: "2025-01-01",
		Data:      make([]clientapi.DataEl, 0),
	}

	for i := 0; i < 60; i++ {
		var el clientapi.DataEl

		el.Name = "DevTest. HR. Тестовая переменная ShortInt"
		el.Value = strconv.Itoa(i)
		el.Qual = "1"
		el.TimeStamp = fmt.Sprintf("2025-05-18T03:01:%d.391321+07:00", i)

		simDataDB.Data = append(simDataDB.Data, el)
	}

	// Сохранение данных в файл
	fileName, err := SaveDataXlsx(simDataDB)
	rxErr := fmt.Sprintf("%v", err)
	require.NoErrorf(t, err, "ожидалось отсутствие ошибки, а принято: {%s}", rxErr)

	// Открытие файла
	file, err := excelize.OpenFile(fileName)
	require.NoErrorf(t, err, "открытие созданного файла - ожидалось отсутствие ошибки, а принято {%s}", fmt.Sprintf("%v", err))

	defer func() {
		err := file.Close()
		require.NoErrorf(t, err, "закрытие файла перед выходом - ожидалось отсутствие ошибки, а принято: {%s}", fmt.Sprintf("%v", err))

		_, err = os.Stat(fileName)
		require.NoErrorf(t, err, "информация о файле перед выходом - ожидалось отсутствие ошибки, а принято: {%s}", fmt.Sprintf("%v", err))

		err = os.Remove(fileName)
		require.NoErrorf(t, err, "удаление файла перед выходом - ожидалось отсутствие ошибки, а принято: {%s}", fmt.Sprintf("%v", err))
	}()

	// Сравнение данных
	nameSheet := "DataDB"

	for i, v := range simDataDB.Data {
		name, err := file.GetCellValue(nameSheet, fmt.Sprintf("A%d", i+2))
		require.NoErrorf(t, err, "чтение name - ожидалось отсутствие ошибки, а принято: {%s}", fmt.Sprintf("%v", err))

		value, err := file.GetCellValue(nameSheet, fmt.Sprintf("B%d", i+2))
		require.NoErrorf(t, err, "чтение value - ожидалось отсутствие ошибки, а принято: {%s}", fmt.Sprintf("%v", err))

		qual, err := file.GetCellValue(nameSheet, fmt.Sprintf("C%d", i+2))
		require.NoErrorf(t, err, "чтение qual - ожидалось отсутствие ошибки, а принято: {%s}", fmt.Sprintf("%v", err))

		timeStamp, err := file.GetCellValue(nameSheet, fmt.Sprintf("D%d", i+2))
		require.NoErrorf(t, err, "чтение timeStamp - ожидалось отсутствие ошибки, а принято: {%s}", fmt.Sprintf("%v", err))

		assert.Equalf(t, v.Name, name, "нет соответствия в name. ожидалось: {%s}, а принято: {%s}", v.Name, name)
		assert.Equalf(t, v.Value, value, "нет соответствия в value. ожидалось: {%s}, а принято: {%s}", v.Value, value)
		assert.Equalf(t, v.Qual, qual, "нет соответствия в qual. ожидалось: {%s}, а принято: {%s}", v.Qual, qual)
		assert.Equalf(t, v.TimeStamp, timeStamp, "нет соответствия в timeStamp. ожидалось: {%s}, а принято: {%s}", v.TimeStamp, timeStamp)
	}

}

// Сохранение данных в xlsx (ошибки)
func Test_SaveDataXlsx_Error(t *testing.T) {

	// Подготовка данных
	argData := []struct {
		testName string
		simData  clientapi.RxDataDB
		wantErr  string
	}{
		{
			testName: "нет даты",
			simData:  clientapi.RxDataDB{StartDate: "", Data: make([]clientapi.DataEl, 0)},
			wantErr:  "save xlsx -> нет даты",
		},
	}

	// Сохранение данных в файл
	for _, tt := range argData {
		t.Run(tt.testName, func(t *testing.T) {

			_, err := SaveDataXlsx(tt.simData)
			rxErr := fmt.Sprintf("%v", err)
			assert.Equalf(t, tt.wantErr, rxErr, "ожидалась ошибка: {%s}, а принято: {%s}", tt.wantErr, rxErr)
		})
	}
}
