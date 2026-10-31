package game

import (
	"errors"
	"log"
)

// loadBackgrounds загружает все фоны из списка. Файлы, которые не нашлись,
// пропускаются; если не нашёлся ни один, возвращается ошибка.
func (g *Game) loadBackgrounds(files []string) error {
	g.backgrounds = nil
	for _, f := range files {
		img, err := loadImage(f)
		if err != nil {
			log.Println("фон не загружен:", f, err)
			continue
		}
		g.backgrounds = append(g.backgrounds, img)
	}
	if len(g.backgrounds) == 0 {
		return errors.New("не удалось загрузить ни одного фона")
	}
	g.bgIndex = 0
	g.imgBackground = g.backgrounds[0]
	return nil
}

// changeBackground переключает фон: delta = 1 - следующий, -1 - предыдущий.
func (g *Game) changeBackground(delta int) {
	n := len(g.backgrounds)
	if n == 0 {
		return
	}
	g.bgIndex = (g.bgIndex + delta + n) % n
	g.imgBackground = g.backgrounds[g.bgIndex]
}
