package main

import (
	"fmt"
	"image/color"
	"math"

	"example.com/drbreakboard"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// Functions are NOT thread safe and assume
// that board updates and board draws are not run simultaneously

type playfieldState struct {
	space drbreakboard.Space
	image *ebiten.Image
}

type playfieldViz struct {
	imageMap      imageMap
	fontMap       fontMap
	xBuffer       int
	yBuffer       int
	xPixelSize    int
	yPixelSize    int
	playfieldY    int
	xOffset       int
	yOffset       int
	statusY       int
	fieldState    [][]playfieldState
	nextPillState [2]playfieldState
}

func NewPlayfieldViz(gameImageMap imageMap, gameFontMap fontMap) *playfieldViz {
	viz := &playfieldViz{}

	viz.imageMap = gameImageMap
	viz.fontMap = gameFontMap

	return viz
}

// set the playfield size in pixels
func (viz *playfieldViz) SetPixelSizeAndOffset(x int, y int, xOffset int, yOffset int) {
	viz.xBuffer = x / 50  // 2% for bufferPixels on each side
	viz.yBuffer = y / 100 // 1% for bufferPixels below and above the status board

	viz.xPixelSize = x - 2*viz.xBuffer
	viz.yPixelSize = y - viz.yBuffer
	viz.playfieldY = 2 * (viz.yPixelSize / 3)
	viz.statusY = viz.yPixelSize/3 - viz.yBuffer // sub out second yBuffer for board bottom border
	viz.xOffset = xOffset
	viz.yOffset = yOffset
}

func (viz *playfieldViz) DrawWaitingPlayerToImage(image *ebiten.Image, playerLevel int, ready bool) {
	// draw left border
	geom := ebiten.GeoM{}
	geom.Scale(float64(viz.xBuffer), float64(viz.yPixelSize))
	geom.Translate(float64(viz.xOffset), float64(viz.yOffset))
	pxImage := viz.imageMap["greenPixel"]
	newImg := ebiten.NewImageFromImage(pxImage)
	image.DrawImage(newImg, &ebiten.DrawImageOptions{GeoM: geom})

	// draw right border
	geom = ebiten.GeoM{}
	geom.Scale(float64(viz.xBuffer), float64(viz.yPixelSize))
	geom.Translate(float64(viz.xOffset+viz.xBuffer+viz.xPixelSize), float64(viz.yOffset))
	image.DrawImage(newImg, &ebiten.DrawImageOptions{GeoM: geom})

	text.Draw(image, "Joined!", viz.fontMap["base"], viz.xOffset+viz.xBuffer, viz.yOffset+viz.yPixelSize/3,
		color.RGBA{128, 128, 128, 255})

	text.Draw(image, fmt.Sprintf("Level: %d", playerLevel), viz.fontMap["base"], viz.xOffset+viz.xBuffer, viz.yOffset+viz.yPixelSize/3+30,
		color.RGBA{128, 128, 128, 255})

	if !ready {
		text.Draw(image, "Press Button\nWhen Ready", viz.fontMap["base"], viz.xOffset+viz.xBuffer, viz.yOffset+viz.yPixelSize/3+60,
			color.RGBA{128, 128, 128, 255})
	} else {
		text.Draw(image, "Ready!", viz.fontMap["base"], viz.xOffset+viz.xBuffer, viz.yOffset+viz.yPixelSize/3+60,
			color.RGBA{128, 128, 128, 255})
	}
}

func (viz *playfieldViz) DrawBoardToImage(image *ebiten.Image) {
	// no field means nothing to draw, just return
	if viz.fieldState == nil {
		return
	}

	// draw left border
	geom := ebiten.GeoM{}
	geom.Scale(float64(viz.xBuffer), float64(viz.yPixelSize))
	geom.Translate(float64(viz.xOffset), float64(viz.yOffset))
	pxImage := viz.imageMap["greenPixel"]
	newImg := ebiten.NewImageFromImage(pxImage)
	image.DrawImage(newImg, &ebiten.DrawImageOptions{GeoM: geom})

	// draw right border
	geom = ebiten.GeoM{}
	geom.Scale(float64(viz.xBuffer), float64(viz.yPixelSize))
	geom.Translate(float64(viz.xOffset+viz.xBuffer+viz.xPixelSize), float64(viz.yOffset))
	image.DrawImage(newImg, &ebiten.DrawImageOptions{GeoM: geom})

	// draw field/status border
	geom = ebiten.GeoM{}
	geom.Scale(float64(2*viz.xBuffer+viz.xPixelSize), float64(viz.yBuffer))
	geom.Translate(float64(viz.xOffset), float64(viz.yOffset+viz.playfieldY))
	image.DrawImage(newImg, &ebiten.DrawImageOptions{GeoM: geom})

	// draw field bottom border
	geom = ebiten.GeoM{}
	geom.Scale(float64(2*viz.xBuffer+viz.xPixelSize), float64(viz.yBuffer))
	geom.Translate(float64(viz.xOffset), float64(viz.yOffset+viz.playfieldY+viz.yBuffer+viz.statusY))
	image.DrawImage(newImg, &ebiten.DrawImageOptions{GeoM: geom})

	// start block draw
	yBlockPx := float64(viz.playfieldY) / float64(len(viz.fieldState))
	xBlockPx := float64(viz.xPixelSize) / float64(len(viz.fieldState[0]))

	// draw blocks
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

				// move block to correct location
				geom.Translate(float64(x)*xBlockPx+float64(viz.xOffset+viz.xBuffer), float64(y)*yBlockPx+float64(viz.yOffset))

				image.DrawImage(space.image, &ebiten.DrawImageOptions{GeoM: geom})
			}
		}
	}
}

func drawPillSpace(space playfieldState, xBlockPx float64, yBlockPx float64, x int, y int, image *ebiten.Image) {
	geom := ebiten.GeoM{}

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

	geom.Translate(float64(x), float64(y))

	image.DrawImage(space.image, &ebiten.DrawImageOptions{GeoM: geom})
}

func (viz *playfieldViz) DrawStatusToImage(image *ebiten.Image, virusCount int,
	nextPill [2]drbreakboard.Space) {
	virusesBoundRect := text.BoundString(viz.fontMap["base"], fmt.Sprintf("Viruses: %d", virusCount))
	firstWordY := virusesBoundRect.Dy()

	text.Draw(image, fmt.Sprintf("Viruses: %d", virusCount), viz.fontMap["base"],
		viz.xOffset+viz.xBuffer, viz.yOffset+viz.playfieldY+viz.yBuffer+firstWordY,
		color.RGBA{128, 128, 128, 255})

	nextBoundRect := text.BoundString(viz.fontMap["base"], "Next:")
	nextX := viz.xOffset + viz.xBuffer
	nextY := viz.yOffset + viz.playfieldY + viz.yBuffer + firstWordY + 30
	text.Draw(image, "Next:", viz.fontMap["base"],
		nextX, nextY,
		color.RGBA{128, 128, 128, 255})

	// see if nextpill has changed
	if viz.nextPillState[0].space != nextPill[0] || viz.nextPillState[1].space != nextPill[1] {
		// nextpill has changed, update next
		viz.nextPillState[0].space = nextPill[0]
		nextImage, _ := viz.getPillImage(nextPill[0])
		viz.nextPillState[0].image = nextImage

		viz.nextPillState[1].space = nextPill[1]
		nextImage, _ = viz.getPillImage(nextPill[1])
		viz.nextPillState[1].image = nextImage
	}

	// draw next
	drawPillSpace(viz.nextPillState[0], 20, 20, nextX+nextBoundRect.Dx(), nextY-nextBoundRect.Dy(), image)
	drawPillSpace(viz.nextPillState[1], 20, 20, nextX+nextBoundRect.Dx()+20, nextY-nextBoundRect.Dy()-1, image)
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
