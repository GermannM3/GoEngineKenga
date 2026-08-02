//go:build wasm

package main

import "unsafe"

//go:wasmimport env debugLog
func debugLog(ptr uint32, l uint32)

//go:wasmimport env getInputKey
func getInputKey(key int32) int32

//go:wasmimport env getDeltaTime
func getDeltaTime() float32

//go:wasmimport env getEntityTransform
func getEntityTransform(entityID uint32) uintptr

//go:wasmimport env setEntityTransform
func setEntityTransform(entityID uint32, posX, posY, posZ, rotX, rotY, rotZ, scaleX, scaleY, scaleZ float32)

//go:wasmimport env getEntityRigidbody
func getEntityRigidbody(entityID uint32) uintptr

//go:wasmimport env setEntityRigidbodyVelocity
func setEntityRigidbodyVelocity(entityID uint32, velX, velY, velZ float32)

//go:wasmimport env getEntityCollider
func getEntityCollider(entityID uint32) uintptr

const (
	KeyA = 0
	KeyD = 3
	KeyW = 22
	KeyS = 18
	KeySpace = 44
	KeyJ = 10
)

// Player entity ID (should match the ID assigned in the scene)
const playerEntityID uint32 = 2  // Player is the 2nd entity created

// Player movement variables
var playerVelocityX float32 = 0
var playerVelocityY float32 = 0
var isOnGround bool = false
var gravity float32 = -500.0
var jumpForce float32 = 300.0
var moveSpeed float32 = 250.0

//export Update
func Update(dtMillis int32) {
	dt := float32(dtMillis) / 1000.0  // Convert milliseconds to seconds

	// Player movement controls
	if getInputKey(KeyA) != 0 {
		playerVelocityX = -moveSpeed
	} else if getInputKey(KeyD) != 0 {
		playerVelocityX = moveSpeed
	} else {
		// Slow down when no input (friction)
		if playerVelocityX > 0 {
			playerVelocityX -= moveSpeed * 0.5 // Increased friction
			if playerVelocityX < 0 {
				playerVelocityX = 0
			}
		} else if playerVelocityX < 0 {
			playerVelocityX += moveSpeed * 0.5 // Increased friction
			if playerVelocityX > 0 {
				playerVelocityX = 0
			}
		}
	}

	// Jumping - checking if player is on ground would require collision detection
	// For now, we'll assume player can jump if velocityY is close to 0 or negative (falling)
	if getInputKey(KeySpace) != 0 && (playerVelocityY <= 0) {
		playerVelocityY = jumpForce
	}

	// Apply gravity
	playerVelocityY += gravity * dt

	// Update player velocity in the physics system
	setEntityRigidbodyVelocity(playerEntityID, playerVelocityX, playerVelocityY, 0)

	// Log the update for debugging
	msg := []byte("Game Update - Player Vel: X=")
	debugLog(uint32(uintptr(unsafe.Pointer(&msg[0]))), uint32(len(msg)))
}

func main() {}