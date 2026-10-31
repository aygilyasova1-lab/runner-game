package game

import (
	"context"
	"fmt"
	"image/color"
	"sync"
	"time"
	"unicode"

	ebiten "github.com/hajimehoshi/ebiten/v2"
	ebitenutil "github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "game.go/gamepb"
)

// Net - клиент сервера рейтинга. Все запросы идут в отдельных горутинах,
// чтобы игра не зависала, если сервер медленный или выключен.
type Net struct {
	client pb.LeaderboardClient

	mu     sync.Mutex
	top    []*pb.ScoreEntry
	rank   int32
	status string
}

func dialServer(addr string) (*Net, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Net{client: pb.NewLeaderboardClient(conn)}, nil
}

// SubmitAndRefresh отправляет счёт и подгружает топ-10. Безопасно вызывать на nil.
func (n *Net) SubmitAndRefresh(name string, score int) {
	if n == nil {
		return
	}
	n.mu.Lock()
	n.rank = 0
	n.top = nil
	n.status = "отправка результата..."
	n.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r, err := n.client.SubmitScore(ctx, &pb.SubmitScoreRequest{Player: name, Score: int32(score)})
		if err != nil {
			n.setStatus("нет связи с сервером")
			return
		}
		t, err := n.client.TopScores(ctx, &pb.TopScoresRequest{Limit: 10})
		if err != nil {
			n.setStatus("не удалось получить рейтинг")
			return
		}

		n.mu.Lock()
		n.rank = r.Rank
		n.top = t.Entries
		n.status = ""
		n.mu.Unlock()
	}()
}

func (n *Net) setStatus(s string) {
	n.mu.Lock()
	n.status = s
	n.mu.Unlock()
}

// Snapshot отдаёт текущее состояние рейтинга для отрисовки.
func (n *Net) Snapshot() ([]*pb.ScoreEntry, int32, string) {
	if n == nil {
		return nil, 0, "сервер не подключён"
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.top, n.rank, n.status
}

// ---- ввод имени (стартовый экран) ----

func (g *Game) updateNameInput() {
	g.inBuf = ebiten.AppendInputChars(g.inBuf[:0])
	for _, r := range g.inBuf {
		if unicode.IsPrint(r) && len(g.name) < 16 {
			g.name = append(g.name, r)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.name) > 0 {
		g.name = g.name[:len(g.name)-1]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.changeBackground(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.changeBackground(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(g.name) > 0 {
		g.state = stateIntro
	}
}

// drawPanel рисует полупрозрачную тёмную панель с белой окантовкой.
func drawPanel(screen *ebiten.Image, x, y, w, h float64) {
	ebitenutil.DrawRect(screen, x, y, w, h, color.RGBA{0, 0, 0, 170}) // прозрачная заливка
	b := color.RGBA{255, 255, 255, 255}
	ebitenutil.DrawRect(screen, x, y, w, 3, b)     // верх
	ebitenutil.DrawRect(screen, x, y+h-3, w, 3, b) // низ
	ebitenutil.DrawRect(screen, x, y, 3, h, b)     // слева
	ebitenutil.DrawRect(screen, x+w-3, y, 3, h, b) // справа
}

// drawTextCentered пишет текст по центру относительно cx.
func drawTextCentered(screen *ebiten.Image, s string, cx, y int) {
	w := text.BoundString(gameFont, s).Dx()
	drawText(screen, s, cx-w/2, y)
}

func (g *Game) drawNameInput(screen *ebiten.Image) {
	// общая рамка меню
	drawPanel(screen, 150, 190, 500, 200)
	drawTextCentered(screen, "Введите имя и нажмите ENTER", 400, 235)

	// поле ввода с мигающим курсором
	drawPanel(screen, 200, 255, 400, 50)
	cursor := ""
	if time.Now().UnixMilli()/500%2 == 0 {
		cursor = "_"
	}
	drawText(screen, string(g.name)+cursor, 215, 288)

	hint := fmt.Sprintf("Стрелки влево/вправо - фон (%d из %d)", g.bgIndex+1, len(g.backgrounds))
	drawTextCentered(screen, hint, 400, 360)
}

// ---- экран конца игры с рейтингом ----

func (g *Game) drawGameOver(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 180, 30, 440, 500, color.RGBA{0, 0, 0, 210})
	drawText(screen, "ИГРА ОКОНЧЕНА", 300, 70)
	drawText(screen, fmt.Sprintf("Ваш счёт: %d", g.score), 300, 100)

	top, rank, status := g.net.Snapshot()
	if rank > 0 {
		drawText(screen, fmt.Sprintf("Ваше место: %d", rank), 300, 130)
	}

	drawText(screen, "ТОП-10", 300, 170)
	for i, e := range top {
		drawText(screen, fmt.Sprintf("%d. %s - %d", i+1, e.Player, e.Score), 220, 200+i*26)
	}
	if status != "" {
		drawText(screen, status, 220, 480)
	}
	drawText(screen, "Пробел - играть снова", 220, 510)
}
