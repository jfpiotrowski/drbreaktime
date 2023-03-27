package main

import (
	_ "embed"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// font stuff
var (
	//go:embed fonts/Arimo-Regular.ttf
	BaseTextTTF []byte

	BaseTextFont font.Face
)

func init() {
	// load some fonts
	tt, err := opentype.Parse(BaseTextTTF)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	BaseTextFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}
}

type GameStage int
type imageMap map[string]image.Image
type fontMap map[string]font.Face

const (
	Title GameStage = iota
	PlayerAssignment
	MatchRunning
	MatchPaused
	MatchEnded
)

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
	fontMap               fontMap
	playfieldViz          []*playfieldViz
	matchDriver           *matchDriver
	inputDriver           *inputDriver
	controllerAssignments map[int]int // map of player index to controller id int
	currentStage          GameStage
	playerCount           int
	lastButtonPresses     map[int][]GamepadEvent
	pausePlayerIndex      int
}

func NewGame() (*Game, error) {
	game := &Game{imageMap: make(imageMap)}
	game.fontMap = make(fontMap)
	game.fontMap["base"] = BaseTextFont

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
	loadImageAndAddToImageMap(game.imageMap, "./img/greenPixel.png", "greenPixel")

	game.inputDriver = NewInputDriver()

	game.ResetGame()

	return game, nil
}

// set game variables to base state with no players or anything
func (g *Game) ResetGame() {
	g.playfieldViz = make([]*playfieldViz, 0)

	g.matchDriver = NewMatchDriver()

	g.controllerAssignments = map[int]int{}

	g.currentStage = Title

	g.playerCount = 0

	g.lastButtonPresses = map[int][]GamepadEvent{}

	g.pausePlayerIndex = 0

}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (g *Game) Update() error {
	// Write your game's logical update.
	_, buttonPressEvents := g.inputDriver.UpdateStateAndReturnPresses()
	g.lastButtonPresses = buttonPressEvents

	switch g.currentStage {
	case Title:

		// pick up start button
		for k := range buttonPressEvents {
			events := buttonPressEvents[k]
			for _, event := range events {
				if event == StartJustPressed {
					g.playerCount = 0
					g.currentStage = PlayerAssignment
				}
			}
		}

	case PlayerAssignment:
		g.updateReadyForPlayers(buttonPressEvents)
	case MatchRunning:
		if !g.matchDriver.matchStarted {
			g.matchDriver.StartMatch()
		} else if !g.matchDriver.matchEnded {
			// all players should have an input device before game start
			// put the controller event array into the player indexed event map
			playerIndexInputs := g.getPlayerButtonPresses(buttonPressEvents)

			// check for pause button press before applying update
			for playerIndex := 0; playerIndex < g.playerCount; playerIndex += 1 {
				controllerEvent, exists := playerIndexInputs[playerIndex]

				if exists && checkControllerEventsForEvent(controllerEvent, StartJustPressed) {
					// player paused the game
					g.currentStage = MatchPaused
					g.pausePlayerIndex = playerIndex
					return nil
				}
			}

			// apply rotations based on button presses
			g.matchDriver.ApplyInputs(playerIndexInputs)

			g.matchDriver.ApplyTick(playerIndexInputs)

			for i := 0; i < g.playerCount; i++ {
				g.playfieldViz[i].UpdateBoard(g.matchDriver.GetPlayfield(i),
					g.matchDriver.GetActivePill(i), g.matchDriver.GetActivePillLocation(i))
			}

			if g.matchDriver.matchEnded {
				g.currentStage = MatchEnded
			}
		}
	case MatchPaused:
		// all players should have an input device before game start
		// put the controller event array into the player indexed event map
		playerIndexInputs := g.getPlayerButtonPresses(buttonPressEvents)

		// check for pause button press for unpause
		pausePlayerEvents, exists := playerIndexInputs[g.pausePlayerIndex]
		if exists && checkControllerEventsForEvent(pausePlayerEvents, StartJustPressed) {
			// player unpaused, restart the game
			g.currentStage = MatchRunning
			return nil
		} else if exists && checkControllerEventsForEvent(pausePlayerEvents, SelectJustPressed) {
			g.ResetGame()
		}
	case MatchEnded:
		// pick up start button and run a new match
		for k := range buttonPressEvents {
			events := buttonPressEvents[k]
			for _, event := range events {
				if event == StartJustPressed {
					// start the match again
					g.matchDriver.ResetAndStartMatch()
					g.currentStage = MatchRunning
					return nil
				} else if event == SelectJustPressed {
					// reset the whole game
					g.ResetGame()
					return nil
				}
			}
		}
	}

	return nil
}

func checkControllerEventsForEvent(controllerEvent []GamepadEvent, targetEvent GamepadEvent) bool {
	if controllerEvent == nil {
		return false
	}

	for _, event := range controllerEvent {
		if event == targetEvent {
			return true
		}
	}

	return false
}

func (g *Game) getPlayerButtonPresses(buttonPressEvents map[int][]GamepadEvent) map[int][]GamepadEvent {
	playerIndexInputs := map[int][]GamepadEvent{}
	for playerIndex := range g.controllerAssignments {
		controllerId := g.controllerAssignments[playerIndex]
		controllerEvent, exists := buttonPressEvents[controllerId]

		if exists {
			playerIndexInputs[playerIndex] = controllerEvent
		}
	}
	return playerIndexInputs
}

func (g *Game) updateReadyForPlayers(buttonPressEvents map[int][]GamepadEvent) {
	// pick up new players
	for controllerId, events := range buttonPressEvents {
		for _, event := range events {
			if event == StartJustPressed {
				// map to player if controller not currently mapped
				playerIndex := -1
				for assignedIndex, assignedId := range g.controllerAssignments {
					if controllerId == assignedId {
						playerIndex = assignedIndex
						break
					}
				}
				if playerIndex == -1 {
					// assign the new controller to a new player
					pv := NewPlayfieldViz(g.imageMap, g.fontMap)
					pv.SetPixelSizeAndOffset(160, 480, g.playerCount*160, 0)
					g.playfieldViz = append(g.playfieldViz, pv)
					g.controllerAssignments[g.playerCount] = controllerId
					g.matchDriver.AddPlayer()
					g.playerCount += 1
				}
			}
		}
	}

	// handle level setting and ready for joined players
	for controllerId, events := range buttonPressEvents {
		playerIndex := -1
		for assignedIndex, assignedId := range g.controllerAssignments {
			if controllerId == assignedId {
				playerIndex = assignedIndex
				break
			}
		}

		// unjoined input, ignore
		if playerIndex < 0 {
			continue
		}

		ready, _ := g.matchDriver.GetPlayerReady(playerIndex)

		// handle level setting and ready buttons
		for _, event := range events {
			if event == UpJustPressed && !ready {
				_ = g.matchDriver.ChangeLevel(playerIndex, 1)
			} else if event == DownJustPressed && !ready {
				_ = g.matchDriver.ChangeLevel(playerIndex, -1)
			} else if event == PrimaryJustPressed {
				_ = g.matchDriver.SetPlayerReady(playerIndex, true)
			} else if event == SecondaryJustPressed {
				_ = g.matchDriver.SetPlayerReady(playerIndex, false)
			} else if event == SelectJustPressed {
				g.ResetGame()
				return
			}
		}
	}

	// no players, do nothing
	if g.playerCount == 0 {
		return
	}

	// check if all players ready
	allPlayersReady := true
	for i := 0; i < g.playerCount; i++ {
		ready, _ := g.matchDriver.GetPlayerReady(i)
		if !ready {
			allPlayersReady = false
		}
	}

	if allPlayersReady {
		// start the match
		g.matchDriver.StartMatch()
		g.currentStage = MatchRunning
	}
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	switch g.currentStage {
	case Title:
		text.Draw(screen, "Dr Breaktime!\n\nPress Start", BaseTextFont, 100, 100, color.RGBA{128, 128, 128, 255})
	case PlayerAssignment:
		for playerIndex, pv := range g.playfieldViz {
			level, err := g.matchDriver.GetLevel(playerIndex)
			if err != nil {
				panic("no level for drawn player during player assignment")
			}
			ready, err := g.matchDriver.GetPlayerReady(playerIndex)
			if err != nil {
				panic("no ready value present during player assignment")
			}
			pv.DrawWaitingPlayerToImage(screen, level, ready)
		}
	case MatchRunning:
		for playerIndex, viz := range g.playfieldViz {
			viz.DrawBoardToImage(screen)

			numVirii, _ := g.matchDriver.GetViriiRemaining(playerIndex)
			dropInbound, _ := g.matchDriver.GetIsDropInbound(playerIndex)
			viz.DrawStatusToImage(screen, numVirii, g.matchDriver.GetNextPill(playerIndex),
				dropInbound)
		}
	case MatchPaused:
		for playerIndex, pv := range g.playfieldViz {
			pv.DrawPausedToImage(screen, g.pausePlayerIndex == playerIndex)
		}
	case MatchEnded:
		for playerIndex, pv := range g.playfieldViz {
			matchWinner := playerIndex == g.matchDriver.winner
			pv.DrawResultToImage(screen, matchWinner, false)
		}
	}

	// Write your game's rendering.
	// g.playfieldViz.DrawBoardToImage(screen)
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480
}
