package gameplay

import (
	"goenginekenga/engine/ecs"
	"goenginekenga/engine/input"
)

// GameLogicSystem handles the core gameplay mechanics for 2D platformer
type GameLogicSystem struct {
	playerID     ecs.EntityID
	groundIDs    []ecs.EntityID
	enemyIDs     []ecs.EntityID
	itemIDs      []ecs.EntityID
	npcIDs       []ecs.EntityID
	lastJumpPress bool
	onGround     bool
	moveSpeed    float32
	jumpForce    float32
}

// NewGameLogicSystem creates a new instance of the game logic system
func NewGameLogicSystem() *GameLogicSystem {
	return &GameLogicSystem{
		moveSpeed: 250.0,
		jumpForce: 300.0,
	}
}

// Update processes the game logic for each frame
func (gls *GameLogicSystem) Update(world *ecs.World, inputState *input.State) {
	// Find all relevant entities if not already cached
	gls.findEntities(world)

	// Process player input and movement
	gls.processPlayerInput(world, inputState)

	// Update enemy AI
	gls.updateEnemies(world)

	// Check item pickups
	gls.checkItemPickups(world)

	// Update other game logic
	gls.updateGameStatus(world)
}

// findEntities locates all relevant entities in the world
func (gls *GameLogicSystem) findEntities(world *ecs.World) {
	gls.playerID = 0
	gls.groundIDs = []ecs.EntityID{}
	gls.enemyIDs = []ecs.EntityID{}
	gls.itemIDs = []ecs.EntityID{}
	gls.npcIDs = []ecs.EntityID{}

	for _, id := range world.Entities() {
		name := world.Name(id)
		if name == "Player" {
			gls.playerID = id
		} else if name == "Ground" || containsSubstring(name, "Platform") {
			gls.groundIDs = append(gls.groundIDs, id)
		} else if containsSubstring(name, "Enemy") {
			gls.enemyIDs = append(gls.enemyIDs, id)
		} else if containsSubstring(name, "Item") {
			gls.itemIDs = append(gls.itemIDs, id)
		} else if containsSubstring(name, "NPC") {
			gls.npcIDs = append(gls.npcIDs, id)
		}
	}
}

// processPlayerInput handles player movement and jumping
func (gls *GameLogicSystem) processPlayerInput(world *ecs.World, inputState *input.State) {
	if gls.playerID == 0 {
		return
	}

	// Get player's rigidbody component
	rb, hasRb := world.GetRigidbody(gls.playerID)
	if !hasRb {
		return
	}

	// Get player's animation controller
	animCtrl, hasAnimCtrl := world.GetAnimationController(gls.playerID)

	// Handle horizontal movement
	moveX := float32(0)
	isMoving := false
	if inputState.IsKeyPressed(input.KeyA) || inputState.IsKeyPressed(input.KeyArrowLeft) {
		moveX = -gls.moveSpeed
		isMoving = true
	} else if inputState.IsKeyPressed(input.KeyD) || inputState.IsKeyPressed(input.KeyArrowRight) {
		moveX = gls.moveSpeed
		isMoving = true
	}

	// Apply horizontal movement
	rb.Velocity.X = moveX

	// Handle jumping
	isJumpPressed := inputState.IsKeyPressed(input.KeySpace) || inputState.IsKeyPressed(input.KeyW) || inputState.IsKeyPressed(input.KeyArrowUp)

	// Check if player is on ground by checking collision with ground entities
	gls.onGround = gls.checkPlayerOnGround(world)

	if isJumpPressed && !gls.lastJumpPress && gls.onGround {
		rb.Velocity.Y = gls.jumpForce
		gls.onGround = false

		// Switch to jump animation
		if hasAnimCtrl {
			animCtrl.CurrentClip = "jump"
			world.SetAnimationController(gls.playerID, animCtrl)

			// Also update animation state to play the animation
			animState, hasAnimState := world.GetAnimationState(gls.playerID)
			if !hasAnimState {
				animState = ecs.AnimationState{
					CurrentClip: "jump",
					CurrentFrame: 0,
					ElapsedTime: 0,
					IsPlaying: true,
					Loop: false,
					Speed: 1.0,
				}
			} else {
				animState.CurrentClip = "jump"
				animState.ElapsedTime = 0
				animState.IsPlaying = true
			}
			world.SetAnimationState(gls.playerID, animState)
		}
	}

	gls.lastJumpPress = isJumpPressed

	// Update animation based on player state
	if hasAnimCtrl {
		// Determine which animation to play
		targetAnimation := "idle"
		if isMoving && gls.onGround {
			targetAnimation = "walk"
		} else if !gls.onGround {
			// Could be jumping or falling
			if rb.Velocity.Y > 0 { // Falling
				targetAnimation = "idle" // or create a fall animation
			} else if animCtrl.CurrentClip != "jump" { // Not currently playing jump animation
				targetAnimation = "idle"
			}
		}

		// Only change animation if it's different from current one
		if animCtrl.CurrentClip != targetAnimation {
			animCtrl.CurrentClip = targetAnimation
			world.SetAnimationController(gls.playerID, animCtrl)

			// Update animation state to play the new animation
			animState, hasAnimState := world.GetAnimationState(gls.playerID)
			if !hasAnimState {
				animState = ecs.AnimationState{
					CurrentClip: targetAnimation,
					CurrentFrame: 0,
					ElapsedTime: 0,
					IsPlaying: true,
					Loop: targetAnimation != "jump", // Jump shouldn't loop
					Speed: 1.0,
				}
			} else {
				animState.CurrentClip = targetAnimation
				animState.CurrentFrame = 0  // Reset frame when changing animation
				animState.ElapsedTime = 0   // Reset timing when changing animation
				animState.IsPlaying = true
				animState.Loop = targetAnimation != "jump" // Jump shouldn't loop
			}
			world.SetAnimationState(gls.playerID, animState)
		}
	}

	// Update the rigidbody in the world
	world.SetRigidbody(gls.playerID, rb)
}

// checkPlayerOnGround determines if the player is standing on ground
func (gls *GameLogicSystem) checkPlayerOnGround(world *ecs.World) bool {
	if gls.playerID == 0 {
		return false
	}

	playerTr, hasPlayerTr := world.GetTransform(gls.playerID)
	playerCol, hasPlayerCol := world.GetCollider(gls.playerID)
	
	if !hasPlayerTr || !hasPlayerCol {
		return false
	}

	// Simple ground check: if player's Y velocity is close to 0 and there's ground below
	playerBottomY := playerTr.Position.Y - playerCol.Size.Y/2
	for _, groundID := range gls.groundIDs {
		groundTr, hasGroundTr := world.GetTransform(groundID)
		groundCol, hasGroundCol := world.GetCollider(groundID)
		
		if hasGroundTr && hasGroundCol {
			groundTopY := groundTr.Position.Y + groundCol.Size.Y/2
			
			// Check if player is close to the ground surface
			if playerBottomY >= groundTopY && playerBottomY <= groundTopY+5 {
				return true
			}
		}
	}
	
	return false
}

// updateEnemies handles enemy AI
func (gls *GameLogicSystem) updateEnemies(world *ecs.World) {
	for _, enemyID := range gls.enemyIDs {
		// Basic AI: patrol between two points or follow player if close
		// For now, just ensure they have physics
		rb, hasRb := world.GetRigidbody(enemyID)
		if hasRb {
			// Apply minimal movement or let physics handle it
			world.SetRigidbody(enemyID, rb)
		}
	}
}

// checkItemPickups handles collecting items
func (gls *GameLogicSystem) checkItemPickups(world *ecs.World) {
	playerTr, hasPlayerTr := world.GetTransform(gls.playerID)
	if !hasPlayerTr {
		return
	}
	
	for _, itemID := range gls.itemIDs {
		itemTr, hasItemTr := world.GetTransform(itemID)
		itemCol, hasItemCol := world.GetCollider(itemID)
		
		if hasItemTr && hasItemCol && itemCol.IsTrigger {
			// Simple distance check for pickup
			dx := playerTr.Position.X - itemTr.Position.X
			dy := playerTr.Position.Y - itemTr.Position.Y
			distanceSquared := dx*dx + dy*dy
			pickupDistance := float32(50.0) // Adjust as needed
			
			if distanceSquared < pickupDistance*pickupDistance {
				// Mark item for removal or trigger pickup event
				// For now, just hide the item
				sprite, hasSprite := world.GetSpriteRenderer(itemID)
				if hasSprite {
					sprite.Visible = false
					world.SetSpriteRenderer(itemID, sprite)
				}
			}
		}
	}
}

// updateGameStatus updates any game state
func (gls *GameLogicSystem) updateGameStatus(world *ecs.World) {
	// Update any game state variables here
	// For example, check win/lose conditions
}

// Helper function to check if a string contains a substring
func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if str[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}