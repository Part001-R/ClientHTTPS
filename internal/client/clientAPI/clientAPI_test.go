package clientapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тест успешности регистрации на сервере
func Test_ReqLoginServerSuccess(t *testing.T) {

	// Подготовка данных
	name := "userTest"
	password := "123"
	token := "1234567890"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application-json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(token))
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

// Проверка формирования ошибок при регистрации на сервере
func Test_ReqLoginServerError(t *testing.T) {

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
