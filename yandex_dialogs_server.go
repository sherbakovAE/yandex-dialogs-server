package main

import (
	"database/sql"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	_ "github.com/mattn/go-sqlite3"
	dialogs "github.com/sherbakovAE/yandex-dialogs"
	"github.com/sherbakovAE/yandex-dialogs-server/logging"
	"github.com/sherbakovAE/yandex-dialogs-server/skills/aecho"
	"github.com/sherbakovAE/yandex-dialogs-server/skills/basket"
	mathem "github.com/sherbakovAE/yandex-dialogs-server/skills/matematica"
	"github.com/sherbakovAE/yandex-dialogs-server/skills/memory"

	"os"
)

var (
	answer dialogs.Answer
)

type State int

var db *sql.DB
var err error
var log = logging.GetInstance() //логгер навыков
// логгер сервера echo
var logConfig = middleware.LoggerConfig{Skipper: middleware.DefaultSkipper,
	Format: `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}","host":"${host}",` +
		`"method":"${method}","uri":"${uri}","status":${status}, "latency":${latency},` +
		`"latency_human":"${latency_human}","bytes_in":${bytes_in},` +
		`"bytes_out":${bytes_out}}` + "\n",
	Output: os.Stdout,
}

func main() {

	log.Debugf("Стартуем!..")

	// создание и запуск сервера
	e := echo.New()
	e.Use(middleware.LoggerWithConfig(logConfig))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST},
	}))

	//создать каналы вопросов и ответов для каждого навыка

	//// запуск навыка Эхо
	var handlerEcho func(c echo.Context) error
	handlerEcho, Echo := aecho.Run()

	go Echo.Start()
	e.POST("/echo", handlerEcho)

	//// запуск навыка "Счёт в уме"
	var handlerMath func(c echo.Context) error
	handlerMath, Mathem := mathem.Run()
	go Mathem.Start()
	e.POST("/math", handlerMath)

	//// запуск навыка "Счёт в уме"
	var handlerBasket func(c echo.Context) error
	handlerBasket, Basket := basket.Run()
	go Basket.Start()
	e.POST("/basket", handlerBasket)

	//// запуск навыка "Повторение слов"
	var handlerMemory func(c echo.Context) error
	handlerMemory, Memory := memory.Run()

	// определение используемой БД для использования в навыке
	db, err := sql.Open("sqlite3", "words.db")
	if err != nil {
		log.Fatal(err.Error())
	} else {
		Memory.DB = db
	}
	defer db.Close()

	go Memory.Start()
	e.POST("/memory", handlerMemory)

	e.Logger.Fatal(e.Start(":1323"))
}
