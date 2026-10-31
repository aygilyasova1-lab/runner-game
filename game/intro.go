package game

import (
	"image/color"

	ebiten "github.com/hajimehoshi/ebiten/v2"
	ebitenutil "github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// updateIntro: на стартовом экране ждём ENTER, после чего начинается игра.
func (g *Game) updateIntro() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.reset()
		g.state = statePlay
	}
}

// drawIntro рисует картинку стартового экрана на весь экран (800x600).
func (g *Game) drawIntro(screen *ebiten.Image) {
	if g.imgStart == nil { // картинки нет - простая заглушка
		ebitenutil.DrawRect(screen, 0, 0, 800, 600, color.RGBA{30, 30, 40, 255})
		drawText(screen, "Нажмите ENTER, чтобы начать", 250, 300)
		return
	}

	w, h := g.imgStart.Bounds().Dx(), g.imgStart.Bounds().Dy()
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(800/float64(w), 600/float64(h))
	screen.DrawImage(g.imgStart, op)

	drawText(screen, "Игрок: "+string(g.name), 10, 25)
}
