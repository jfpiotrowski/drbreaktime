package main

import (
	"errors"
	"math/rand"
	"time"

	"example.com/drbreakboard"
)

// ticks by 10 pieces dropped at 30fps
var medTicksPerIter = [...]int{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 5, 5, 5, 5, 5, 4, 4, 4, 4, 4, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2}

const fallTick = 7 // fall rate frames at 30 fps

type PlayerAction int
type GameResult int

const (
	Start PlayerAction = iota
	PlacingPill
	Evaluate
	ReadyForNext
	FilledBoard
	VirusesCleared
)

const (
	Cleared GameResult = iota
	Filled
)

type playerState struct {
	nextPill       [2]drbreakboard.Space // 2 spaces, first is the main piece, second linked
	activePill     [2]drbreakboard.Space // 2 spaces, first is the main piece, second linked
	pillPosition   [2]int
	pillRand       *rand.Rand
	playfield      *drbreakboard.PlayField
	ticksSinceIter int
	piecesDropped  int
	currentAction  PlayerAction
	level          int
	ready          bool
}

type matchDriver struct {
	matchStarted   bool
	matchEnded     bool
	playerStates   []*playerState
	matchRand      rand.Source
	playerFinishes []playerFinish
}

type playerFinish struct {
	playerIndex int
	result      GameResult
}

func NewMatchDriver() *matchDriver {
	md := &matchDriver{}
	md.matchRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	md.playerStates = make([]*playerState, 0)

	return md
}

func (md *matchDriver) AddPlayer() {
	newPlayerState := &playerState{}
	newPlayerState.level = 10

	md.playerStates = append(md.playerStates, newPlayerState)
}

func (md *matchDriver) ChangeLevel(playerIndex int, changeAmount int) error {
	if playerIndex < 0 || playerIndex >= len(md.playerStates) {
		return errors.New("playerindex not in range")
	}

	md.playerStates[playerIndex].level += changeAmount

	// put level between 0 and 20 inclusive
	if md.playerStates[playerIndex].level < 0 {
		md.playerStates[playerIndex].level = 0
	}

	if md.playerStates[playerIndex].level > 20 {
		md.playerStates[playerIndex].level = 20
	}

	return nil
}

func (md *matchDriver) GetLevel(playerIndex int) (int, error) {
	if playerIndex < 0 || playerIndex >= len(md.playerStates) {
		return 0, errors.New("playerindex not in range")
	}

	return md.playerStates[playerIndex].level, nil
}

func (md *matchDriver) SetPlayerReady(playerIndex int, ready bool) error {
	if playerIndex < 0 || playerIndex >= len(md.playerStates) {
		return errors.New("playerindex not in range")
	}

	md.playerStates[playerIndex].ready = ready

	return nil
}

func (md *matchDriver) GetPlayerReady(playerIndex int) (bool, error) {
	if playerIndex < 0 || playerIndex >= len(md.playerStates) {
		return false, errors.New("playerindex not in range")
	}

	return md.playerStates[playerIndex].ready, nil
}

func (md *matchDriver) GetViriiRemaining(playerIndex int) (int, error) {
	if playerIndex < 0 || playerIndex >= len(md.playerStates) {
		return 0, errors.New("playerindex not in range")
	}

	viriiRemaining := 0

	// scan through cols for 3 same color in a row
	playfield := md.GetPlayfield(playerIndex)
	for col := 0; col < playfield.GetWidth(); col++ {
		for row := 0; row < playfield.GetHeight(); row++ {
			space, _ := playfield.GetSpaceAtCoordinate(row, col)
			if space.Content == drbreakboard.Virus {
				viriiRemaining++
			}
		}
	}

	return viriiRemaining, nil
}

func (md *matchDriver) StartMatch() {
	if md.matchStarted && !md.matchEnded {
		// match already started or has not ended, return
		return
	}

	md.playerFinishes = make([]playerFinish, 0)

	// pick the seed for the match to sync random number generators
	// ensures same board, pills, etc.
	matchSeed := md.matchRand.Int63()

	// set up each playerstate for a new match
	for _, playerState := range md.playerStates {
		// initialize player board
		playerState.playfield = drbreakboard.NewPlayField(8, 16)
		populateBoardViruses(playerState.playfield, playerState.level, matchSeed)

		playerState.pillRand = rand.New(rand.NewSource(matchSeed))
		playerState.nextPill[0], playerState.nextPill[1] = generatePill(playerState.pillRand)
	}

	md.matchStarted = true
}

func (md *matchDriver) ApplyInputs(playerInputs map[int][]GamepadEvent) {
	for index, ps := range md.playerStates {
		if ps.currentAction != PlacingPill {
			// if not placing pill, input means nothing
			continue
		}

		playerInput, hasInput := playerInputs[index]
		if !hasInput {
			// no input for that player, skip
			continue
		}

		for _, input := range playerInput {
			switch input {
			case LeftJustPressed:
				md.moveLeftIfPossible(index, ps)
			case RightJustPressed:
				md.moveRightIfPossible(index, ps)
			case PrimaryJustPressed:
				md.rotateIfPossible(index, ps, true)
			case SecondaryJustPressed:
				md.rotateIfPossible(index, ps, false)
			}
		}
	}
}

func (md *matchDriver) rotateIfPossible(index int, ps *playerState, clockwise bool) {
	if ps.currentAction != PlacingPill {
		// if not placing pill, there's no pill to rotate
		return
	}

	if ps.activePill[0].Linkage == drbreakboard.Up {
		// piece is vertical trying to go horizontal
		md.rotateVertToHor(index, ps, clockwise)
	} else if ps.activePill[0].Linkage == drbreakboard.Right {
		// piece is horizontal trying to go vertical
		md.rotateHorToVert(index, ps, clockwise)
	}
}

func (md *matchDriver) rotateHorToVert(index int, ps *playerState, clockwise bool) {
	// first, see if spot above primary is open and go there
	aboveRoot, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0]-1, ps.pillPosition[1])
	if err == nil && aboveRoot.Content == drbreakboard.Empty {
		// space is open, rotate there
		if clockwise {
			// root piece becomes the linked piece
			tempSpace := ps.activePill[0]
			ps.activePill[0] = ps.activePill[1]
			ps.activePill[1] = tempSpace
		}
		// make the linkage vertical
		ps.activePill[0].Linkage = drbreakboard.Up
		ps.activePill[1].Linkage = drbreakboard.Down
		return // we made the move, done
	}

	// if we get here we couldn't do a basic rotation
	// next look at space above the linked piece
	aboveLinked, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0]-1, ps.pillPosition[1]+1)
	if err == nil && aboveLinked.Content == drbreakboard.Empty {
		// linked space becomes the new root piece location
		ps.pillPosition[1] = ps.pillPosition[1] + 1

		if clockwise {
			tempSpace := ps.activePill[0]
			ps.activePill[0] = ps.activePill[1]
			ps.activePill[1] = tempSpace
		}

		ps.activePill[0].Linkage = drbreakboard.Up
		ps.activePill[1].Linkage = drbreakboard.Down
		return // made the move
	}

	// next look at space below the root space
	belowRoot, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0]+1, ps.pillPosition[1])
	if err == nil && belowRoot.Content == drbreakboard.Empty {
		// piece moves downwards
		ps.pillPosition[0] = ps.pillPosition[0] + 1

		if clockwise {
			tempSpace := ps.activePill[0]
			ps.activePill[0] = ps.activePill[1]
			ps.activePill[1] = tempSpace
		}

		ps.activePill[0].Linkage = drbreakboard.Up
		ps.activePill[1].Linkage = drbreakboard.Down
		return // made the move
	}

	// next look at space below the linked space
	belowLinked, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0]+1, ps.pillPosition[1]+1)
	if err == nil && belowLinked.Content == drbreakboard.Empty {
		// piece moves downwards
		ps.pillPosition[0] = ps.pillPosition[0] + 1
		ps.pillPosition[1] = ps.pillPosition[1] + 1

		if clockwise {
			tempSpace := ps.activePill[0]
			ps.activePill[0] = ps.activePill[1]
			ps.activePill[1] = tempSpace
		}

		ps.activePill[0].Linkage = drbreakboard.Up
		ps.activePill[1].Linkage = drbreakboard.Down
	}

	// no valid spots return having done nothing
}

func (md *matchDriver) rotateVertToHor(index int, ps *playerState, clockwise bool) {
	// first, see if spot to the left is open and move the piece there
	rightOfPiece, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0], ps.pillPosition[1]+1)
	if err == nil && rightOfPiece.Content == drbreakboard.Empty {
		// space to right is open, rotate there
		if !clockwise {
			// root piece becomes the linked piece
			tempSpace := ps.activePill[0]
			ps.activePill[0] = ps.activePill[1]
			ps.activePill[1] = tempSpace
		}
		// make the linkage horizontal
		ps.activePill[0].Linkage = drbreakboard.Right
		ps.activePill[1].Linkage = drbreakboard.Left
		return // we made the move, done
	}

	// if we get here we couldn't do a basic rotation
	// look at the left space and "kick" off of the right obstruction
	leftOfPiece, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0], ps.pillPosition[1]-1)
	if err == nil && leftOfPiece.Content == drbreakboard.Empty {
		// left space becomes the new root piece location
		ps.pillPosition[1] = ps.pillPosition[1] - 1

		if !clockwise {
			tempSpace := ps.activePill[0]
			ps.activePill[0] = ps.activePill[1]
			ps.activePill[1] = tempSpace
		}

		ps.activePill[0].Linkage = drbreakboard.Right
		ps.activePill[1].Linkage = drbreakboard.Left
	}
}

func (md *matchDriver) moveLeftIfPossible(index int, ps *playerState) {
	leftOfPiece, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0], ps.pillPosition[1]-1)

	// out of bounds
	if err != nil {
		return
	}

	// left isn't empty
	if leftOfPiece.Content != drbreakboard.Empty {
		return
	}

	// same check for piece above if piece oriented vertically
	// represented with an up linkage on the primary space
	if ps.activePill[0].Linkage == drbreakboard.Up {
		leftOfLinked, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0]-1, ps.pillPosition[1]-1)
		if err != nil {
			return
		}

		if leftOfLinked.Content != drbreakboard.Empty {
			return
		}
	}

	ps.pillPosition[1] -= 1
}

func (md *matchDriver) moveRightIfPossible(index int, ps *playerState) {

	// same check for piece above if piece oriented vertically
	// represented with an up linkage on the primary space
	if ps.activePill[0].Linkage == drbreakboard.Up {
		// linkage is up, need to check to right of both in stack
		rightOfBottomHalf, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0], ps.pillPosition[1]+1)

		// out of bounds
		if err != nil {
			return
		}

		// right isn't empty
		if rightOfBottomHalf.Content != drbreakboard.Empty {
			return
		}

		rightOfTopHalf, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0]-1, ps.pillPosition[1]+1)
		if err != nil {
			return
		}

		if rightOfTopHalf.Content != drbreakboard.Empty {
			return
		}
	} else {
		// linkage is right, need to check primary piece location + 2
		rightOfWholePill, err := md.GetPlayfield(index).GetSpaceAtCoordinate(ps.pillPosition[0], ps.pillPosition[1]+2)

		// out of bounds
		if err != nil {
			return
		}

		// right isn't empty
		if rightOfWholePill.Content != drbreakboard.Empty {
			return
		}
	}

	ps.pillPosition[1] += 1
}

func (md *matchDriver) ApplyTick(playerInputs map[int][]GamepadEvent) {
	for playerIndex, ps := range md.playerStates {
		tickRateIndex := ps.piecesDropped / 10
		if tickRateIndex >= len(medTicksPerIter) {
			tickRateIndex = len(medTicksPerIter) - 1
		}
		iterTicks := medTicksPerIter[tickRateIndex] * 2 //convert to 60 ticks per sec

		switch ps.currentAction {
		case Start, ReadyForNext:
			// check that there's a place to put the piece
			left, _ := ps.playfield.GetSpaceAtCoordinate(0, 3)
			right, _ := ps.playfield.GetSpaceAtCoordinate(0, 4)
			if left.Content != drbreakboard.Empty || right.Content != drbreakboard.Empty {
				// board is full, you lose
				md.playerFinishes = append(md.playerFinishes, playerFinish{playerIndex, Filled})
				ps.currentAction = FilledBoard
			} else {
				// put the piece into play
				ps.activePill = ps.nextPill
				ps.nextPill[0], ps.nextPill[1] = generatePill(ps.pillRand)

				// put the pill in row 0, middle column
				ps.pillPosition[0] = 0
				ps.pillPosition[1] = 3

				ps.currentAction = PlacingPill
			}
		case FilledBoard:
			// no-op, lost the game
		case PlacingPill:
			dropIterTicks := iterTicks
			const controllerHoldIterTicks = 5 // tenth of a second

			// if down is held, lower the tick wait threshold
			playerInput, hasInput := playerInputs[playerIndex]
			if hasInput {
				for _, command := range playerInput {
					if command == DownPressed && iterTicks > controllerHoldIterTicks {
						dropIterTicks = controllerHoldIterTicks
					}
				}
			}

			if ps.ticksSinceIter >= dropIterTicks {
				// time to drop the pill
				dropBlocked := false
				space, err := ps.playfield.GetSpaceAtCoordinate(ps.pillPosition[0]+1, ps.pillPosition[1])
				if err != nil {
					// bounds check failed due to bottom
					dropBlocked = true
				}

				if space.Content != drbreakboard.Empty {
					// there's a thing in the space
					dropBlocked = true
				}

				// check the linked spot
				linkedY, linkedX, _ := drbreakboard.GetLinkedCoordinate(ps.pillPosition[0], ps.pillPosition[1], ps.activePill[0].Linkage)
				space, err = ps.playfield.GetSpaceAtCoordinate(linkedY+1, linkedX)
				if err != nil {
					// bounds check failed due to bottom
					dropBlocked = true
				}

				if space.Content != drbreakboard.Empty {
					// there's a thing in the space
					dropBlocked = true
				}

				if dropBlocked {
					// something under the piece, stop the drop
					ps.playfield.PutTwoLinkedSpacesAtCoordinate(ps.pillPosition[0], ps.pillPosition[1],
						ps.activePill[0], ps.activePill[1])

					// mark active pill as empty
					ps.activePill[0].Content = drbreakboard.Empty

					// add one to pills dropped
					ps.piecesDropped += 1

					// set state to evaluate to check pill effect
					ps.currentAction = Evaluate

					// execute evaluation immediately
					md.evaluateAndIterateBoard(ps, playerIndex, true)
				} else {
					// not blocked, drop the pill
					ps.pillPosition[0] = ps.pillPosition[0] + 1
				}

				ps.ticksSinceIter = 0
			} else {
				// not ready yet,
				ps.ticksSinceIter++
			}
		case Evaluate:
			// evaluate the dropped pill's effect
			md.evaluateAndIterateBoard(ps, playerIndex, false)
		case VirusesCleared:
			// no-op
		}
	}
}

func (md *matchDriver) evaluateAndIterateBoard(ps *playerState, playerIndex int, ignoreTicks bool) {
	if !ignoreTicks && ps.ticksSinceIter < fallTick {
		// not time for next eval add to tick count
		ps.ticksSinceIter++
		return
	}

	_, nextIteration := ps.playfield.EvaluateBoardIteration()
	if nextIteration == drbreakboard.NoAction {
		// board has no falls or clears
		ps.currentAction = ReadyForNext
	} else {
		// board has activity, iterate and evaluate again
		err := ps.playfield.IterateBoard()
		if err != nil {
			panic("iterate went wrong")
		}

		if nextIteration == drbreakboard.Clear {
			// after clear, see if we still have viruses
			if ps.playfield.GetVirusCount() == 0 {
				// match is over, make the state match
				md.playerFinishes = append(md.playerFinishes, playerFinish{playerIndex, Cleared})
				ps.currentAction = VirusesCleared
			}
		}
	}

	ps.ticksSinceIter = 0
}

func (md *matchDriver) GetPlayfield(playerIndex int) *drbreakboard.PlayField {
	if !md.matchStarted {
		return nil
	}

	if playerIndex >= len(md.playerStates) {
		return nil
	}

	return md.playerStates[playerIndex].playfield
}

func (md *matchDriver) GetActivePill(playerIndex int) [2]drbreakboard.Space {
	return md.playerStates[playerIndex].activePill
}

func (md *matchDriver) GetNextPill(playerIndex int) [2]drbreakboard.Space {
	return md.playerStates[playerIndex].nextPill
}

func (md *matchDriver) GetActivePillLocation(playerIndex int) [2]int {
	return md.playerStates[playerIndex].pillPosition
}

func populateBoardViruses(playfield *drbreakboard.PlayField, level int, seedInt int64) {
	// 20 is max level
	if level > 20 {
		level = 20
	}

	virusRows := 9
	numVirii := level*4 + 4
	if level >= 15 {
		virusRows = 9 + (level-13)/2
	}

	// use a seedInt so players get same board
	virusRand := rand.New(rand.NewSource(seedInt))

	// create slice of virii
	colorSlice := make([]drbreakboard.SpaceColor, numVirii)
	for i := range colorSlice {
		colorSlice[i] = drbreakboard.Red
		if i%3 == 1 {
			colorSlice[i] = drbreakboard.Blue
		} else if i%3 == 2 {
			colorSlice[i] = drbreakboard.Yellow
		}
	}

	success := false

	for !success {
		// shuffle the viruses for random placement
		virusRand.Shuffle(len(colorSlice), func(i, j int) {
			colorSlice[i], colorSlice[j] = colorSlice[j], colorSlice[i]
		})
		playfield.ClearBoard()
		populateViriiCanonical(playfield, virusRows, numVirii, virusRand)

		// board should always validate with canonical placement
		success = validateBoard(playfield)
	}
}

func populateViriiCanonical(playfield *drbreakboard.PlayField, virusRows int, numVirii int,
	virusRand *rand.Rand) {
	maxRow := playfield.GetBottomRowIndex()
	minRow := maxRow - (virusRows - 1)

	virusTypes := []drbreakboard.SpaceColor{drbreakboard.Yellow, drbreakboard.Red, drbreakboard.Blue}

	// while there are still virii to place
	for numVirii > 0 {
		xPos := virusRand.Intn(8)

		yPos := 0

		// pull random numbers until yPos is a valid virus row
		for yPos < minRow {
			yPos = virusRand.Intn(16)
		}

		// weird logic
		virusIndex := numVirii % 4
		if virusIndex == 3 {
			seed := virusRand.Int()
			if ((seed / 3) % 2) == 0 {
				virusIndex = seed % 3
			} else {
				virusIndex = 2 - (seed % 3)
			}
		}

		for {
			space, err := playfield.GetSpaceAtCoordinate(yPos, xPos)
			if err != nil {
				// ypos out of bounds, need to restart
				break
			}

			if space.Content == drbreakboard.Empty {
				// empty spot, break
				break
			}

			// move down and diagonal
			yPos = yPos + 1
			xPos = (xPos + 1) % playfield.GetWidth()
		}

		if yPos > maxRow {
			// out of rows to try, go back to head to try again
			continue
		}

		adjMap := make(map[drbreakboard.SpaceColor]bool)

		for {
			adjMap[drbreakboard.Red] = false
			adjMap[drbreakboard.Blue] = false
			adjMap[drbreakboard.Yellow] = false

			space, err := playfield.GetSpaceAtCoordinate(yPos, xPos-2)
			if err == nil {
				adjMap[space.Color] = true
			}

			space, err = playfield.GetSpaceAtCoordinate(yPos, xPos+2)
			if err == nil {
				adjMap[space.Color] = true
			}

			space, err = playfield.GetSpaceAtCoordinate(yPos+2, xPos)
			if err == nil {
				adjMap[space.Color] = true
			}

			space, err = playfield.GetSpaceAtCoordinate(yPos-2, xPos)
			if err == nil {
				adjMap[space.Color] = true
			}

			// at least one color has no 2nd neighbor
			// set virus index to one of those colors
			if !adjMap[drbreakboard.Red] || !adjMap[drbreakboard.Blue] || !adjMap[drbreakboard.Yellow] {
				for adjMap[virusTypes[virusIndex]] {
					// set to the first non-match
					if virusIndex == 0 {
						virusIndex = 2
					} else {
						virusIndex--
					}
				}

				break
			}

			// this xpos, ypos is a dead spot, keep going down the diagonal
			yPos = yPos + 1
			xPos = (xPos + 1) % playfield.GetWidth()

			// repeat of the test above
			// maybe combine logic sometime
			for {
				space, err := playfield.GetSpaceAtCoordinate(yPos, xPos)
				if err != nil {
					// ypos out of bounds, need to restart
					break
				}

				if space.Content == drbreakboard.Empty {
					// empty spot, break
					break
				}

				// move down and diagonal
				yPos = yPos + 1
				xPos = (xPos + 1) % playfield.GetWidth()
			}

			if yPos > maxRow {
				// out of rows to try, go back to head to try again
				continue
			}
		}

		if yPos <= playfield.GetBottomRowIndex() {
			virus, _ := drbreakboard.MakeVirus(virusTypes[virusIndex])
			playfield.PutSpaceAtCoordinateIfEmpty(yPos, xPos, virus)
			numVirii--
		}
	}
}

// a valid board has no more than 2 colors in a row vertically or horizontially
func validateBoard(playfield *drbreakboard.PlayField) bool {

	// scan through rows for 3 same color in group of 4
	for row := 0; row < playfield.GetHeight(); row++ {
		for col := 0; col < playfield.GetWidth(); col++ {
			var colorCount [4]int
			for i := col - 3; i <= col; i++ {
				space, err := playfield.GetSpaceAtCoordinate(row, i)

				if err != nil {
					// out of bounds, treat as uncolored
					space = drbreakboard.Space{}
				}
				colorCount[space.Color]++
			}

			if colorCount[1] >= 3 || colorCount[2] >= 3 || colorCount[3] >= 3 {
				return false
			}
		}
	}

	// scan through cols for 3 same color in a row
	for col := 0; col < playfield.GetWidth(); col++ {
		for row := 0; row < playfield.GetHeight(); row++ {
			var colorCount [4]int
			for i := row - 3; i <= row; i++ {
				space, err := playfield.GetSpaceAtCoordinate(i, col)

				if err != nil {
					// out of bounds, treat as uncolored
					space = drbreakboard.Space{}
				}
				colorCount[space.Color]++
			}

			if colorCount[1] >= 3 || colorCount[2] >= 3 || colorCount[3] >= 3 {
				return false
			}
		}
	}

	return true
}

func generatePill(pillRand *rand.Rand) (drbreakboard.Space, drbreakboard.Space) {
	virusTypes := []drbreakboard.SpaceColor{drbreakboard.Yellow, drbreakboard.Red, drbreakboard.Blue}

	primary := pillRand.Intn(3)
	linked := pillRand.Intn(3)

	a, b, _ := drbreakboard.MakeLinkedPillSpaces(drbreakboard.Right, virusTypes[primary], virusTypes[linked])
	return a, b
}
