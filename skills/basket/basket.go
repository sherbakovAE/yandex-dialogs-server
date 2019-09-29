package basket

import (
	"github.com/labstack/echo"
	"github.com/scylladb/go-set/strset"
	dialogs "github.com/sherbakovAE/yandex-dialogs"
	"github.com/sherbakovAE/yandex-dialogs-server/logging"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

type Question = dialogs.Question
type Answer = dialogs.Answer
type Filter = dialogs.Filter

var minSize int = 3
var maxSize int = 7
var setsForBasket = [3]*strset.Set{strset.New("красный", "белый", "чёрный", "синий", "зелёный", "голубой", "оранжевый", "жёлтый", "фиолетовый"),
	strset.New("финик", "ананас", "манго", "авокадо", "банан", "персик", "абрикос", "апельсин", "мандарин"),
	strset.New("груздь", "подберёзовик", "подосиновик", "боровик", "рыжик", "опёнок", "моховик", "дождевик", "сморчок"),
}

type UserData struct {
	basket    *strset.Set
	numberSet int
}

const (
	All                 = -1
	Start dialogs.State = iota
	Continue
	HelpState
	What
)

var log = logging.GetInstance()

// генератор  случайных чисел
func RangeInt(min int, max int) int {
	return rand.Intn(max) + min

}

var redigit = regexp.MustCompile(`(?m)\d+`)

func GetNumber(answerString string) (int, error) {
	result, err := strconv.Atoi(redigit.FindString(answerString))
	if err != nil {
		log.Errorln("Ошибка %s", err)
		return -1, err
	}
	return result, nil
}

func AnswerToSetString(answerString string) *strset.Set {

	delimiter := regexp.MustCompile(`[\s\,\.]+`)
	// преобразование ответа в набор слов
	words := delimiter.Split(answerString, -1)
	return strset.New(words...)

}

func TalkWhatsOfBasket(basket *strset.Set, answer *Answer, itemName string) *Answer {
	// озвучить содержимое корзины
	answer.Response.Text = "В корзине находятся "
	answer.Response.TTS = "В корзине находятся "
	answer.Response.Text += strings.Join(basket.List(), `,`)
	answer.Response.TTS += strings.Join(basket.List(), `- - - - - - - - - - `)
	answer.Response.Text += itemName + ".Повторите."
	answer.Response.TTS += itemName + ".Повторите."
	return answer
}

func checkNewSession(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	if question.Session.New == true {
		p.Storage.Delete(question.Session.UserID)
		p.Storage.SetState(question.Session.UserID, Start)

		return false, nil
	} else {
		return false, nil
	}
}

// стартовый диалог - запрос на размер корзины
func StartDialog(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	// проверка наличия сохраненной сессии, при отсутсвии притворяемся новой

	answer.Response.Text = "Привет, это игра `Запомни что лежит в корзине`. Выберите сколько предметов помещается в корзине (от " +
		strconv.Itoa(minSize) + " до " + strconv.Itoa(maxSize) + ")."
	p.Storage.SetState(question.Session.UserID, Continue)

	return true, nil

}

// распознавание числа - ответа на вопрос размера корзины
// создание корзины с предметами указанного размера
func CreateBasket(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	sizeOfBasket, err := GetNumber(strings.ToLower(question.Request.Command))

	// если в запросе не содержится требуемое число, повторяем вопрос
	if sizeOfBasket < minSize || sizeOfBasket > maxSize || err != nil {
		answer.Response.Text = "Выберите сколько предметов помещается в корзине (от 3 до 9)"
		return true, nil
	}
	// создание корзины нужного размера
	var numberSet int = RangeInt(0, 3)
	basket := setsForBasket[numberSet].Copy() // заготовки по 9 строк
	// обрезать набор до нужного размера
	for {
		if basket.Size() <= sizeOfBasket {
			break
		}
		basket.Pop()
	}
	p.Storage.SetState(question.Session.UserID, What)
	p.Storage.SetData(question.Session.UserID, UserData{basket, numberSet})

	var itemName string // дополнительное название предметов, если они описаны прилагательными
	switch numberSet {
	case 0:
		itemName = " шар"
	default:
		itemName = ""
	}

	answer = TalkWhatsOfBasket(basket, answer, itemName) // озвучить содержимое корзины
	return true, nil
}

// проверка , смена одного предмета
func CheckBasket(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	setAnswer := AnswerToSetString(question.Request.Command)
	userData := p.Storage.GetData(question.Session.UserID).(UserData)

	// чего не хватает
	forgotten := strset.Difference(userData.basket, setAnswer)

	// сообщить чего нет в корзине
	if forgotten.Size() > 0 {
		answer.Response.Text += "Вы забыли " + strings.Join(forgotten.List(), `,`) + "."
	}

	// что-то лишнее
	extra := strset.Difference(setAnswer, userData.basket)              //  нет в корзине из произнесенного
	extra = strset.Difference(extra, setsForBasket[userData.numberSet]) // нет в корзине из существующего набора
	// сообщить что из названного нет в корзине
	if extra.Size() > 0 {
		answer.Response.Text += "Не было  " + strings.Join(extra.List(), `,`) + "."
	}
	return false, nil
}
func ChangeItem(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	userData := p.Storage.GetData(question.Session.UserID).(UserData)
	// выкинуть из корзины произвольный элемент
	rejected := userData.basket.Pop()
	answer.Response.Text += "Достала " + rejected + "."

	// поместить в корзину новый элемент
	for {
		newItem := strset.Difference(setsForBasket[userData.numberSet], userData.basket).Pop()
		if newItem != rejected { // подбор элемента, не совпадающего с выкинутым
			userData.basket.Add(newItem)
			answer.Response.Text += "Добавила " + newItem + ". Что сейчас в корзине ?"
			break
		}
	}
	return true, nil
}

func EndRound(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	answer.Response.Text = "До встречи в следующий раз."
	answer.Response.EndSession = true // заканчиваем сессию
	return true, nil

}
func Help(question Question, answer *Answer, p *dialogs.Pipeline) (finish bool, err error) {
	userData := p.Storage.GetData(question.Session.UserID).(UserData)
	answer.Response.Text = "В корзине " + strings.Join(userData.basket.List(), `,`) + "."
	p.Storage.SetState(question.Session.UserID, HelpState)
	return false, nil
}

func Run() (func(c echo.Context) error, dialogs.Pipeline) {

	var handler func(c echo.Context) error
	var pipeline dialogs.Pipeline
	pipeline.Storage = dialogs.NewMemoryStorage()
	pipeline.Questions, pipeline.Answers, handler = dialogs.New()

	//запуск горутин обработки вопросов и ответов
	pipeline.Register(checkNewSession, Filter{All, []string{}, ""})
	pipeline.Register(EndRound, Filter{-1, []string{"хватит"}, ""})
	pipeline.Register(Help, Filter{What, []string{"подсказка", "помощь", "сдаюсь", "что"}, ""})
	pipeline.Register(StartDialog, Filter{Start, []string{}, ""})
	pipeline.Register(CreateBasket, Filter{Continue, []string{}, ""})
	pipeline.Register(CheckBasket, Filter{What, []string{}, ""})
	pipeline.Register(ChangeItem, Filter{What, []string{}, ""})
	pipeline.Register(ChangeItem, Filter{HelpState, []string{}, ""})
	return handler, pipeline
}
