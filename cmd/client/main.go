package main

import (
	clientapi "clienthttps/internal/client/clientAPI"
	"clienthttps/internal/client/libre"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/joho/godotenv"
	"golang.org/x/term"
)

func main() {
	var user clientapi.UserLogin

	clientHttps, userInfo := prepare(&user)

	run(clientHttps, userInfo)
}

// Подготовительные действия. Возвращается указатель на http кдиент и данные пользователя.
//
// Параметры:
//
// usr - данные пользователя
func prepare(usr *clientapi.UserLogin) (client *http.Client, userInfo clientapi.UserLogin) {

	// Чтение переменных окружения
	err := godotenv.Load("./configs/.env")
	if err != nil {
		log.Fatalf("ошибка чтения переменных окружения: {%v}\n", err)
	}

	// Создание Https клиента
	client, err = clientapi.CreateHttpsClient()
	if err != nil {
		log.Fatalf("ошибка создания https клиента: {%v}\n", err)
	}

	// Ввод данных пользователя при запуске приложения
	err = typeUserData(usr)
	if err != nil {
		log.Fatalf("ошибка ввода данных при старте приложения: {%v}\n", err)
	}

	// Регистрация на сервере и получение токена
	u := "https://" + os.Getenv("HTTPS_SERVER_IP") + ":" + os.Getenv("HTTPS_SERVER_PORT") + "/registration"

	userInfo, err = clientapi.ReqLoginServer(usr.Name, usr.Password, u, client)
	if err != nil {
		log.Fatalf("Ошибка регистрации на сервере: {%v}\n", err)
	}
	fmt.Println("Регистрация пользователя выполнена")
	fmt.Println()

	return client, userInfo
}

// Вывод меню действия.
//
// Параметры:
//
// client - указатель на http клиента
func run(client *http.Client, usr clientapi.UserLogin) {
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

			// Запрос данных сервера
			u := fmt.Sprintf("https://%s:%s/status", os.Getenv("HTTPS_SERVER_IP"), os.Getenv("HTTPS_SERVER_PORT"))

			statusSrv, err := clientapi.ReqStatusServer(usr.Token, usr.Name, u, client)
			if err != nil {
				log.Fatalf("ошибка при запросе состояния сервера: {%v}\n", err)
			}

			// Отображение принятых данных
			err = showStatusServer(statusSrv)
			if err != nil {
				fmt.Println("Ошибка:", err)
				fmt.Println("Работа прервана")
				return
			}
			continue

		case "2": // Запрос архивных данных
			var date string
			fmt.Println()
			fmt.Print("Введите дату экспорта (YYYY-MM-DD): ")
			fmt.Scanln(&date)

			// Запрос количества строк по дате
			u := fmt.Sprintf("https://%s:%s/cntstr", os.Getenv("HTTPS_SERVER_IP"), os.Getenv("HTTPS_SERVER_PORT"))

			cntStr, err := clientapi.ReqCntStrByDateDB(usr.Token, usr.Name, date, u, client)
			if err != nil {
				fmt.Println("Ошибка: ", err)
				fmt.Println("Работа прервана")
				return
			}
			fmt.Printf("По дате {%s} содержится {%d} строк\n", date, cntStr)

			// Выполнение очереди запросов на получение строк
			u = fmt.Sprintf("https://%s:%s/partdatadb", os.Getenv("HTTPS_SERVER_IP"), os.Getenv("HTTPS_SERVER_PORT"))

			rxData, err := clientapi.QueReqPartDataDB(date, usr.Token, usr.Name, u, cntStr, client)
			if err != nil {
				fmt.Println("Ошибка: ", err)
				fmt.Println("Работа прервана")
				return
			}

			// Подготовка данных для сохранения
			forSave := clientapi.RxDataDB{
				StartDate: date,
				Data:      make([]clientapi.DataEl, 0),
			}
			for _, v := range rxData {
				forSave.Data = append(forSave.Data, v.Data...)
			}

			// Формирование Exlx файла данных
			fileName, err := libre.SaveDataXlsx(forSave)
			if err != nil {
				fmt.Printf("ошибка при сохранении данных в xlsx файл: {%v}", err)
				fmt.Println("Работа прервана")
				return
			}

			fmt.Printf("Задача выполнена. Создан файл - %s", fileName)
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
//
// statusSrv - сводная информация сервера
func showStatusServer(statusSrv clientapi.RxStatusSrv) error {

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

// Ввод данных пользователя при запуске приложения.
//
// Параметры:
//
// user - указатель на данные пользователя. Возвращается ошибка.
func typeUserData(usr *clientapi.UserLogin) error {

	fd := int(syscall.Stdin)

	fmt.Println("Необходима регистрация на сервере.")
	fmt.Print("Имя пользователя: ")
	data, err := term.ReadPassword(fd)
	if err != nil {
		return fmt.Errorf("ошибка при чтении имени: {%v}", err)
	}
	usr.Name = string(data)
	fmt.Println()

	fmt.Print("Пароль пользователя: ")
	data, err = term.ReadPassword(fd)
	if err != nil {
		return fmt.Errorf("ошибка при чтении пароля: {%v}", err)
	}
	usr.Password = string(data)

	fmt.Println()
	fmt.Println()
	return nil
}
