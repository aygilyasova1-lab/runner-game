package game

import (
	ebiten "github.com/hajimehoshi/ebiten/v2"
	ebitenutil "github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"game.go/assets"
)

// loadImage загружает картинку из папки assets, вшитой в программу.
// name - имя файла без папки, например "leo.png".
func loadImage(name string) (*ebiten.Image, error) {
	f, err := assets.FS.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := ebitenutil.NewImageFromReader(f)
	return img, err
}
