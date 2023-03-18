package main

import (
	"github.com/rs/zerolog/log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type GamepadEvent int

const (
	GamepadConnected GamepadEvent = iota
	GamepadDisconnected
	PrimaryJustPressed
	SecondaryJustPressed
	StartJustPressed
	LeftPressed
	RightPressed
	DownPressed
	LeftJustPressed
	RightJustPressed
	DownJustPressed
	UpJustPressed
)

type inputDriver struct {
	gamepadIDsBuf []ebiten.GamepadID
	gamepadIDs    map[ebiten.GamepadID]struct{}
}

func NewInputDriver() *inputDriver {
	id := &inputDriver{}
	id.gamepadIDs = map[ebiten.GamepadID]struct{}{}
	return id
}

// returns connection change events, button press events
func (driver *inputDriver) UpdateStateAndReturnPresses() (map[int]GamepadEvent, map[int][]GamepadEvent) {
	if driver.gamepadIDs == nil {
		driver.gamepadIDs = map[ebiten.GamepadID]struct{}{}
	}

	connectionChanges := map[int]GamepadEvent{}

	// Log the gamepad connection events.
	driver.gamepadIDsBuf = inpututil.AppendJustConnectedGamepadIDs(driver.gamepadIDsBuf[:0])
	for _, id := range driver.gamepadIDsBuf {
		log.Printf("gamepad connected: id: %d, SDL ID: %s", id, ebiten.GamepadSDLID(id))
		driver.gamepadIDs[id] = struct{}{}

		// report the gamepad connected
		connectionChanges[int(id)] = GamepadConnected
	}

	for id := range driver.gamepadIDs {
		if inpututil.IsGamepadJustDisconnected(id) {
			log.Printf("gamepad disconnected: id: %d", id)
			delete(driver.gamepadIDs, id)

			// mark this gamepad disconnected
			connectionChanges[int(id)] = GamepadDisconnected
		}
	}

	buttonEvents := map[int][]GamepadEvent{}

	for id := range driver.gamepadIDs {
		buttonEventList := make([]GamepadEvent, 0)

		maxButton := ebiten.GamepadButton(ebiten.GamepadButtonCount(id))
		for b := ebiten.GamepadButton(0); b < maxButton; b++ {
			// Log button events.
			if inpututil.IsGamepadButtonJustPressed(id, b) {
				log.Printf("button pressed: id: %d, button: %d", id, b)
				if b == 0 {
					buttonEventList = append(buttonEventList, PrimaryJustPressed)
				} else if b == 1 || b == 2 {
					buttonEventList = append(buttonEventList, SecondaryJustPressed)
				} else if b == 7 {
					buttonEventList = append(buttonEventList, StartJustPressed)
				} else if b == 13 {
					buttonEventList = append(buttonEventList, LeftJustPressed)
				} else if b == 11 {
					buttonEventList = append(buttonEventList, RightJustPressed)
				} else if b == 12 {
					buttonEventList = append(buttonEventList, DownJustPressed)
				} else if b == 10 {
					buttonEventList = append(buttonEventList, UpJustPressed)
				}
			}

			if ebiten.IsGamepadButtonPressed(id, b) {
				log.Printf("button pressed: id: %d, button: %d", id, b)
				if b == 13 {
					buttonEventList = append(buttonEventList, LeftPressed)
				} else if b == 11 {
					buttonEventList = append(buttonEventList, RightPressed)
				} else if b == 12 {
					buttonEventList = append(buttonEventList, DownPressed)
				}
			}
		}

		buttonEvents[int(id)] = buttonEventList
	}

	return connectionChanges, buttonEvents
}
