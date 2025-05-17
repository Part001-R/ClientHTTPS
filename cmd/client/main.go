package main

import (
	clientapi "clienthttps/internal/client/clientAPI"
	"clienthttps/internal/client/libre"
	"errors"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
	"golang.org/x/term"
)

func main() {
	var user clientapi.UserLogin

	prepare(&user)
	run(&user)
}

// Подготовительные действия
func prepare(usr *clientapi.UserLogin) {

	// Чтение переменных окружения
	err := godotenv.Load("./configs/.env")
	if err != nil {
		log.Fatal("ошибка чтения переменных окружения:", err)
	}

	// Ввод данных пользователя при запуске приложения
	typeUserData(usr)

	// Регистрация на сервере и получение токена
	err = usr.LoginHttpsServer()
	if err != nil {
		log.Fatal("Ошибка регистрации на сервере: ", err)
	}
	fmt.Println("Регистрация пользователя выполнена")
	fmt.Println()

}

// Вывод меню действия
func run(usr *clientapi.UserLogin) {
	var str string

	for {
		fmt.Println("---------------------------")
		fmt.Println("1: Вывод информации сервера")
		fmt.Println("2: Запрос архивных данных")
		fmt.Println("3: Завершение работы")
		fmt.Print("Выбор действия-> ")
		_, err := fmt.Scanln(&str)
		if err != nil {
			log.Fatal("Ошибка ввода данных")
		}
		fmt.Println("---------------------------")

		switch str {
		case "1": // Вывод статусной информации сервера
			err := showStatusServer(usr)
			if err != nil {
				fmt.Println("Ошибка:", err)
				fmt.Println("Работа прервана")
				return
			}
			continue

		case "2": // Запрос архивных данных
			fmt.Println()
			fmt.Print("Введите дату экспорта (YYYY-MM-DD): ")
			fmt.Scanln(&str)

			err := expDataDB(str, usr)
			if err != nil {
				fmt.Println("Ошибка при экспорте данных из БД", err)
				fmt.Println("Работа прервана")
				return
			}
			fmt.Println("Экспорт данных выполнен")
			fmt.Println()
			continue

		case "3": // Завершение работы
			return

		default: // Ошибка ввода пользователя
			fmt.Println("Ошибка ввода. Работа завершена")
			return
		}
	}

}

// Запрос состояния сервера и вывод в терминал. Функция возвращает ошибку.
func showStatusServer(usr *clientapi.UserLogin) error {

	statusSrv := clientapi.RxStatusSrv{}

	err := statusSrv.ReqStatusServer(usr.Token, usr.Name)
	if err != nil {
		return fmt.Errorf("ошибка при запросе состояния сервера: %v", err)
	}

	fmt.Println()
	fmt.Println("Время запуска сервера :", statusSrv.TimeStart)
	fmt.Println()

	fmt.Println("Интерфейсов Modbus-RTU:", len(statusSrv.MbRTU))
	fmt.Println("Интерфейсов Modbus-TCP:", len(statusSrv.MbTCP))
	fmt.Println()

	for i, v := range statusSrv.MbRTU {
		fmt.Printf("Интерфейс Modbus-RTU {%d}\n", i+1)
		fmt.Println("Имя :", v.ConName)
		fmt.Println("Порт:", v.Con)
		fmt.Println("Параметры:", v.ConParams)
	}
	fmt.Println()

	for i, v := range statusSrv.MbTCP {
		fmt.Printf("Интерфейс Modbus-TCP {%d}\n", i+1)
		fmt.Println("Имя :", v.ConName)
		fmt.Println("Порт:", v.Con)
	}
	fmt.Println()

	fmt.Printf("Размер в МБ файла логирования - Информация    :{%d}\n", statusSrv.SizeF.I)
	fmt.Printf("Размер в МБ файла логирования - Предупреждение:{%d}\n", statusSrv.SizeF.W)
	fmt.Printf("Размер в МБ файла логирования - Ошибки        :{%d}\n", statusSrv.SizeF.E)
	fmt.Println()

	return nil
}

// Запрос архивных данных БД. Функция возвращает ошибку
func expDataDB(startDate string, usr *clientapi.UserLogin) error {

	// Проверка корректности ввода даты
	t, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("ошибка ввода даты: {%s}", t)
	}

	var dataDB clientapi.RxDataDB

	dataDB.StartDate = startDate

	// Запрос архивных данных БД
	err = dataDB.ReqDataDB(usr.Token, usr.Name)
	if err != nil {
		return fmt.Errorf("ошибка запроса архивных данных ДБ: {%v}", err)
	}

	// Формирование Exlx файла данных
	err = saveDataXlsx(dataDB)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении данных в xlsx файл: {%v}", err)
	}

	return nil
}

// Функция зодаёт xlsx файл и сохраняет туда принятые данные от сервера. Возвращает ошибку.
func saveDataXlsx(data clientapi.RxDataDB) (err error) {

	tn := time.Now().Format("02.01.2006-15:04:05")

	// Создание файла
	fName := fmt.Sprintf("exportData:%s------------", data.StartDate)

	fileName, err := libre.CreateXlsx("./", fName, tn, ".xlsx")
	if err != nil {
		return fmt.Errorf("ошибка при создании xlsx файла экспорта: {%v}", err)
	}

	// Открытие файла
	file, err := excelize.OpenFile(fileName)
	if err != nil {
		return fmt.Errorf("ошибка при открытии файла: {%v}", fileName)
	}

	// Заполнение файла

	nameSheet := "DataDB"

	// Формирование заголовков
	// Name: Value:	Quality: TimeStamp:
	err = file.SetCellValue(nameSheet, "A1", "Name:")
	if err != nil {
		return errors.New("ошибка при добавлении заголовка столбца Name")
	}
	err = file.SetCellValue(nameSheet, "B1", "Value:")
	if err != nil {
		return errors.New("ошибка при добавлении заголовка столбца Value")
	}
	err = file.SetCellValue(nameSheet, "C1", "Quality:")
	if err != nil {
		return errors.New("ошибка при добавлении заголовка столбца Quality")
	}
	err = file.SetCellValue(nameSheet, "D1", "TimeStamp:")
	if err != nil {
		return errors.New("ошибка при добавлении заголовка столбца TimeStamp")
	}

	// Перенос данных
	for i, str := range data.Data {

		i++

		err = file.SetCellValue(nameSheet, fmt.Sprintf("A%d", i), str.Name)
		if err != nil {
			return fmt.Errorf("ошибка добавления значения {%s} в ячейку {A%d}", str.Name, i)
		}
		err = file.SetCellValue(nameSheet, fmt.Sprintf("B%d", i), str.Value)
		if err != nil {
			return fmt.Errorf("ошибка добавления значения {%s} в ячейку {B%d}", str.Value, i)
		}
		err = file.SetCellValue(nameSheet, fmt.Sprintf("C%d", i), str.Qual)
		if err != nil {
			return fmt.Errorf("ошибка добавления значения {%s} в ячейку {C%d}", str.Qual, i)
		}
		err = file.SetCellValue(nameSheet, fmt.Sprintf("D%d", i), str.TimeStamp)
		if err != nil {
			return fmt.Errorf("ошибка добавления значения {%s} в ячейку {D%d}", str.TimeStamp, i)
		}
	}

	// Сохрангение
	err = file.Save()
	if err != nil {
		return errors.New("ошибка при сохранении Xlsx файла")
	}

	return nil
}

// Ввод данных пользователя при запуске приложения.
//
// Параметры:
//
// user - указатель на данные пользователя
func typeUserData(usr *clientapi.UserLogin) {

	fd := int(syscall.Stdin)

	fmt.Println("Необходима регистрация на сервере.")
	fmt.Print("Имя пользователя: ")
	data, err := term.ReadPassword(fd)
	if err != nil {
		log.Fatal(err)
	}
	usr.Name = string(data)
	fmt.Println()

	fmt.Print("Пароль пользователя: ")
	data, err = term.ReadPassword(fd)
	if err != nil {
		log.Fatal(err)
	}
	usr.Password = string(data)

	fmt.Println()
	fmt.Println()
}
