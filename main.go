package main

import (
	"fmt"
	"github.com/gen2brain/raylib-go/raylib"
	"crypto/rand"
	"math"
	"math/big"
	//"time"
)

type i64  = int64
type i32  = int32 
type f64  = float64
type f32  = float32
type u8   = uint8
type Vec2 = rl.Vector2

func Vec2Add(v1, v2 Vec2) Vec2 { return Vec2{v1.X + v2.X, v1.Y + v2.Y} }

func Vec2Sub(v1, v2 Vec2) Vec2 { return Vec2{v1.X - v2.X, v1.Y - v2.Y} }

func Vec2Div(v Vec2, f f32) Vec2 { return Vec2{v.X / f, v.Y / f} }

func Vec2Neg(v Vec2) Vec2 { return Vec2{-v.X, -v.Y} }

func Vec2LenSq(v Vec2) f32 { return v.X * v.X + v.Y * v.Y }

func Vec2Len(v Vec2) f64 { return math.Sqrt(f64(Vec2LenSq(v))) }

func Vec2DistSq(v1, v2 Vec2) f32 { return Vec2LenSq(Vec2Sub(v1, v2)) }

func Vec2Dist(v1, v2 Vec2) f64 { return math.Sqrt(f64(Vec2DistSq(v1, v2))) }

func assert(b bool, msg string) { if !b { panic("Assert failed: " + msg + "!\n") } }

const (
	WINDOW_WIDTH      = 560
	WINDOW_HEIGHT     = 800
	UPPER_LAND_HEIGHT = 560
	MARGIN_HEIGHT     = 20
	MARGIN_WIDTH      = 20
	MSG_BOARD_HEIGHT  = 40
	NUM_ROW           = 4
	NUM_COL           = 4
	ROW_HEIGHT        = (UPPER_LAND_HEIGHT - 2 * MARGIN_HEIGHT) / NUM_ROW 
	COL_WIDTH         = (WINDOW_WIDTH - 2 * MARGIN_WIDTH) / NUM_COL
	ANIM_SIZE f32     = ROW_HEIGHT * 0.65
	MIN_ANIM_HEIGHT	  =	5
	MIN_JUMP_HEIGHT   = MIN_ANIM_HEIGHT * 3
	JUMP_SCALE_INC_RATE = 0.075
	MAX_DUST_DURATION = 20
	RESQUE_SPOT_X     = MARGIN_WIDTH + (WINDOW_WIDTH - 2 * MARGIN_WIDTH) / 2 
	RESQUE_SPOT_Y     = (UPPER_LAND_HEIGHT + 7 * MARGIN_HEIGHT) + 
						 (WINDOW_HEIGHT - (UPPER_LAND_HEIGHT + 7 * MARGIN_HEIGHT)) / 2 
	BOARD_SIZE        = NUM_ROW * NUM_COL
	FRONT_ROW_BASEINDEX = BOARD_SIZE - NUM_COL
	NUM_COLOR         = 4
	NUM_KIND          = 4

	FPS = 60

	// Raylib input int32 map
	KEY_A = 65
	KEY_S = 83
	KEY_D = 68
	KEY_F = 70
	KEY_G = 71
	KEY_Q = 81
	KEY_SPACE = 32
	MOUSE_LEFT = 0
	MOUSE_RIGHT = 1
	KEY_RIGHT = 262
	KEY_LEFT = 263
	KEY_DOWN = 264
	KEY_UP = 265
)

// accel, veloc and press in pixels/frame.
type Animal struct {
	pos Vec2
	dest Vec2
	accel Vec2
	veloc Vec2
	rot i32
	scale f32
	height f32
	press f32
	scaleDecRate f32
	dustDuration u8
	totalJumpFrames u8
	ascFrames u8
	currJumpFrame u8
	animType u8
}

func setAnimals(animals *[BOARD_SIZE]Animal) {
	color, kind := u8(1), u8(1)

	for row := 0; row < NUM_ROW; row++ {
		colorBit := color << NUM_KIND
		for col := 0; col < NUM_COL; col++ {
			boardIndex := row * NUM_COL + col
			animals[boardIndex].height = ANIM_SIZE 
			animals[boardIndex].animType = colorBit | kind
			animals[boardIndex].scale = 1
			kind <<= 1
		}
		kind = 1
		color <<= 1
	}
}

func shuffleBoard(board *[BOARD_SIZE]*Animal, frontRowPos *[NUM_COL]Vec2) {
	for i := 0; i < BOARD_SIZE - 2; i++ {
		rangeToLastIndex := i64(BOARD_SIZE - 1 - i)
		randomIndex, _ := rand.Int(rand.Reader, big.NewInt(rangeToLastIndex))
        indexToSwap :=  i + 1 + int(randomIndex.Uint64())
        board[i], board[indexToSwap] = board[indexToSwap], board[i]
    }
	board[0], board[BOARD_SIZE - 1] = board[BOARD_SIZE - 1], board[0]

    for row := 0; row < NUM_COL; row++ {
        posY := f32(MARGIN_HEIGHT + (row * ROW_HEIGHT) + (ROW_HEIGHT / 2))
        for col := 0; col < NUM_ROW; col++ {
            boardIndex := row * NUM_COL + col
            posX := f32(MARGIN_WIDTH + (col * COL_WIDTH) + (COL_WIDTH / 2))
            board[boardIndex].dest = Vec2{posX, posY}
            // remove setting pos to dest after implementing the movement animation 
            board[boardIndex].pos = board[boardIndex].dest 
        }
    }
	for i := 0; i < NUM_COL; i++ {
		frontRowIndex := i + BOARD_SIZE - NUM_COL
		frontRowPos[i] = board[frontRowIndex].dest
	}
}

func findFirst1Bit(target u8) int {
    var order int
	b := u8(1)
    for target & b == 0 { 
        b <<= 1
        order++
    }
    return order
}

// Returns the number of animals that could be chosen from the front row(last NUM_COL elements of 
// the board) and fills the nextAnimIndexes array with the indexes of them

func findResquables(board *[BOARD_SIZE]*Animal, mostRecentRescuedType u8,
					nextAnimIndexes *[NUM_COL]int) int {
    var nextAnimNum int
    frontRowOffset := BOARD_SIZE - NUM_COL

    for i := 0; i < NUM_COL; i++ {
        if board[i + frontRowOffset] != nil && 
           (board[i + frontRowOffset].animType & mostRecentRescuedType != 0) {
            nextAnimIndexes[i] = i + frontRowOffset
            nextAnimNum++
        }
    }

    return nextAnimNum
}

func resqueAt(board, resqued *[BOARD_SIZE]*Animal, resqueIndex, numAnimalLeft int) {
    i := resqueIndex			  
    anim := board[i]
    assert(anim.animType != 0, "animal to be resque has type 0")

    ascFrames := 2.5 * ANIM_SIZE/anim.height
    totalFrames := 20 + ascFrames
    if ascFrames < 4  { ascFrames, totalFrames = 4, 20 }
    jumpAnimal(anim, Vec2{RESQUE_SPOT_X, RESQUE_SPOT_Y}, totalFrames, ascFrames)
    resqued[BOARD_SIZE - numAnimalLeft] = anim

	// Advance the row where the selected animal is at 
    for i >= 0 && board[i] != nil {
        if i < NUM_COL || board[i - NUM_COL] == nil { 
            board[i] = nil
            break
        } else {
            board[i] = board[i - NUM_COL]
            anim := board[i]
            jumpAnimal(anim, Vec2{anim.pos.X, anim.pos.Y + ROW_HEIGHT}, 20, 6)
		}
        i -= NUM_COL
    }
}

// totalFrames: the total duration of the jump in frames.
// ascFrames: the duration of animal moving upward ing in frames.
// if dest.Y is greater than anim.pos.Y, ascFrames has to be less than totalFrames/2
// and dest.Y is less than anim.pos.Y, ascFrames has to be greater than totalFrmaes/2
// TotalFrames CANNOT be exactly 2*ascFrames otherwise divided by 0 occurs
func jumpAnimal(anim *Animal, dest Vec2, totalFrames, ascFrames f32) {

	// desFrames: frames left at the its original position while desending to dest
	desFrames := totalFrames - ascFrames
	
	//veloc = -accel * ascFrames (accel is positive)
	//leastAlt = anim.pos.Y + 0.5*ascFrames*veloc (veloc is negative)
	//dest.Y = leastAlt + 0.5*accel*(desFrame)^2
	anim.accel = Vec2{0, 
	                   2*(dest.Y-anim.pos.Y)/(desFrames*desFrames - ascFrames*ascFrames) }
	anim.veloc = Vec2{(dest.X-anim.pos.X)/totalFrames, -anim.accel.Y*ascFrames}
	anim.dest = dest
	anim.totalJumpFrames = u8(totalFrames)
	anim.ascFrames = u8(ascFrames)
	anim.currJumpFrame = 0
	
	if anim.height < ANIM_SIZE { anim.press = -anim.height/10}
}


func drawAnimal(animalsTexture, dustTexture *rl.Texture2D, anim *Animal) {
	colorBitfield := anim.animType >> NUM_KIND
	kindBitfield := anim.animType & 0b1111
	colorOffset := NUM_COLOR - 1 - findFirst1Bit(colorBitfield)
	kindOffset := NUM_KIND - 1 - findFirst1Bit(kindBitfield)
    srcRect := rl.Rectangle{f32(kindOffset) * ANIM_SIZE, f32(colorOffset) * ANIM_SIZE,
                         ANIM_SIZE, ANIM_SIZE}
	
	if anim.totalJumpFrames > 0 {
		if anim.currJumpFrame <= anim.ascFrames {
			anim.scale += JUMP_SCALE_INC_RATE
			if anim.currJumpFrame == anim.ascFrames { 
				anim.scaleDecRate = (anim.scale - 1) / 
									f32(anim.totalJumpFrames - anim.ascFrames)
			}
		} else {
			anim.scale -= anim.scaleDecRate 
			if anim.scale < 1 { anim.scale = 1 }
		}
	}
	
	sc := anim.scale
	animOrigin := Vec2Sub(anim.pos, Vec2{sc*ANIM_SIZE / 2, sc*ANIM_SIZE / 2})
	desPos := Vec2{animOrigin.X, animOrigin.Y + sc*ANIM_SIZE - sc*anim.height}
	desRect := rl.Rectangle{desPos.X, desPos.Y, sc*ANIM_SIZE, sc*anim.height}

	rl.DrawTexturePro(*animalsTexture, srcRect, desRect, Vec2{}, 0, rl.RayWhite)
	
	// Draw dust
	if anim.dustDuration != 0 { 
		srcRect := rl.Rectangle{0, 0, 320, 256}
		desRect := rl.Rectangle{anim.pos.X - ANIM_SIZE*0.7, anim.pos.Y + ANIM_SIZE*0.4, 
								ANIM_SIZE*0.5, ANIM_SIZE*0.15} 
		rl.DrawTexturePro(*dustTexture, srcRect, desRect, Vec2{}, 0, rl.RayWhite)

		desRect = rl.Rectangle{anim.pos.X + ANIM_SIZE*0.2, anim.pos.Y + ANIM_SIZE*0.4,  
								ANIM_SIZE*0.5, ANIM_SIZE*0.15} 
		rl.DrawTexturePro(*dustTexture, srcRect, desRect, Vec2{}, 0, rl.RayWhite)
		anim.dustDuration -= 1
	}
}

func printbd (board *[BOARD_SIZE]*Animal) {
	var boardIndex int
	for i := 0; i < NUM_ROW; i++ {
		for j := 0; j < NUM_COL; j++ {
			if board[boardIndex] == nil {
				fmt.Printf("00000000 ")
			} else {
				fmt.Printf("%08b ", board[boardIndex].animType)
			}
			boardIndex++
		}
		fmt.Printf("\n")
	}
}

func isAnimRectClicked(animal *Animal) bool {
	if animal == nil { return false}

	mouseX := f32(rl.GetMouseX())
	mouseY := f32(rl.GetMouseY())
	animPosX := animal.pos.X
	animPosY := animal.pos.Y
	halfLength := ANIM_SIZE / 2

	//fmt.Printf("Mouse Clicked at : %d, %d\n", mouseX, mouseY)
	//fmt.Printf("AnimPos : %d, %d\n", animPosX, animPosY)

	return mouseX >= animPosX - halfLength && mouseX <= animPosX + halfLength &&
		   mouseY >= animPosY - halfLength && mouseY <= animPosY + halfLength   
}

func loadTextures(groundTexture, animalsTexture, dustTexture *rl.Texture2D) {
    groundImage := rl.LoadImage("textures/background.png")
    animalsImage := rl.LoadImage("textures/animals.png")
    dustImage := rl.LoadImage("textures/dust.png")
    
	rl.ImageResize(groundImage, WINDOW_WIDTH, WINDOW_HEIGHT)
    rl.ImageResize(animalsImage, i32(ANIM_SIZE * NUM_COL), i32(ANIM_SIZE * NUM_ROW))

    *groundTexture = rl.LoadTextureFromImage(groundImage)
    *animalsTexture = rl.LoadTextureFromImage(animalsImage)
    *dustTexture = rl.LoadTextureFromImage(dustImage)
    
	rl.UnloadImage(groundImage)
    rl.UnloadImage(animalsImage)
    rl.UnloadImage(dustImage)
}

func findLastRowEmpties(board *[BOARD_SIZE]*Animal, empties *[NUM_COL]bool) int {
	var emptyColCount int
	for i := 0; i < NUM_COL; i++ {
		if board[i] == nil {
			empties[i] = true
			emptyColCount++
		}
	}
	return emptyColCount
}

func moveResquedToBoard(board, resqued *[BOARD_SIZE]*Animal, boardIndex, resquedIndex int,
                        numAnimalLeft *int, resquedChanged *bool) {
    i := boardIndex
	animToPush := resqued[resquedIndex]
	resqued[resquedIndex] = nil

    for i >= 0 {
		nextAnimToPush := board[i]
		board[i] = animToPush
		if nextAnimToPush != nil {
			jumpAnimal(nextAnimToPush, 
			           Vec2{nextAnimToPush.pos.X, nextAnimToPush.pos.Y - ROW_HEIGHT}, 
					   20, 12) 
			animToPush = nextAnimToPush
			i -= NUM_COL
		} else { 
			break 
		}
	}

	*numAnimalLeft += 1
	*resquedChanged = true
}

func scatterResqued(board, resqued *[BOARD_SIZE]*Animal, maxIndexToScatter int, 
                    frontRowPos *[NUM_COL]Vec2, numAnimalLeft *int, resquedChanged *bool) {
	lastRowEmptyIndices := [NUM_COL]bool{}
	emptyCount := findLastRowEmpties(board, &lastRowEmptyIndices)
    indexToMoveToBoard := maxIndexToScatter

	for i := 0; i < NUM_COL; i++ {
		if lastRowEmptyIndices[i] {
			jumpAnimal(resqued[indexToMoveToBoard], frontRowPos[i], 24, 16)
			frontRowIndexToJump := BOARD_SIZE - NUM_COL + i
			moveResquedToBoard(board, resqued, frontRowIndexToJump, indexToMoveToBoard,
		                       numAnimalLeft, resquedChanged)
			indexToMoveToBoard--
			emptyCount--
			if indexToMoveToBoard < 0 || emptyCount < 1  { 
				break
			}
		}
	}

	resqued[indexToMoveToBoard + 1] = resqued[maxIndexToScatter + 1]
	resqued[maxIndexToScatter + 1] = nil
}

func updateAnimState(animals *[BOARD_SIZE]Animal, board, resqued *[BOARD_SIZE]*Animal, 
                     frontRowPos *[NUM_COL]Vec2, numAnimalLeft *int, 
					 resquedChanged *bool) bool {
	isAllUpdated := true

	for i := range animals {
		anim := &animals[i]
		// Update Press and Height 
		if anim.press > 0 {
			anim.height -= anim.press 	
			if anim.height < MIN_ANIM_HEIGHT {
				anim.height = MIN_ANIM_HEIGHT
			}
			anim.press /= 2
			if anim.press < 0.0001 {
				anim.press = -1
			}
		}
		if anim.press < 0 {
			if anim.height - anim.press >= ANIM_SIZE {
				anim.height = ANIM_SIZE
				anim.press = 0
			} else {
				anim.height -= anim.press
				anim.press *= 2
			}
		}

		// Update Pos and Veloc if it's moving or has accel
		if !(anim.veloc == Vec2{}) || !(anim.accel == Vec2{}) {
			isAllUpdated = false
			pos := anim.pos
			veloc := anim.veloc
			anim.pos = Vec2Add(pos, veloc)
			anim.veloc = Vec2Add(anim.veloc, anim.accel)
			if anim.totalJumpFrames > 0 { anim.currJumpFrame++ }

			// Take care of landing
			if Vec2DistSq(anim.pos, anim.dest) < Vec2LenSq(anim.veloc) / 2 {
				if anim.veloc.Y > FPS/2 { anim.press = anim.veloc.Y / 3 }
				if anim.veloc.Y > FPS { anim.dustDuration = MAX_DUST_DURATION }

				// if the landing animal is the last resqued(the one crossing the bridge)
				lastResquedIndex := BOARD_SIZE - 1 - *numAnimalLeft
				if lastResquedIndex > 0 && anim == resqued[lastResquedIndex] {
					// if the landing veloc is great, send previously resqued animals
					// back to the main land above the bridge
				    if anim.veloc.Y > FPS { 
						scatterResqued(board, resqued, lastResquedIndex - 1, frontRowPos,
									   numAnimalLeft, resquedChanged)
				    } else {
					// if veloc is little, compress and move the previously 
					// resqued animal sideway
						prevAnimIndex := lastResquedIndex - 1
						prevAnim := resqued[prevAnimIndex]
						prevAnim.press = ANIM_SIZE

						pushFactor := f32(prevAnimIndex/2 + 1)
						if prevAnimIndex % 2 == 0 {
							prevAnim.dest = Vec2Sub(prevAnim.pos, 
													Vec2{pushFactor * ANIM_SIZE * 0.25, 0})
							prevAnim.accel = Vec2{pushFactor * ANIM_SIZE*0.075, 0}
							prevAnim.veloc = Vec2{pushFactor * -ANIM_SIZE*0.15, 0}
						} else {
							prevAnim.dest = Vec2Add(prevAnim.pos, 
													Vec2{pushFactor * ANIM_SIZE * 0.25, 0})
							prevAnim.accel = Vec2{pushFactor * -ANIM_SIZE*0.075, 0}
							prevAnim.veloc = Vec2{pushFactor * ANIM_SIZE*0.15, 0}
						}
					}
				}

				anim.pos = anim.dest
				anim.veloc, anim.accel = Vec2{}, Vec2{}
				anim.scale = 1
				anim.scaleDecRate = 0
				anim.totalJumpFrames = 0
				anim.ascFrames = 0
				anim.currJumpFrame = 0
			}
		}
	}

	return isAllUpdated
}

func resetState(animals *[BOARD_SIZE]Animal, board, resqued *[BOARD_SIZE]*Animal,
			    frontRowPos *[NUM_COL]Vec2) {
	
	*animals = [BOARD_SIZE]Animal{}
	setAnimals(animals)

	*board = [BOARD_SIZE]*Animal{}
	for i := 0; i < BOARD_SIZE; i++ {
		board[i] = &animals[i]
	}

	shuffleBoard(board, frontRowPos)
	printbd(board)

	*resqued = [BOARD_SIZE]*Animal{}
	printbd(resqued)
}

func processKeyDown(anim *Animal) {
	anim.press = anim.height/20
	if anim.height < MIN_JUMP_HEIGHT { anim.height = MIN_JUMP_HEIGHT } 
}

func main() {

	animals := [BOARD_SIZE]Animal{}
	board := [BOARD_SIZE]*Animal{}
	resqued := [BOARD_SIZE]*Animal{}
	frontRowPos := [NUM_COL]Vec2{}
	resetState(&animals, &board, &resqued, &frontRowPos)

    numAnimalLeft := BOARD_SIZE
    resquableIndex := [NUM_COL]int{}
	isAllAnimUpdated := true
	isQuitting := false

	resquedChanged := true
	mostRecentResqueType := u8(0xFF)  // initially, all front row animals can be resqued.
    numPossibleMoves := findResquables(&board, mostRecentResqueType, &resquableIndex)
    fmt.Printf("numPossibleMoves: %d, %v\n", numPossibleMoves, resquableIndex)

    rl.InitWindow(WINDOW_WIDTH, WINDOW_HEIGHT, "Animal Logic")
    rl.SetTargetFPS(FPS)

	var groundTexture, animalsTexture, dustTexture rl.Texture2D
	loadTextures(&groundTexture, &animalsTexture, &dustTexture)
    
    for !isQuitting && !rl.WindowShouldClose() {

		if isAllAnimUpdated {
			if numAnimalLeft < BOARD_SIZE && resquedChanged { 
			    assert(resqued[BOARD_SIZE - numAnimalLeft - 1] != nil, "resqued array has nil")
				mostRecentResqueType := resqued[BOARD_SIZE - numAnimalLeft - 1].animType
				for i := range resquableIndex { resquableIndex[i] = 0 }
				numNextMoves := findResquables(&board, mostRecentResqueType, &resquableIndex)
				resquedChanged = false
				fmt.Printf("numNextMoves: %d, %v\n", numNextMoves, resquableIndex)
				fmt.Printf("numAnimalLeft: %d\n", numAnimalLeft)
				printbd(&board)
				if numNextMoves == 0 {
					fmt.Println("NO MORE MOVE LEFT!!")
				}
			}

			if rl.IsKeyDown(KEY_A) || (rl.IsMouseButtonDown(MOUSE_LEFT) && 
			   isAnimRectClicked(board[FRONT_ROW_BASEINDEX])) {
				fmt.Println("A pressed!")
				if resquableIndex[0] != 0 { processKeyDown(board[FRONT_ROW_BASEINDEX]) }
			} else if rl.IsKeyDown(KEY_S) || (rl.IsMouseButtonDown(MOUSE_LEFT) && 
					  isAnimRectClicked(board[FRONT_ROW_BASEINDEX + 1])) {
				fmt.Println("S pressed!")
				if resquableIndex[1] != 0 { processKeyDown(board[FRONT_ROW_BASEINDEX+1]) }
			} else if rl.IsKeyDown(KEY_D) || (rl.IsMouseButtonDown(MOUSE_LEFT) && 
					  isAnimRectClicked(board[FRONT_ROW_BASEINDEX + 2])) {
				fmt.Println("D pressed!")
				if resquableIndex[2] != 0 { processKeyDown(board[FRONT_ROW_BASEINDEX+2]) }
			} else if rl.IsKeyDown(KEY_F) || (rl.IsMouseButtonDown(MOUSE_LEFT) && 
					  isAnimRectClicked(board[FRONT_ROW_BASEINDEX + 3])) {
				fmt.Println("F pressed!")
				if resquableIndex[3] != 0 { processKeyDown(board[FRONT_ROW_BASEINDEX+3]) }
			} else if rl.IsKeyReleased(KEY_A) || (rl.IsMouseButtonReleased(MOUSE_LEFT) && 
					  isAnimRectClicked(board[FRONT_ROW_BASEINDEX])) {
				fmt.Println("A released!")
				if resquableIndex[0] != 0 {
					resqueAt(&board, &resqued, FRONT_ROW_BASEINDEX, numAnimalLeft)
					numAnimalLeft--
					resquedChanged = true
					//time.Sleep(time.Second * 1)
				}
			} else if rl.IsKeyReleased(KEY_S) || (rl.IsMouseButtonReleased(MOUSE_LEFT) && 
					  isAnimRectClicked(board[FRONT_ROW_BASEINDEX + 1])) {
				fmt.Println("S released!")
				if resquableIndex[1] != 0 {
					resqueAt(&board, &resqued, FRONT_ROW_BASEINDEX + 1, numAnimalLeft)
					numAnimalLeft--
					resquedChanged = true
				}
			} else if rl.IsKeyReleased(KEY_D) || (rl.IsMouseButtonReleased(MOUSE_LEFT) && 
					  isAnimRectClicked(board[FRONT_ROW_BASEINDEX + 2])) {
				fmt.Println("D released!")
				if resquableIndex[2] != 0 {
					resqueAt(&board, &resqued, FRONT_ROW_BASEINDEX + 2, numAnimalLeft)
					numAnimalLeft--
					resquedChanged = true
				}
			} else if rl.IsKeyReleased(KEY_F) || (rl.IsMouseButtonReleased(MOUSE_LEFT) && 
					  isAnimRectClicked(board[FRONT_ROW_BASEINDEX + 3])) {
				fmt.Println("F released!")
				if resquableIndex[3] != 0 {
					resqueAt(&board, &resqued, FRONT_ROW_BASEINDEX + 3, numAnimalLeft)
					numAnimalLeft--
					resquedChanged = true
				}
			} else if resqued[BOARD_SIZE - 1] != nil && (rl.IsKeyReleased(KEY_G) || 
				(rl.IsMouseButtonReleased(MOUSE_LEFT) && 
				 isAnimRectClicked(resqued[BOARD_SIZE - 1]))) {
				fmt.Println("G released!! Play Again!")
				resetState(&animals, &board, &resqued, &frontRowPos)
				numAnimalLeft = BOARD_SIZE
				resquableIndex = [NUM_COL]int{}
				resquedChanged = true
				mostRecentResqueType = u8(0xFF)  
				numPossibleMoves = findResquables(&board, mostRecentResqueType, &resquableIndex)
			} else if resqued[BOARD_SIZE - 1] != nil &&
				(rl.IsKeyReleased(KEY_Q) || rl.IsMouseButtonReleased(MOUSE_RIGHT)) { 
				fmt.Println("Quiting Game! Bye!")
				isQuitting = true
			} else if rl.IsKeyDown(KEY_Q) {
				fmt.Println("Q released!! Resetting the board!")
				resetState(&animals, &board, &resqued, &frontRowPos)
				numAnimalLeft = BOARD_SIZE
				resquableIndex = [NUM_COL]int{}
				resquedChanged = true
				mostRecentResqueType = u8(0xFF)  
				numPossibleMoves = findResquables(&board, mostRecentResqueType, &resquableIndex)
			}
		}

		isAllAnimUpdated = updateAnimState(&animals, &board, &resqued, &frontRowPos, 
		                                   &numAnimalLeft, &resquedChanged)

        rl.BeginDrawing()
        {

            rl.DrawTextureEx(groundTexture, rl.Vector2{0, 0}, 0, 1, rl.RayWhite)

			for i := 0; i < BOARD_SIZE; i++ {
                if board[i] != nil {
                    drawAnimal(&animalsTexture, &dustTexture, board[i])
                }
            }
			
			for i := 0; i < BOARD_SIZE - numAnimalLeft; i++ {
                if resqued[i] != nil {
                    drawAnimal(&animalsTexture, &dustTexture, resqued[i])
                }
            }
        }
        rl.EndDrawing()
    
		if numAnimalLeft == 0 {
			fmt.Println("ALL RESQUED!!")
			fmt.Println("Press G or Click the last animal resqued to play again!")
			fmt.Println("Press Q or Right Click to quit!")
		}
    }
}
