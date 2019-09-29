package memory

import (
	"database/sql"
	"github.com/labstack/echo"
	"github.com/scylladb/go-set/strset"
	dialogs "github.com/sherbakovAE/yandex-dialogs"
	"github.com/sherbakovAE/yandex-dialogs-server/logging"
	lv "github.com/texttheater/golang-levenshtein/levenshtein"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var log = logging.GetInstance()

type Question = dialogs.Question
type Answer = dialogs.Answer
type Filter = dialogs.Filter

const (
	All   = -1
	Start = iota
	Size
	Continue
	Check
)

// генератор  случайных чисел
func RangeInt(min int, max int) int {
	return rand.Intn(max) + min

}

var redigit = regexp.MustCompile(`(?m)\d+`)

func GetNumber(answerString string) (int, error) {
	result, err := strconv.Atoi(redigit.FindString(answerString))
	if err != nil {
		log.Println("Ошибка %s", err)
		return -1, err
	}

	return result, nil
}

func VerifyWords(answerString string, quest *strset.Set) *strset.Set {

	delimiter := regexp.MustCompile(`[\s\,\.]+`)

	// преобразование ответа в набор слов
	words := delimiter.Split(answerString, -1)
	result := 0
	for _, sword := range quest.List() {
		for _, word := range words {
			if lv.DistanceForStrings([]rune(word), []rune(sword), lv.DefaultOptions) < 2 {
				result++
				quest.Remove(sword)
			}
		}
	}
	return quest

}

type Talk struct {
	quest     *strset.Set // слова для запоминания
	size      int         // число слов для запоминания
	game      int         // сколько раундов сыграно
	result    int         // забытых слов (всего )
	timeStart time.Time   // время последнего доступа
	/*
		1 Старт (приветствие, вопрос размера набора слов)
		2 Загадка набора слов -> выпилено (в составе других шагов)
		3 Вопрос продолжения
		4 Оценка результата, вопрос увеличения - уменьшения набора слов
		5 Увеличение- уменьшение размера набора слов по результату
	*/
}

func GetWords(db *sql.DB, size int) (*strset.Set, error) {

	words := strset.New()
	inds := strset.New()
	rand.Seed(time.Now().Unix())

	// создание набора индексов слов
	for inds.Size() < size {
		inds.Add(strconv.Itoa(RangeInt(1, 3000)))
	}
	sqlc := `select word from words where type = ` + strconv.Itoa(RangeInt(1, 3)) +
		`  ORDER BY random() LIMIT (` + strconv.Itoa(size) + `)`

	rows, err := db.Query(sqlc)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var w string
		err = rows.Scan(&w)
		if err != nil {
			return nil, err
		}
		w = strings.TrimSpace(w)
		words.Add(w)
	}
	return words, nil
}

func checkNewSession(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	if question.Session.New == true {
		p.Storage.SetState(question.Session.UserID, Start)

		return false, nil
	} else {
		return false, nil
	}
}

func welcome(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {

	answer.Response.Text += "В этой игре нужно повторить произнесённые слова в любом порядке." +
		" Выберите количество запоминаемых слов от одного до девяти"
	p.Storage.SetState(question.Session.UserID, Size)
	return true, nil

}

func start(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {

	words, err := GetNumber(strings.ToLower(question.Request.Command))
	if err != nil {
		return true, nil
	}

	// если в запросе не содержится требуемое число, повторяем вопрос
	if words < 2 || words > 10 {
		answer.Response.Text += "Сколько слов вы готовы запомнить? Назовите число от 2 до 10"

		return true, nil
	} else {

		talk := Talk{timeStart: time.Now(), size: words}
		talk.quest, err = GetWords(p.DB, words)
		if err != nil {
			log.Errorln("Произошла ошибка %s", err)
			answer.Response.Text = "Извините. Произошла ошибка на сервере. Поиграем позже."
			answer.Response.EndSession = true // завершаем сессию
		}
		p.Storage.SetData(question.Session.UserID, talk)
		p.Storage.SetState(question.Session.UserID, Check)
		answer.Response.Text += strings.Join(talk.quest.List(), `,`)
		answer.Response.TTS = strings.Join(talk.quest.List(), `- - - - - - - - - - `)
		return true, nil
	}
}

// проверка ответа (повторение заданного набора слов)
func check(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	var result *strset.Set
	talk := p.Storage.GetData(question.Session.UserID).(Talk)
	result = VerifyWords(strings.ToLower(question.Request.Command), talk.quest)
	talk.result += result.Size()
	talk.quest = strset.New()
	talk.game++

	if result.Size() == 0 {
		answer.Response.Text = "Все угадано. Продолжим?"
	} else {
		if result.Size() == 1 {
			answer.Response.Text = "Вы забыли слово "
		} else {
			answer.Response.Text = "Вы забыли слова "
		}
		answer.Response.Text += strings.Trim(result.String(), "[]") + ". Продолжим?"
	}
	p.Storage.SetState(question.Session.UserID, Continue)
	return true, nil

}
func endRound(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {

	talk := p.Storage.GetData(question.Session.UserID).(Talk)
	switch strings.ToLower(question.Request.Command) {
	case "да", "конечно", "давай":
		// сбрасываем тестовый набор и начинаем по новой
		talk.quest, err = GetWords(p.DB, talk.size)
		answer.Response.Text += strings.Join(talk.quest.List(), `,`)
		answer.Response.TTS = strings.Join(talk.quest.List(), `- - - - - - - - - - `)
		p.Storage.SetData(question.Session.UserID, talk)
		p.Storage.SetState(question.Session.UserID, Check)
		return true, nil

	case "нет", "хватит", "всё":
		answer.Response.Text = "До встречи в следующий раз."
		answer.Response.EndSession = true // заканчиваем сессию
		return true, nil

	default: // Это что-то совсем иное - выражение непонимания.
		answer.Response.Text = "Простите, я не поняла. Продолжать? Да или нет?"
		return true, nil
	}
}

func Run() (func(c echo.Context) error, dialogs.Pipeline) {

	var handler func(c echo.Context) error
	var pipeline dialogs.Pipeline
	pipeline.Storage = dialogs.NewMemoryStorage()

	pipeline.Questions, pipeline.Answers, handler = dialogs.New()

	//запуск горутин обработки вопросов и ответов

	pipeline.Register(checkNewSession, Filter{All, []string{}, ""})
	pipeline.Register(welcome, Filter{Start, []string{}, ""})
	pipeline.Register(check, Filter{Check, []string{}, ""})
	pipeline.Register(start, Filter{Size, []string{}, ""})
	pipeline.Register(endRound, Filter{Continue, []string{}, ""})
	return handler, pipeline
}
