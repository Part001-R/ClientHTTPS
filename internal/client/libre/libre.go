package libre

import (
	clientapi "clienthttps/internal/client/clientAPI"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Создание xlsx файла. Возвращается имя файла и ошибка.
//
// Параметры:
//
// path - путь к файлу
// name - имя файла
// time - время создания файла
// typ - тип файла
func createXlsx(path, name, time, typ string) (nameFile string, err error) {

	// Проверка значений аргументов
	if path == "" {
		return "", errors.New("создание xlsx -> нет содержимого в path")
	}
	if name == "" {
		return "", errors.New("создание xlsx -> нет содержимого в name")
	}
	if time == "" {
		return "", errors.New("создание xlsx -> нет содержимого в time")
	}
	if typ == "" {
		return "", errors.New("создание xlsx -> нет содержимого в typ")
	}
	if !strings.Contains(typ, ".") {
		return "", errors.New("создание xlsx -> нет точки")
	}
	sl := strings.Split(typ, ".")
	if sl[1] != "xlsx" {
		return "", errors.New("создание xlsx -> указан тип не xlsx")
	}

	// Логика
	file := excelize.NewFile()

	_, err = file.NewSheet("DataDB")
	if err != nil {
		return "", errors.New("создание xlsx -> ошибка при добавлении вкладки DataDB")
	}
	err = file.DeleteSheet("Sheet1")
	if err != nil {
		return "", errors.New("создание xlsx -> ошибка при удалении вкладки Sheet1")
	}

	f := path + name + "-" + time + typ
	err = file.SaveAs(f)
	if err != nil {
		return "", fmt.Errorf("создание xlsx -> ошибка при сохранении: {%v} ", err)
	}
	return f, nil
}

// Функция зодаёт xlsx файл и сохраняет туда принятые данные от сервера. Возвращает ошибку.
//
// Параметры:
//
// data - данные для сахранения
// date - дата
func SaveDataXlsx(data clientapi.RxDataDB) (fileName string, err error) {

	// Проверка аргументов
	if data.StartDate == "" {
		return "", errors.New("save xlsx -> нет даты")
	}

	tn := time.Now().Format("02.01.2006-15:04:05")

	// Создание файла
	fName := fmt.Sprintf("exportData:%s------------", data.StartDate)

	fileName, err = createXlsx("./", fName, tn, ".xlsx")
	if err != nil {
		return "", fmt.Errorf("ошибка при создании xlsx файла экспорта: {%v}", err)
	}

	// Открытие файла
	file, err := excelize.OpenFile(fileName)
	if err != nil {
		return "", fmt.Errorf("ошибка при открытии файла: {%v}", fileName)
	}

	// Заполнение файла

	nameSheet := "DataDB"

	// Формирование заголовков
	// Name: Value:	Quality: TimeStamp:
	err = file.SetCellValue(nameSheet, "A1", "Name:")
	if err != nil {
		return "", errors.New("ошибка при добавлении заголовка столбца Name")
	}
	err = file.SetCellValue(nameSheet, "B1", "Value:")
	if err != nil {
		return "", errors.New("ошибка при добавлении заголовка столбца Value")
	}
	err = file.SetCellValue(nameSheet, "C1", "Quality:")
	if err != nil {
		return "", errors.New("ошибка при добавлении заголовка столбца Quality")
	}
	err = file.SetCellValue(nameSheet, "D1", "TimeStamp:")
	if err != nil {
		return "", errors.New("ошибка при добавлении заголовка столбца TimeStamp")
	}

	// Перенос данных
	for i, str := range data.Data {

		i++

		err = file.SetCellValue(nameSheet, fmt.Sprintf("A%d", i+1), str.Name)
		if err != nil {
			return "", fmt.Errorf("ошибка добавления значения {%s} в ячейку {A%d}", str.Name, i)
		}
		err = file.SetCellValue(nameSheet, fmt.Sprintf("B%d", i+1), str.Value)
		if err != nil {
			return "", fmt.Errorf("ошибка добавления значения {%s} в ячейку {B%d}", str.Value, i)
		}
		err = file.SetCellValue(nameSheet, fmt.Sprintf("C%d", i+1), str.Qual)
		if err != nil {
			return "", fmt.Errorf("ошибка добавления значения {%s} в ячейку {C%d}", str.Qual, i)
		}
		err = file.SetCellValue(nameSheet, fmt.Sprintf("D%d", i+1), str.TimeStamp)
		if err != nil {
			return "", fmt.Errorf("ошибка добавления значения {%s} в ячейку {D%d}", str.TimeStamp, i)
		}
	}

	// Сохрангение
	err = file.Save()
	if err != nil {
		return "", errors.New("ошибка при сохранении Xlsx файла")
	}

	return fileName, nil
}
