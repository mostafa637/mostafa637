package ui

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
)

//go:embed assets/ish-paste.png
var ishPasteData []byte

//go:embed assets/ish-hide-keyboard.png
var ishHideKeyboardData []byte

var ishPasteImage = decodeAsset(ishPasteData)
var ishHideKeyboardImage = decodeAsset(ishHideKeyboardData)

func decodeAsset(data []byte) image.Image {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	return img
}
