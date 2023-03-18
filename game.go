package main

import (
	"image"
	_ "image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

type imageMap map[string]image.Image

func getImageFromFilePath(filePath string) (image.Image, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	image, _, err := image.Decode(f)
	return image, err
}

func loadImageAndAddToImageMap(imageMap imageMap, filePath string, key string) error {
	img, err := getImageFromFilePath(filePath)
	if err != nil {
		return err
	}

	imageMap[key] = img
	return nil
}

// Game implements ebiten.Game interface.
type Game struct {
	imageMap              imageMap
	playfieldViz          *playfieldViz
	matchDriver           *matchDriver
	inputDriver           *inputDriver
	controllerAssignments map[int]int // map of player index to controller id int
}

func NewGame() (*Game, error) {
	game := &Game{imageMap: make(imageMap)}

	// TODO: error check
	// refactor into an asset loader that either just loads everything in the dir
	// or takes in a list of needed assets
	loadImageAndAddToImageMap(game.imageMap, "./img/redVirus.png", "redVirus")
	loadImageAndAddToImageMap(game.imageMap, "./img/blueVirus.png", "blueVirus")
	loadImageAndAddToImageMap(game.imageMap, "./img/yellowVirus.png", "yellowVirus")
	loadImageAndAddToImageMap(game.imageMap, "./img/redSingle.png", "redSingle")
	loadImageAndAddToImageMap(game.imageMap, "./img/blueSingle.png", "blueSingle")
	loadImageAndAddToImageMap(game.imageMap, "./img/yellowSingle.png", "yellowSingle")
	loadImageAndAddToImageMap(game.imageMap, "./img/redLinked.png", "redLinked")
	loadImageAndAddToImageMap(game.imageMap, "./img/yellowLinked.png", "yellowLinked")
	loadImageAndAddToImageMap(game.imageMap, "./img/blueLinked.png", "blueLinked")

	game.playfieldViz = NewPlayfieldViz(game.imageMap)
	game.playfieldViz.SetPixelSize(200, 400)

	game.matchDriver = NewMatchDriver()

	game.inputDriver = NewInputDriver()

	game.controllerAssignments = map[int]int{}
	/*
		redVirus, _ := drbreakboard.MakeVirus(drbreakboard.Red)
		game.playfield.PutSpaceAtCoordinateIfEmpty(0, 0, redVirus)
		game.playfield.PutSpaceAtCoordinateIfEmpty(1, 1, redVirus)

		coord, linked, _ := drbreakboard.MakeLinkedPillSpaces(drbreakboard.Up, drbreakboard.Red, drbreakboard.Blue)
		game.playfield.PutTwoLinkedSpacesAtCoordinate(2, 2, coord, linked)
		coord, linked, _ = drbreakboard.MakeLinkedPillSpaces(drbreakboard.Right, drbreakboard.Red, drbreakboard.Blue)
		game.playfield.PutTwoLinkedSpacesAtCoordinate(4, 1, coord, linked)
		game.playfieldViz.UpdateBoard(game.playfield)
	*/

	return game, nil
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (g *Game) Update() error {
	// Write your game's logical update.
	_, buttonPressEvents := g.inputDriver.UpdateStateAndReturnPresses()

	if !g.matchDriver.matchStarted {
		for k := range buttonPressEvents {
			events := buttonPressEvents[k]
			for _, event := range events {
				if event == StartJustPressed {
					g.controllerAssignments[0] = k
					g.matchDriver.StartMatch(1)
				}
			}
		}
	} else if !g.matchDriver.matchEnded {
		playerIndexInputs := map[int][]GamepadEvent{}
		// all players should have an input device before game start
		for playerIndex := range g.controllerAssignments {
			controllerId := g.controllerAssignments[playerIndex]
			controllerEvent, exists := buttonPressEvents[controllerId]

			// put the controller event array into the player indexed event map
			if exists {
				playerIndexInputs[playerIndex] = controllerEvent
			}
		}

		// TODO: check for pause button press before applaying update

		// apply rotations based on button presses
		g.matchDriver.ApplyInputs(playerIndexInputs)

		g.matchDriver.ApplyTick(buttonPressEvents)
		g.playfieldViz.UpdateBoard(g.matchDriver.GetPlayfield(0),
			g.matchDriver.GetActivePill(0), g.matchDriver.GetActivePillLocation(0))
	}

	return nil
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	// Write your game's rendering.
	g.playfieldViz.DrawBoardToImage(screen)
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480
}
