package matematica

import (
	"fmt"
	"github.com/labstack/echo"
	dialogs "github.com/sherbakovAE/yandex-dialogs"
	"github.com/sherbakovAE/yandex-dialogs-server/logging"
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
	Start dialogs.State = iota
	WaitLevel
	CreateTask
	Check
	End
	All dialogs.State = -1
)

type Operation int8

const (
	add = iota
	sub
	multi
	divide
)

var redigit = regexp.MustCompile(`(?m)\d+`)

// описание примера
type Task struct {
	numberOne int       // первый
	numberTwo int       // и второй
	operator  Operation // действие
	solution  int       // правильный ответ
}

func (t *Task) Create(level int) (err error) {

	t.numberOne, t.numberTwo, err = CreateNumbers(level, t.operator)
	if err != nil {
		return err
	}
	switch t.operator {
	case add:
		t.solution = t.numberOne + t.numberTwo
	case multi:
		t.solution = t.numberOne * t.numberTwo
	case sub:
		{
			t.numberOne = t.numberOne + t.numberTwo
			t.solution = t.numberOne - t.numberTwo
		}
	case divide:
		{
			t.numberOne = t.numberOne * t.numberTwo
			t.solution = t.numberOne / t.numberTwo
		}
	default:
		return fmt.Errorf("invalid operator %d", t.operator)
	}

	return nil
}
func (t *Task) ToString() string {
	return strings.Join([]string{strconv.Itoa(t.numberOne), Operator2TTS(t.operator), strconv.Itoa(t.numberTwo)}, " ")
}

// содержимое диалога для описания его состояния
type Dialog struct {
	level     int  // уровень сложности
	task      Task // пример с решением
	correct   bool // правильный ответ или нет
	game      int  // сколько раундов сыграно
	result    int  // правильно решенных примеров
	startTime time.Time
}

// создание примера
func (d *Dialog) CreateTask() error {
	d.task = Task{}
	if d.level == 0 {
		d.task.operator = Operation(RangeInt(0, 2))
	} else {
		d.task.operator = Operation(RangeInt(0, 4))
	}
	return d.task.Create(d.level)
}

// генератор списка случайных чисел
func RangeInt(min int, max int) int {
	return rand.Intn(max) + min

}
func GetNumber(answerString string) int {
	result, err := strconv.Atoi(redigit.FindString(answerString))
	if err != nil {
		log.Errorln("Ошибка %s", err)
		return -1
	}

	return result
}

// преобразователь операции в текст (строку)
func Operator2TTS(operator Operation) string {
	switch operator {
	case add:
		return "плюс"
	case sub:
		return "минус"
	case multi:
		return "умножить на"
	case divide:
		return "разделить на"
	}
	return ""
}

// создать два аргумента и ответ в зависимости от уровня и операции
// только для операций сложения и умножения
// вычитание и деление генерируются на основе сложения и деления
func CreateNumbers(level int, operator Operation) (number1, number2 int, err error) {

	switch operator {
	case add, sub:
		{
			switch level {
			case 0:
				return RangeInt(2, 9), RangeInt(2, 9), nil
			case 1:
				return RangeInt(11, 49), RangeInt(2, 9), nil
			case 2:
				return RangeInt(11, 99), RangeInt(11, 49), nil
			case 3:
				return RangeInt(11, 99), RangeInt(10, 99), nil
			case 4:
				return RangeInt(101, 199), RangeInt(10, 99), nil
			case 5:
				return RangeInt(101, 499), RangeInt(10, 499), nil
			default:
				return 0, 0, fmt.Errorf("invalid level %d", level)
			}
		}
	case divide, multi:
		{
			switch level {
			case 0, 1:
				return RangeInt(2, 9), RangeInt(2, 9), nil
			case 2:
				return RangeInt(2, 19), RangeInt(2, 9), nil
			case 3:
				return RangeInt(11, 49), RangeInt(2, 9), nil
			case 4:
				return RangeInt(11, 99), RangeInt(2, 19), nil
			case 5:
				return RangeInt(11, 99), RangeInt(11, 99), nil
			default:
				return 0, 0, fmt.Errorf("invalid level %d", level)
			}
		}
	}
	return 0, 0, fmt.Errorf("invalid operator %d", operator)
}

func start(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	// проверка наличия сохраненной сессии, при отсутсвии притворяемся новой

	//if question.Session.New == true {
	answer.Response.Text += "Это тренировка счета в уме. В течение двух минут нужно правильно решить как можно больше примеров." +
		" Выберите уровень сложности? Назовите число от 0 (лёгкий) до 5 (сложный)"
	p.Storage.SetState(question.Session.UserID, WaitLevel)
	p.Storage.SetData(question.Session.UserID, Dialog{level: -1, correct: false, startTime: time.Now()})
	return true, nil
	//} else {
	//	return false, nil
	//}
}
func errorLevel(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {

	answer.Response.Text = "Назовите число от 0 (лёгкий) до 5 (сложный)"
	p.Storage.SetState(question.Session.UserID, WaitLevel)
	return true, nil
}
func rightLevel(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {

	data := p.Storage.GetData(question.Session.UserID)
	var d Dialog
	if data == nil {
		d = Dialog{level: GetNumber(strings.ToLower(question.Request.Command)), startTime: time.Now()}
	} else {
		d = data.(Dialog)
		d.level = GetNumber(question.Request.Command)
	}

	p.Storage.SetState(question.Session.UserID, CreateTask)
	p.Storage.SetData(question.Session.UserID, d)

	return false, nil // перейти к созданию примера
}

func help(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	answer.Response.Text = "Нужно решать примеры и называть ответы. Начнем с начала."
	p.Storage.SetState(question.Session.UserID, Start)
	return false, nil
}

func task(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {

	data := p.Storage.GetData(question.Session.UserID)
	var d Dialog
	if data == nil {
		d = Dialog{level: GetNumber(strings.ToLower(question.Request.Command)), startTime: time.Now()}
	} else {
		d = data.(Dialog)
	}
	err = d.CreateTask()
	if err != nil {
		return true, err
	}
	answer.Response.Text = d.task.ToString()
	p.Storage.SetState(question.Session.UserID, Check)
	p.Storage.SetData(question.Session.UserID, d)
	return true, err
}

func check(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	d, ok := p.Storage.GetData(question.Session.UserID).(Dialog)
	if !ok {
		log.Errorln("Ошибка преобразования данных")
		return true, err
	}
	result, err := strconv.Atoi(redigit.FindString(question.Request.Command))
	if err != nil { // засчитывается как ошибка
		d.correct = false
	}
	if result != d.task.solution {
		d.correct = false
	} else {
		d.correct = true
	}

	if d.correct {
		d.game++
		d.result++
	} else {
		d.game++
	}
	if d.correct {
		answer.Response.Text = "Верно."
	} else {
		answer.Response.Text = "Вы ошиблись. Правильный ответ " + strconv.Itoa(d.task.solution) + ". "
	}
	// если время не закончилось, нужно создать новый пример
	// иначе подсчитать результат и дать оценку
	if time.Now().Sub(d.startTime) > time.Duration(2*time.Minute) {

		answer.Response.Text += fmt.Sprintf("Время вышло, правильных ответов %d из %d.", d.result, d.game)

		switch r := float32(d.result) / float32(d.game); {
		// хороши
		case r > 0.95:
			answer.Response.Text += "Отличный результат."
		case r < 0.8:
			answer.Response.Text += "Для вас это слишком сложно."
		case d.game < 5:
			answer.Response.Text += "Решено мало примеров."
		default:
			answer.Response.Text += "Хороший результат."
		}
		answer.Response.Text += "Продолжим?"
		p.Storage.SetState(question.Session.UserID, End)
	} else {
		err = d.CreateTask()
		if err != nil {
			log.Errorln(err)
		}

		answer.Response.Text += "Следующий пример. " + d.task.ToString()
		p.Storage.SetData(question.Session.UserID, d)
		p.Storage.SetState(question.Session.UserID, Check)

	}

	return false, nil

}

func restart(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	p.Storage.SetState(question.Session.UserID, Start)
	return false, nil
}
func end(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	answer.Response.Text = "До свидания. Заходите ещё."
	answer.Response.EndSession = true
	return true, nil
}

func Run() (func(c echo.Context) error, dialogs.Pipeline) {
	var handler func(c echo.Context) error
	var pipeline dialogs.Pipeline
	pipeline.Storage = dialogs.NewMemoryStorage()
	pipeline.Questions, pipeline.Answers, handler = dialogs.New()

	// регистрация функций в порядке обработки, в начале функции проверки ответов на вопросы,
	// чтобы при положительном исходе проваливаться ниже на обработку корректных ответов
	// либо сначала обработка корректных ответов, но с фильтром , а далее функция обработка ошибки
	// можно регистрировать одну и ту же функцию несколько раз в разных местах если ожидается смена состояния
	// функцией расположенной после

	// служебные ответы, например запрос о помощи
	pipeline.Register(help, Filter{All, []string{"помощь"}, ""})
	pipeline.Register(end, Filter{All, []string{"хватит", "выйти", "закончить", "всё"}, ""})

	// реакция на завершение набора примеров
	pipeline.Register(restart, Filter{End, []string{"ок", "хорошо", "да"}, ""})
	pipeline.Register(end, Filter{End, []string{}, ""})

	// приветсвие и запрос уровня сложности
	pipeline.Register(start, Filter{Start, []string{}, ""})

	// проверка уровня сложности и создание примера
	pipeline.Register(rightLevel, Filter{WaitLevel, []string{}, `^[0-5]{1}\b`})
	pipeline.Register(errorLevel, Filter{WaitLevel, []string{}, ``})
	pipeline.Register(task, Filter{CreateTask, []string{}, ``})

	// проверка ответа и переход к новому примеру или завершение испытания и запрос нового уровня
	pipeline.Register(check, Filter{Check, []string{}, ""})
	pipeline.Register(task, Filter{CreateTask, []string{}, ""})

	return handler, pipeline
}
