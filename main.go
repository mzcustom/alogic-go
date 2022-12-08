package main

import (
	"fmt"
	"github.com/gen2brain/raylib-go/raylib"
	"crypto/rand"
	"math"
	"math/big"
	//"time"
)

type i64 = int64
type i32 = int32 
type f64 = float64
type f32 = float32
type u8 = uint8

type Vec2 = rl.Vector2

func Vec2Add(v1, v2 Vec2) Vec2 { return Vec2{v1.X + v2.X, v1.Y + v2.Y} }

func Vec2Sub(v1, v2 Vec2) Vec2 { return Vec2{v1.X - v2.X, v1.Y - v2.Y} }

func Vec2Div(v Vec2, f f32) Vec2 { return Vec2{v.X / f, v.Y / f} }

func Vec2Neg(v Vec2) Vec2 { return Vec2{-v.X, -v.Y} }

func Vec2LenSq(v Vec2) f32 { return v.X * v.X + v.Y * v.Y }

func Vec2Len(v Vec2) f64 { return math.Sqrt(f64(Vec2LenSq(v))) }

func Vec2DistSq(v1, v2 Vec2) f32 { 
	//distSqVec2 := Vec2Sub(v1, v2)
	return Vec2LenSq(Vec2Sub(v1, v2)) 
}

func Vec2Dist(v1, v2 Vec2) f64 {
	return math.Sqrt(f64(Vec2DistSq(v1, v2)))
}

func assert(b bool, msg string) {
	if !b {
		panic("Assert failed: " + msg + "!\n")
	}
}

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
	RESQUE_SPOT_X     = MARGIN_WIDTH + (WINDOW_WIDTH - 2 * MARGIN_WIDTH) / 2 
	RESQUE_SPOT_Y     = (UPPER_LAND_HEIGHT + 7 * MARGIN_HEIGHT) + 
						 (WINDOW_HEIGHT - (UPPER_LAND_HEIGHT + 7 * MARGIN_HEIGHT)) / 2 
	BOARD_SIZE        = NUM_ROW * NUM_COL
	NUM_COLOR         = 4
	NUM_KIND          = 4

	FPS = 60
	FRAME_TIME = f32(1) / f32(FPS)
	// Raylib input int32 map
	KEY_A = 65
	KEY_S = 83
	KEY_D = 68
	KEY_F = 70
	MOUSE_LEFT = 0
	//MOUSE_RIGHT = 1
	KEY_RIGHT = 262
	KEY_LEFT = 263
	KEY_DOWN = 264
	KEY_UP = 265
)

// accel and veloc in pixel/frame
type Animal struct {
	pos Vec2
	dest Vec2
	accel Vec2
	veloc Vec2
	animType u8
}

func setAnimals(animals *[BOARD_SIZE]Animal) {
	color, kind := u8(1), u8(1)

	for row := 0; row < NUM_ROW; row++ {
		colorBit := color << NUM_KIND
		for col := 0; col < NUM_COL; col++ {
			boardIndex := row * NUM_COL + col
			animals[boardIndex].animType = colorBit | kind
			kind <<= 1
		}
		kind = 1
		color <<= 1
	}
}

func shuffleBoard(board *[BOARD_SIZE]*Animal) {
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

func resqueAt(board *[BOARD_SIZE]*Animal, resqued *[BOARD_SIZE]*Animal, 
              resqueIndex int, numAnimalLeft int) {
	i := resqueIndex			  
    anim := board[i]
    assert(anim.animType != 0, "animal to be resque has type 0")

	jumpAnimal(anim, Vec2{RESQUE_SPOT_X, RESQUE_SPOT_Y}, 20, 4)
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
func jumpAnimal(anim *Animal, dest Vec2, totalFrames f32, ascFrames f32) {
	// desFrames: frames left at the its original position while desending to dest
	// it takes exactly same frames going up and falling down to its original pos.
	desFrames := totalFrames - 2 * ascFrames
	diff := Vec2Sub(dest, anim.pos)

	// v * t + 1/2 * a * t*t = diff.Y => Equation of motion
	// t is in frame because veloc and accel is in pixel/frame
	// a = -v / ascFrames => v is 0 when it reaches its top pos after ascFrames
	// v * desFrames + 0.5 * (v / ascFrames) * desFrames * desFrames = diff.Y
	// v(desFrames + 0.5 * desFrame * desFrame / ascFrames) = diff.Y
	// v = diff.Y / (desFrames + 0.5 * desFrame * desFrame / ascFrames) 

	anim.veloc = Vec2{diff.X / totalFrames, 
					 -diff.Y / (desFrames + 0.5 * desFrames * desFrames / ascFrames)}
	anim.accel = Vec2{0, -anim.veloc.Y / ascFrames}
	anim.dest = dest
}


func drawAnimal(animalsTexture *rl.Texture2D, animal *Animal) {
	colorBitfield := animal.animType >> NUM_KIND
	kindBitfield := animal.animType & 0b1111
	colorOffset := NUM_COLOR - 1 - findFirst1Bit(colorBitfield)
	kindOffset := NUM_KIND - 1 - findFirst1Bit(kindBitfield)
    rect := rl.Rectangle{f32(kindOffset) * ANIM_SIZE, f32(colorOffset) * ANIM_SIZE,
                         ANIM_SIZE, ANIM_SIZE}
    animSizeOffset := Vec2{ANIM_SIZE / 2, ANIM_SIZE / 2}
    rl.DrawTextureRec(*animalsTexture, rect, Vec2Sub(animal.pos, animSizeOffset), 
					  rl.RayWhite)
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
	if animal == nil {
		return false
	}

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

func loadTextures(groundTexture, animalsTexture *rl.Texture2D) {
    groundImage := rl.LoadImage("textures/background.png")
    animalsImage := rl.LoadImage("textures/animals.png")
    rl.ImageResize(groundImage, WINDOW_WIDTH, WINDOW_HEIGHT)
    rl.ImageResize(animalsImage, i32(ANIM_SIZE * NUM_COL), i32(ANIM_SIZE * NUM_ROW))

    *groundTexture = rl.LoadTextureFromImage(groundImage)
    *animalsTexture = rl.LoadTextureFromImage(animalsImage)
    rl.UnloadImage(groundImage)
    rl.UnloadImage(animalsImage)
}

func updatePos(animals *[BOARD_SIZE]Animal) bool {
	isAllUpdated := true
	for i := range animals {
		anim := &animals[i]
		if !(anim.veloc == Vec2{}) || !(anim.accel == Vec2{}) {
			isAllUpdated = false
			pos := anim.pos
			veloc := anim.veloc
			anim.pos = Vec2Add(pos, veloc)
			anim.veloc = Vec2Add(anim.veloc, anim.accel) 
			
			if Vec2DistSq(anim.pos, anim.dest) < Vec2LenSq(anim.veloc) / 2 {
				anim.pos = anim.dest
				anim.veloc = Vec2{}
				anim.accel = Vec2{}

				fmt.Printf("Touchdown anim[%d]\n", i) 
			}
		}
	}
	return isAllUpdated
}

func main() {
	fmt.Println("Let's implement Animal logic in GO")
	
	animals := [BOARD_SIZE]Animal{}

	setAnimals(&animals)
	fmt.Printf("Animals: %v\n", animals)

	board := [BOARD_SIZE]*Animal{}
	for i := 0; i < BOARD_SIZE; i++ {
		board[i] = &animals[i]
	}

	shuffleBoard(&board)
	printbd(&board)

	resqued := [BOARD_SIZE]*Animal{}
	printbd(&resqued)

    numAnimalLeft := BOARD_SIZE
    resquableIndex := [NUM_COL]int{}
	isAllPosUpdated := true

    prevLeft := numAnimalLeft
    mostRecentResqueType := u8(0xFF)  // initially, all front row animals can be resqued.
    numPossibleMoves := findResquables(&board, mostRecentResqueType, &resquableIndex)
    fmt.Printf("numPossibleMoves: %d, %v\n", numPossibleMoves, resquableIndex)

    rl.InitWindow(WINDOW_WIDTH, WINDOW_HEIGHT, "Animal Logic")
    rl.SetTargetFPS(FPS)

	var groundTexture, animalsTexture rl.Texture2D
	loadTextures(&groundTexture, &animalsTexture)
    
    for numAnimalLeft > 0 && !rl.WindowShouldClose() {

		if isAllPosUpdated {
			if prevLeft > numAnimalLeft {
				assert(resqued[BOARD_SIZE - numAnimalLeft - 1] != nil, "resqued array has nil")
				mostRecentResqueType := resqued[BOARD_SIZE - numAnimalLeft - 1].animType
				for i := range resquableIndex { resquableIndex[i] = 0 }
				numNextMoves := findResquables(&board, mostRecentResqueType, &resquableIndex)
				prevLeft = numAnimalLeft
				fmt.Printf("numNextMoves: %d, %v\n", numNextMoves, resquableIndex)
				fmt.Printf("numAnimalLeft: %d\n", numAnimalLeft)
				printbd(&board)
				if numNextMoves == 0 {
					fmt.Println("NO MORE MOVE LEFT!!")
				}
			}

			if rl.IsKeyDown(KEY_A) || 
			   (rl.IsMouseButtonDown(MOUSE_LEFT) && 
				isAnimRectClicked(board[BOARD_SIZE - NUM_COL])) {
				fmt.Println("A pressed ")
				if resquableIndex[0] != 0 {
					resqueAt(&board, &resqued, BOARD_SIZE - NUM_COL, numAnimalLeft)
					numAnimalLeft--
					//time.Sleep(time.Second * 1)
				}
			} else if rl.IsKeyDown(KEY_S) || 
				(rl.IsMouseButtonDown(MOUSE_LEFT) && 
				isAnimRectClicked(board[BOARD_SIZE - NUM_COL + 1])) {
				fmt.Println("S pressed ")
				if resquableIndex[1] != 0 {
					resqueAt(&board, &resqued, BOARD_SIZE - NUM_COL + 1, numAnimalLeft)
					numAnimalLeft--
					//time.Sleep(time.Second * 1)
				}
			} else if rl.IsKeyDown(KEY_D) || 
				(rl.IsMouseButtonDown(MOUSE_LEFT) && 
				 isAnimRectClicked(board[BOARD_SIZE - NUM_COL + 2])) {
				fmt.Println("D pressed ")
				if resquableIndex[2] != 0 {
					resqueAt(&board, &resqued, BOARD_SIZE - NUM_COL + 2, numAnimalLeft)
					numAnimalLeft--
					//time.Sleep(time.Second * 1)
				}
			} else if rl.IsKeyDown(KEY_F) ||  
				(rl.IsMouseButtonDown(MOUSE_LEFT) && 
				 isAnimRectClicked(board[BOARD_SIZE - NUM_COL + 3])) {
				fmt.Println("F pressed ")
				if resquableIndex[3] != 0 {
					resqueAt(&board, &resqued, BOARD_SIZE - NUM_COL + 3, numAnimalLeft)
					numAnimalLeft--
					//time.Sleep(time.Second * 1)
				}
			}
		}

		isAllPosUpdated = updatePos(&animals)

        rl.BeginDrawing()
        {

            /*
            for anim in animals {
              rl.DrawText(cstring(raw_data(anim.texture)), i32(anim.x), i32(anim.y), 20, rl.DARKGRAY)
            }
            */

            rl.DrawTextureEx(groundTexture, rl.Vector2{0, 0}, 0, 1, rl.RayWhite)

			for i :=0; i < BOARD_SIZE; i++ {
                if board[i] != nil {
                    drawAnimal(&animalsTexture, board[i])
                }
            }
			
			for i :=0; i < BOARD_SIZE; i++ {
                if resqued[i] != nil {
                    drawAnimal(&animalsTexture, resqued[i])
                }
            }
            
			/*
            if numAnimalLeft < BOARD_SIZE {
                mostRecentResque := resqued[BOARD_SIZE - numAnimalLeft - 1]
                assert(mostRecentResque != nil, "mostRecentResque is nil")
                drawAnimal(&animalsTexture, mostRecentResque)
            }
			*/
        }
        rl.EndDrawing()
    }

    if numAnimalLeft == 0 {
        fmt.Println("ALL RESQUED!!")
    }
}
