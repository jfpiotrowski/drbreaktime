package main

import (
	"math"

	"example.com/drbreakboard"
	"github.com/hajimehoshi/ebiten/v2"
)

// Functions are NOT thread safe and assume
// that board updates and board draws are not run simultaneously

type playfieldState struct {
	space drbreakboard.Space
	image *ebiten.Image
}

type playfieldViz struct {
	imageMap   imageMap
	xPixelSize int
	yPixelSize int
	fieldState [][]playfieldState
}

func NewPlayfieldViz(gameImageMap imageMap) *playfieldViz {
	viz := &playfieldViz{}

	viz.imageMap = gameImageMap

	return viz
}

// set the playfield size in pixels
func (viz *playfieldViz) SetPixelSize(x int, y int) {
	viz.xPixelSize = x
	viz.yPixelSize = y
}

func (viz *playfieldViz) DrawBoardToImage(image *ebiten.Image) {
	// no field means nothing to draw, just return
	if viz.fieldState == nil {
		return
	}

	yBlockPx := float64(viz.yPixelSize) / float64(len(viz.fieldState))
	xBlockPx := float64(viz.xPixelSize) / float64(len(viz.fieldState[0]))

	for y, row := range viz.fieldState {
		for x, space := range row {
			// if we have an assigned image for the space, draw it in the block
			if space.image != nil {
				geom := ebiten.GeoM{}

				// scale to size
				imgX, imgY := space.image.Size()
				geom.Scale(xBlockPx/float64(imgX), yBlockPx/float64(imgY))

				if space.space.Linkage != drbreakboard.Unlinked {
					geom.Translate(xBlockPx/-2, yBlockPx/-2)
					switch space.space.Linkage {
					case drbreakboard.Left:
						geom.Rotate(math.Pi / 2)
					case drbreakboard.Up:
						geom.Rotate(math.Pi)
					case drbreakboard.Right:
						geom.Rotate(-math.Pi / 2)
					}
					geom.Translate(xBlockPx/2, yBlockPx/2)

				}

				// scale to block size
				geom.Translate(float64(x)*xBlockPx, float64(y)*yBlockPx)

				image.DrawImage(space.image, &ebiten.DrawImageOptions{GeoM: geom})
			}
		}
	}
}

func (viz *playfieldViz) UpdateBoard(playfield *drbreakboard.PlayField,
	activePill [2]drbreakboard.Space, activePillLocation [2]int) {
	// no field means this is the first board setup
	// allocate the board
	if viz.fieldState == nil {
		viz.fieldState = make([][]playfieldState, playfield.GetHeight())
		for i := range viz.fieldState {
			viz.fieldState[i] = make([]playfieldState, playfield.GetWidth())
		}
	}

	for y := 0; y < playfield.GetHeight(); y++ {
		for x := 0; x < playfield.GetWidth(); x++ {
			space, _ := playfield.GetSpaceAtCoordinate(y, x)
			vizSpace := &viz.fieldState[y][x]
			if space != vizSpace.space {
				// space has changed, need to update image
				if space.Content == drbreakboard.Empty {
					// space is now empty, clear out the stored image
					vizSpace.image = nil
				}

				var newImg *ebiten.Image
				var err error

				if space.Content == drbreakboard.Virus {
					switch space.Color {
					case drbreakboard.Blue:
						newImg = ebiten.NewImageFromImage(viz.imageMap["blueVirus"])
					case drbreakboard.Red:
						newImg = ebiten.NewImageFromImage(viz.imageMap["redVirus"])
					case drbreakboard.Yellow:
						newImg = ebiten.NewImageFromImage(viz.imageMap["yellowVirus"])
					default:
						panic("virus had bad color")
					}
				}

				if space.Content == drbreakboard.Pill {
					newImg, err = viz.getPillImage(space)
				}

				// TODO: implement pill draw

				if err == nil {
					vizSpace.space = space
					vizSpace.image = newImg
				} else {
					panic("something terrible with images")
				}
			}
		}
	}

	if activePill[0].Content != drbreakboard.Empty {
		// we have an active pill, draw it too
		newImg, _ := viz.getPillImage(activePill[0])
		viz.fieldState[activePillLocation[0]][activePillLocation[1]].space = activePill[0]
		viz.fieldState[activePillLocation[0]][activePillLocation[1]].image = newImg

		newImg, _ = viz.getPillImage(activePill[1])
		linkedY, linkedX, _ := drbreakboard.GetLinkedCoordinate(activePillLocation[0], activePillLocation[1], activePill[0].Linkage)
		viz.fieldState[linkedY][linkedX].space = activePill[1]
		viz.fieldState[linkedY][linkedX].image = newImg
	}
}

func (viz *playfieldViz) getPillImage(space drbreakboard.Space) (*ebiten.Image, error) {
	var newImg *ebiten.Image
	var err error

	if space.Linkage == drbreakboard.Unlinked {
		switch space.Color {
		case drbreakboard.Blue:
			newImg = ebiten.NewImageFromImage(viz.imageMap["blueSingle"])
		case drbreakboard.Red:
			newImg = ebiten.NewImageFromImage(viz.imageMap["redSingle"])
		case drbreakboard.Yellow:
			newImg = ebiten.NewImageFromImage(viz.imageMap["yellowSingle"])
		default:
			panic("pill had bad color")
		}
	} else {
		switch space.Color {
		case drbreakboard.Blue:
			newImg = ebiten.NewImageFromImage(viz.imageMap["blueLinked"])
		case drbreakboard.Red:
			newImg = ebiten.NewImageFromImage(viz.imageMap["redLinked"])
		case drbreakboard.Yellow:
			newImg = ebiten.NewImageFromImage(viz.imageMap["yellowLinked"])
		default:
			panic("pill had bad color")
		}
	}
	return newImg, err
}
