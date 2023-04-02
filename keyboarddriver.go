package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type keyboardDriver struct {
}

func NewKeyboardDriver() *keyboardDriver {
	id := &keyboardDriver{}
	return id
}

func (driver *keyboardDriver) GetKeyboardAsGamepadEvents() []GamepadEvent {
	buttonEventList := make([]GamepadEvent, 16)

	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		buttonEventList = append(buttonEventList, DownPressed)
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		buttonEventList = append(buttonEventList, LeftPressed)
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		buttonEventList = append(buttonEventList, RightPressed)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		buttonEventList = append(buttonEventList, RightJustPressed)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		buttonEventList = append(buttonEventList, LeftJustPressed)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		buttonEventList = append(buttonEventList, DownJustPressed)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		buttonEventList = append(buttonEventList, UpJustPressed)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		buttonEventList = append(buttonEventList, PrimaryJustPressed)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		buttonEventList = append(buttonEventList, SecondaryJustPressed)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		buttonEventList = append(buttonEventList, StartJustPressed)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		buttonEventList = append(buttonEventList, SelectJustPressed)
	}

	return buttonEventList
}
