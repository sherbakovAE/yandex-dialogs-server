package aecho

import (
	"github.com/labstack/echo"
	dialogs "github.com/sherbakovAE/yandex-dialogs"
	"github.com/sherbakovAE/yandex-dialogs-server/logging"
)

type Question = dialogs.Question
type Answer = dialogs.Answer
type Filter = dialogs.Filter

const (
	Start dialogs.State = iota
	Continue
)

var log = logging.GetInstance()

func Echo(question Question, answer *Answer, _ *dialogs.Pipeline) (finish bool, err error) {
	answer.Response.Text = question.Request.Command
	return true, nil
}
func InitEcho(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	// проверка наличия сохраненной сессии, при отсутсвии притворяемся новой
	if question.Session.New == true {
		answer.Response.Text = "Привет"
		p.Storage.SetState(question.Session.UserID, Continue)
		return true, nil
	} else {
		return false, nil
	}
}

func Run() (func(c echo.Context) error, dialogs.Pipeline) {

	var handler func(c echo.Context) error
	var pipeline dialogs.Pipeline
	pipeline.Storage = dialogs.NewMemoryStorage()

	pipeline.Questions, pipeline.Answers, handler = dialogs.New()

	//запуск горутин обработки вопросов и ответов

	pipeline.Register(Echo, Filter{Continue, []string{}, ""})
	pipeline.Register(InitEcho, Filter{Start, []string{}, ""})
	return handler, pipeline
}
