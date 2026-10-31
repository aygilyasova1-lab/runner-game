// Package assets хранит файлы игры (шрифт, музыку), вшитые в программу.
// Директива go:embed видит только файлы из папки своего пакета,
// поэтому этот файл лежит прямо в assets.
package assets

import "embed"

//go:embed *
var FS embed.FS
