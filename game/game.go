package game

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"strconv"

	ebiten "github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	ebitenutil "github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"game.go/assets"
)

const (
	groundY  = 500.0 // уровень земли
	noFloor  = 9999.0
	deathY   = 560.0 // ниже этой линии ног игрок утонул
	waterLen = 1200.0

	hideFrames = 120 // 2 секунды (60 кадров в секунду): через столько после ускорения вернутся преследователи
	bikeFrames = 150 // 2,5 секунды на велосипеде

	sampleRate = 44100
)

// состояния игры: g.state хранит номер текущего экрана
const (
	stateSetup = 0 // ввод имени и выбор фона
	statePlay  = 1 // игра
	stateOver  = 2 // конец игры
	stateIntro = 3 // стартовый экран с картинкой
)

var (
	gameFont font.Face
	music    *audio.Player
)

type Obstacle struct {
	obstX      float64
	obstY      float64
	obstWidth  float64
	obstHeight float64
	isHigh     bool
}

type Player struct {
	playerX   float64
	playerY   float64
	playerW   float64
	playerH   float64
	velocityY float64
	onGround  bool
	fallen    bool
}

type Chaser struct {
	x         float64
	y         float64
	height    float64
	width     float64
	active    bool
	catching  bool
	speed     float64
	hidden    bool // временно исчез
	hideTimer int  // сколько кадров ещё не показывать после ускорения
}

type Water struct {
	x      float64
	width  float64
	active bool
}

// Boat - льдина с Леонардо. y - верхняя поверхность, на ней стоит игрок.
type Boat struct {
	x       float64
	y       float64
	width   float64
	height  float64
	active  bool
	riding  bool // игрок на льдине, мир ускорен
	sinking bool // льдина тонет
}

// Biker - человек с велосипедом, стоит на земле
type Biker struct {
	x      float64
	width  float64
	height float64
	active bool
}

type Game struct {
	player     Player
	chaser     Chaser
	obstacles  []Obstacle
	water      Water
	boat       Boat
	biker      Biker
	bikeTimer  int  // сколько кадров осталось ехать на велосипеде
	waterPhase bool // игрок на льдине или ещё не вернулся на землю после воды
	spawnTimer int
	nextSpawn  int
	gameSpeed  float64
	score      int
	obstCount  int
	waterAfter int // после скольких препятствий появится вода (случайно)

	state int // 0 - стартовый экран, 1 - игра, 2 - конец

	// имя игрока и рейтинг (см. net.go)
	name      []rune
	inBuf     []rune
	submitted bool
	net       *Net

	// фоны (см. background.go)
	imgBackground *ebiten.Image
	backgrounds   []*ebiten.Image
	bgIndex       int
	bkX           float64

	imgPlayer     *ebiten.Image
	imgChaser     *ebiten.Image
	imgPlayerDuck *ebiten.Image
	imgLeo        *ebiten.Image
	imgBiker      *ebiten.Image
	imgStart      *ebiten.Image // картинка стартового экрана
}

// ---- шрифт, текст, музыка ----

func loadFont() error {
	data, err := assets.FS.ReadFile("font.ttf")
	if err != nil {
		return fmt.Errorf("не найден assets/font.ttf: %w", err)
	}
	tt, err := opentype.Parse(data)
	if err != nil {
		return err
	}
	gameFont, err = opentype.NewFace(tt, &opentype.FaceOptions{Size: 20, DPI: 72})
	return err
}

// drawText - белый текст с чёрной тенью
func drawText(screen *ebiten.Image, s string, x, y int) {
	text.Draw(screen, s, gameFont, x+1, y+1, color.Black)
	text.Draw(screen, s, gameFont, x, y, color.White)
}

// startMusic запускает музыку по кругу. Если файла нет, игра идёт без звука.
func startMusic(path string) {
	data, err := assets.FS.ReadFile(path)
	if err != nil {
		log.Println("музыка не найдена:", err)
		return
	}
	stream, err := mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		log.Println("не удалось прочитать музыку:", err)
		return
	}
	loop := audio.NewInfiniteLoop(stream, stream.Length())
	music, err = audio.NewContext(sampleRate).NewPlayer(loop)
	if err != nil {
		log.Println(err)
		music = nil
		return
	}
	music.SetVolume(0.5)
	music.Play()
}

// drawBubble - реплика как в комиксе. tailX, tailY - куда указывает хвостик.
func drawBubble(screen *ebiten.Image, lines []string, tailX, tailY float64) {
	m := gameFont.Metrics()
	lineH := m.Height.Ceil()
	asc := m.Ascent.Ceil()
	pad := 10

	maxW := 0
	for _, l := range lines {
		if w := text.BoundString(gameFont, l).Dx(); w > maxW {
			maxW = w
		}
	}
	w := float64(maxW + pad*2)
	h := float64(len(lines)*lineH + pad*2)

	x := tailX
	y := tailY - 14 - h // 14 - высота хвостика

	// рамка и заливка
	ebitenutil.DrawRect(screen, x-2, y-2, w+4, h+4, color.Black)
	ebitenutil.DrawRect(screen, x, y, w, h, color.White)

	// хвостик ступеньками: сначала чёрный контур, потом белая заливка
	for i := 0; i < 14; i++ {
		tw := 20 - float64(i)*1.4
		tx := x + 15 + float64(i)*0.8
		ebitenutil.DrawRect(screen, tx-2, y+h+float64(i), tw+4, 1, color.Black)
	}
	for i := 0; i < 14; i++ {
		tw := 20 - float64(i)*1.4
		tx := x + 15 + float64(i)*0.8
		ebitenutil.DrawRect(screen, tx, y+h+float64(i), tw, 1, color.White)
	}

	for i, l := range lines {
		text.Draw(screen, l, gameFont, int(x)+pad, int(y)+pad+asc+i*lineH, color.Black)
	}
}

// drawBike рисует простой велосипед (два колеса и рама), x - левый край, bottom - уровень земли
func drawBike(screen *ebiten.Image, x, bottom float64) {
	r := 12.0
	rear := x + r
	front := x + 52 - r
	cy := bottom - r

	for _, cx := range []float64{rear, front} {
		ebitenutil.DrawCircle(screen, cx, cy, r, color.Black)
		ebitenutil.DrawCircle(screen, cx, cy, r-3, color.RGBA{170, 170, 170, 255})
	}
	red := color.RGBA{200, 40, 40, 255}
	ebitenutil.DrawRect(screen, rear, cy-2, front-rear, 3, red)     // рама
	ebitenutil.DrawRect(screen, x+26, cy-18, 3, 18, red)            // подседельная труба
	ebitenutil.DrawRect(screen, x+21, cy-20, 13, 3, color.Black)    // седло
	ebitenutil.DrawRect(screen, front-1, cy-16, 3, 16, red)         // вилка
	ebitenutil.DrawRect(screen, front-6, cy-18, 12, 3, color.Black) // руль
}

// drawScaled рисует картинку нужной высоты с сохранением пропорций; bottom - нижняя линия
func drawScaled(screen, img *ebiten.Image, x, bottom, h float64) {
	s := h / float64(img.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(x, bottom-h)
	screen.DrawImage(img, op)
}

// ---- состояние игры ----

// reset ставит игру в начальное состояние (и для старта, и для рестарта)
func (g *Game) reset() {
	g.obstacles = nil
	g.water = Water{}
	g.boat = Boat{}
	g.biker = Biker{}
	g.bikeTimer = 0
	g.waterPhase = false
	g.score = 0
	g.spawnTimer = 0
	g.nextSpawn = 100
	g.gameSpeed = 5
	g.obstCount = 0
	g.waterAfter = 2 + rand.Intn(4) // от 2 до 5 препятствий
	g.player = Player{playerX: 300, playerY: 450, playerW: 50, playerH: 50, onGround: true}
	g.chaser = Chaser{x: 50, y: 450, width: 50, height: 50, speed: 3, active: true}
}

// floorAt возвращает высоту опоры под игроком: льдина, вода (опоры нет) или земля
func (g *Game) floorAt(cx float64) float64 {
	if g.boat.active && cx >= g.boat.x && cx <= g.boat.x+g.boat.width {
		return g.boat.y
	}
	if g.water.active && cx >= g.water.x && cx <= g.water.x+g.water.width {
		return noFloor
	}
	return groundY
}

func (g *Game) Update() error {
	// сначала вводим имя и выбираем фон, потом стартовый экран; героев ещё нет
	switch g.state {
	case stateSetup:
		g.updateNameInput()
		return nil
	case stateIntro:
		g.updateIntro()
		return nil
	}

	if g.state == 2 {
		if !g.submitted { // один раз отправляем результат на сервер
			g.submitted = true
			g.net.SubmitAndRefresh(string(g.name), g.score)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.submitted = false
			g.reset()
			g.state = statePlay
		}
		return nil // мир замер
	}

	// ---- state == 1 ----
	speed := g.gameSpeed
	if g.boat.riding {
		speed *= 2 // на льдине всё ускоряется
	} else if g.bikeTimer > 0 {
		speed *= 1.7 // на велосипеде тоже быстрее
	}
	if g.bikeTimer > 0 {
		g.bikeTimer--
	}

	g.bkX -= speed * 0.3
	if g.bkX <= -800 {
		g.bkX = 0
	}

	// ---- игрок ----
	p := &g.player
	center := p.playerX + p.playerW/2
	floor := g.floorAt(center)

	if p.onGround && floor == noFloor { // земля кончилась, опоры нет
		p.onGround = false
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && p.onGround && !p.fallen {
		p.velocityY = -13
		p.onGround = false
	}

	if p.onGround {
		if ebiten.IsKeyPressed(ebiten.KeyDown) && !p.fallen {
			p.playerH = 20
			p.playerW = 20
		} else {
			p.playerH = 50
			p.playerW = 50
		}
		p.playerY = floor - p.playerH
	}

	p.velocityY += 0.7 // гравитация
	p.playerY += p.velocityY

	bottom := p.playerY + p.playerH
	if p.velocityY >= 0 && floor != noFloor && bottom >= floor && bottom-p.velocityY <= floor+2 {
		p.playerY = floor - p.playerH
		p.velocityY = 0
		p.onGround = true
	}

	if bottom > deathY { // утонул
		g.state = 2
		return nil
	}

	// ---- появление препятствий ----
	if !g.water.active { // пока на экране вода, новые препятствия не создаём
		g.spawnTimer++
		if g.spawnTimer >= g.nextSpawn {
			g.spawnTimer = 0
			if g.obstCount >= g.waterAfter {
				g.water = Water{x: 800, width: waterLen, active: true}
				g.boat = Boat{x: 800, y: groundY, width: 150, height: 20, active: true}
				g.obstCount = 0
				g.waterAfter = 2 + rand.Intn(4) // следующая вода через 2-5 препятствий
			} else {
				g.obstCount++
				g.score++
				g.gameSpeed += 0.2
				g.nextSpawn = 60 + rand.Intn(80)

				isHigh := rand.Float64() > 0.5
				w := rand.Float64()*20 + 20
				h := rand.Float64()*30 + 30
				var y float64
				if isHigh {
					y = 410
				} else {
					y = groundY - h
				}
				g.obstacles = append(g.obstacles, Obstacle{obstX: 800, obstY: y, obstWidth: w, obstHeight: h, isHigh: isHigh})

				// иногда за препятствием стоит велосипедист
				if g.obstCount == 1 && !g.biker.active && g.bikeTimer == 0 && rand.Float64() < 0.4 {
					bh := 120.0 // высота велосипедиста, меняйте эту цифру
					bw := bh
					if g.imgBiker != nil {
						bw = bh * float64(g.imgBiker.Bounds().Dx()) / float64(g.imgBiker.Bounds().Dy())
					}
					g.biker = Biker{x: 1000, width: bw, height: bh, active: true}
				}
			}
		}
	}

	// ---- движение препятствий и столкновения ----
	alive := g.obstacles[:0]
	for i := range g.obstacles {
		o := &g.obstacles[i]
		o.obstX -= speed

		if !g.boat.riding && !p.fallen &&
			p.playerX < o.obstX+o.obstWidth &&
			p.playerX+p.playerW > o.obstX &&
			p.playerY < o.obstY+o.obstHeight &&
			p.playerY+p.playerH > o.obstY {
			p.fallen = true
			p.onGround = false
			p.velocityY = 2
			g.bikeTimer = 0 // велосипед потерян
			g.chaser.hidden = false
			g.chaser.hideTimer = 0
			g.chaser.catching = true
			g.chaser.active = true
			g.chaser.x = -60
		}

		if o.obstX+o.obstWidth > 0 { // убираем ушедшие за экран
			alive = append(alive, *o)
		}
	}
	g.obstacles = alive

	// ---- велосипедист ----
	if g.biker.active {
		k := &g.biker
		k.x -= speed
		if p.playerX < k.x+k.width && p.playerX+p.playerW > k.x &&
			p.playerY < groundY && p.playerY+p.playerH > groundY-k.height {
			k.active = false // забрала велосипед
			g.bikeTimer = bikeFrames
			p.fallen = false
		} else if k.x+k.width < 0 {
			k.active = false
		}
	}

	// ---- вода и льдина ----
	b := &g.boat
	if g.water.active {
		g.water.x -= speed
		if g.water.x+g.water.width < 0 {
			g.water.active = false
			g.spawnTimer = 0
			g.nextSpawn = 100
		}
	}

	if b.active {
		// посадка: центр льдины дошёл до игрока
		if !b.riding && !b.sinking && p.onGround && center >= b.x+b.width/2 {
			b.riding = true
			g.waterPhase = true
			b.x = center - b.width/2
			p.fallen = false
		}

		// до конца воды осталось мало - льдина начинает тонуть
		if b.riding && g.water.x+g.water.width-center < 150 {
			b.riding = false
			b.sinking = true
		}

		if !b.riding {
			b.x -= speed // обычная льдина плывёт вместе с миром
		}
		if b.sinking {
			b.y += 1.5
		}
		if b.x+b.width < 0 {
			b.active = false
		}
	}

	// игрок спрыгнул на землю за водой - начинается отсчёт возвращения преследователей
	if g.waterPhase && !b.riding &&
		(!g.water.active || (p.onGround && center > g.water.x+g.water.width)) {
		g.waterPhase = false
	}

	// ---- преследователь ----
	c := &g.chaser
	away := b.riding || g.waterPhase || g.bikeTimer > 0
	if away {
		c.hidden = true
		c.hideTimer = hideFrames
		c.catching = false
	} else if c.hidden {
		c.hideTimer--
		if c.hideTimer <= 0 {
			c.hidden = false
			c.x = -60 // выезжает слева
		}
	}

	switch {
	case c.hidden: // отстаёт и уходит за левый край
		if c.x > -200 {
			c.x -= 12
		}
	case c.active && c.catching:
		c.x += c.speed
		if c.x+c.width >= p.playerX {
			g.state = 2
		}
	case c.x < 50: // возвращается на своё место
		c.x += 2
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// стартовый экран с картинкой
	if g.state == stateIntro {
		g.drawIntro(screen)
		return
	}

	op1 := &ebiten.DrawImageOptions{}
	op1.GeoM.Translate(g.bkX, 0)
	screen.DrawImage(g.imgBackground, op1)

	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(g.bkX+800, 0)
	screen.DrawImage(g.imgBackground, op2)

	ebitenutil.DrawRect(screen, 0, groundY, 800, 100, color.RGBA{128, 128, 128, 255}) // земля

	// стартовый экран: только фон и ввод имени, героев ещё нет
	if g.state == 0 {
		g.drawNameInput(screen)
		return
	}

	// велосипедист
	if g.biker.active {
		k := g.biker
		if g.imgBiker != nil {
			drawScaled(screen, g.imgBiker, k.x, groundY, k.height)
		} else {
			ebitenutil.DrawRect(screen, k.x+20, groundY-52, 22, 30, color.RGBA{40, 140, 70, 255})
			ebitenutil.DrawCircle(screen, k.x+31, groundY-58, 8, color.RGBA{240, 200, 160, 255})
			drawBike(screen, k.x+4, groundY)
		}
	}

	// льдина и Леонардо
	if g.boat.active {
		b := g.boat
		ebitenutil.DrawRect(screen, b.x, b.y, b.width, b.height, color.RGBA{200, 230, 255, 255})

		leoX, leoY := b.x+b.width-55, b.y-50
		if g.imgLeo != nil {
			lw, lh := g.imgLeo.Bounds().Dx(), g.imgLeo.Bounds().Dy()
			op := &ebiten.DrawImageOptions{}
			op.Filter = ebiten.FilterLinear
			op.GeoM.Scale(70/float64(lw), 70/float64(lh))
			op.GeoM.Translate(leoX, leoY)
			screen.DrawImage(g.imgLeo, op)
			drawText(screen, "Лео", int(leoX)+5, int(leoY)-10)
		} else {
			ebitenutil.DrawRect(screen, leoX, leoY, 50, 50, color.RGBA{255, 180, 0, 255})
		}
	}

	// игрок (на велосипеде - с колёсами)
	pl := g.player
	if g.bikeTimer > 0 {
		drawBike(screen, pl.playerX-1, pl.playerY+pl.playerH)
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) && pl.onGround {
		if g.imgPlayerDuck != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(pl.playerX, pl.playerY + 10)
			screen.DrawImage(g.imgPlayerDuck, op)
		} else {
			ebitenutil.DrawRect(screen, pl.playerX, pl.playerY, pl.playerW, pl.playerH, color.White)
		}
	} else {
		if g.imgPlayer != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(pl.playerX, pl.playerY)
			screen.DrawImage(g.imgPlayer, op)
		} else {
			ebitenutil.DrawRect(screen, pl.playerX, pl.playerY, pl.playerW, pl.playerH, color.White)
		}
	}

	// "АВ" над игроком, по центру
	abW := text.BoundString(gameFont, "АВ").Dx()
	drawText(screen, "АВ", int(pl.playerX+pl.playerW/2)-abW/2, int(pl.playerY)-10)

	// преследователь и его реплика
	c := g.chaser
	if c.active && c.x+c.width > 0 {
		chs := &ebiten.DrawImageOptions{}
		chs.GeoM.Translate(c.x, c.y)
		screen.DrawImage(g.imgChaser, chs)
		if !c.hidden && c.x >= 0 {
			drawBubble(screen,
				[]string{"Александра Валерьевна,", "скиньте билеты на экзамен"},
				c.x, c.y)
		}
	}

	// препятствия
	for _, v := range g.obstacles {
		ebitenutil.DrawRect(screen, v.obstX, v.obstY, v.obstWidth, v.obstHeight, color.RGBA{128, 128, 128, 255})
	}

	// вода рисуется последней и чуть прозрачная: тонущая льдина уходит "под воду"
	if g.water.active {
		ebitenutil.DrawRect(screen, g.water.x, groundY, g.water.width, 100, color.RGBA{15, 50, 100, 160})
	}

	ebitenutil.DebugPrint(screen, "Score: "+strconv.Itoa(g.score))

	if g.state == 2 {
		g.drawGameOver(screen)
	}
}

func (g *Game) Layout(outsideWidth int, outsideHeight int) (int, int) {
	return 800, 600
}

// Run загружает всё нужное и запускает игру. Вызывается из main.
func Run() error {
	if err := loadFont(); err != nil {
		return err
	}

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("МХК")

	game := &Game{}
	game.reset()

	err := game.loadBackgrounds([]string{
		"wallaper.png",
		"bg2.png",
		"bg3.png",
	})
	if err != nil {
		return err
	}

	game.imgPlayer, err = loadImage("heroFinal.png")
	if err != nil {
		return fmt.Errorf("не удалось загрузить героя: %w", err)
	}

	game.imgChaser, err = loadImage("chaser.png")
	if err != nil {
		log.Println("assets/chaser.png не найден")
		game.imgChaser = nil
	}

	game.imgPlayerDuck, err = loadImage("heroDuck.png")
	if err != nil {
		return fmt.Errorf("не удалось загрузить героя (присед): %w", err)
	}

	// Леонардо и велосипедист: если файла нет, рисуется заглушка
	game.imgLeo, err = loadImage("leo.png")
	if err != nil {
		log.Println("assets/leo.png не найден, пока будет заглушка")
		game.imgLeo = nil
	}

	game.imgBiker, err = loadImage("biker.png")
	if err != nil {
		log.Println("assets/biker.png не найден, пока будет заглушка")
		game.imgBiker = nil
	}

	// стартовый экран; если файла нет, будет простая заглушка
	game.imgStart, err = loadImage("start.png")
	if err != nil {
		log.Println("assets/start.png не найден, пока будет заглушка")
		game.imgStart = nil
	}

	// сервер рейтинга; если он выключен, игра всё равно работает
	game.net, err = dialServer(serverAddress())
	if err != nil {
		log.Println("нет подключения к серверу:", err)
		game.net = nil
	}

	startMusic("music.mp3")

	return ebiten.RunGame(game)
}
