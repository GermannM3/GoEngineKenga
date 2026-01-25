package input

import "github.com/hajimehoshi/ebiten/v2"

// EbitenKeyToKey converts Ebiten key to our Key type
func EbitenKeyToKey(ek ebiten.Key) Key {
	switch ek {
	case ebiten.KeyA:
		return KeyA
	case ebiten.KeyB:
		return KeyB
	case ebiten.KeyC:
		return KeyC
	case ebiten.KeyD:
		return KeyD
	case ebiten.KeyE:
		return KeyE
	case ebiten.KeyF:
		return KeyF
	case ebiten.KeyG:
		return KeyG
	case ebiten.KeyH:
		return KeyH
	case ebiten.KeyI:
		return KeyI
	case ebiten.KeyJ:
		return KeyJ
	case ebiten.KeyK:
		return KeyK
	case ebiten.KeyL:
		return KeyL
	case ebiten.KeyM:
		return KeyM
	case ebiten.KeyN:
		return KeyN
	case ebiten.KeyO:
		return KeyO
	case ebiten.KeyP:
		return KeyP
	case ebiten.KeyQ:
		return KeyQ
	case ebiten.KeyR:
		return KeyR
	case ebiten.KeyS:
		return KeyS
	case ebiten.KeyT:
		return KeyT
	case ebiten.KeyU:
		return KeyU
	case ebiten.KeyV:
		return KeyV
	case ebiten.KeyW:
		return KeyW
	case ebiten.KeyX:
		return KeyX
	case ebiten.KeyY:
		return KeyY
	case ebiten.KeyZ:
		return KeyZ
	case ebiten.KeyDigit0:
		return Key0
	case ebiten.KeyDigit1:
		return Key1
	case ebiten.KeyDigit2:
		return Key2
	case ebiten.KeyDigit3:
		return Key3
	case ebiten.KeyDigit4:
		return Key4
	case ebiten.KeyDigit5:
		return Key5
	case ebiten.KeyDigit6:
		return Key6
	case ebiten.KeyDigit7:
		return Key7
	case ebiten.KeyDigit8:
		return Key8
	case ebiten.KeyDigit9:
		return Key9
	case ebiten.KeySpace:
		return KeySpace
	case ebiten.KeyEnter:
		return KeyEnter
	case ebiten.KeyEscape:
		return KeyEscape
	case ebiten.KeyTab:
		return KeyTab
	case ebiten.KeyBackspace:
		return KeyBackspace
	case ebiten.KeyDelete:
		return KeyDelete
	case ebiten.KeyInsert:
		return KeyInsert
	case ebiten.KeyHome:
		return KeyHome
	case ebiten.KeyEnd:
		return KeyEnd
	case ebiten.KeyPageUp:
		return KeyPageUp
	case ebiten.KeyPageDown:
		return KeyPageDown
	case ebiten.KeyArrowUp:
		return KeyArrowUp
	case ebiten.KeyArrowDown:
		return KeyArrowDown
	case ebiten.KeyArrowLeft:
		return KeyArrowLeft
	case ebiten.KeyArrowRight:
		return KeyArrowRight
	case ebiten.KeyShiftLeft:
		return KeyShiftLeft
	case ebiten.KeyShiftRight:
		return KeyShiftRight
	case ebiten.KeyControlLeft:
		return KeyControlLeft
	case ebiten.KeyControlRight:
		return KeyControlRight
	case ebiten.KeyAltLeft:
		return KeyAltLeft
	case ebiten.KeyAltRight:
		return KeyAltRight
	case ebiten.KeyF1:
		return KeyF1
	case ebiten.KeyF2:
		return KeyF2
	case ebiten.KeyF3:
		return KeyF3
	case ebiten.KeyF4:
		return KeyF4
	case ebiten.KeyF5:
		return KeyF5
	case ebiten.KeyF6:
		return KeyF6
	case ebiten.KeyF7:
		return KeyF7
	case ebiten.KeyF8:
		return KeyF8
	case ebiten.KeyF9:
		return KeyF9
	case ebiten.KeyF10:
		return KeyF10
	case ebiten.KeyF11:
		return KeyF11
	case ebiten.KeyF12:
		return KeyF12
	default:
		return -1
	}
}

// AllEbitenKeys returns all Ebiten keys we care about
var AllEbitenKeys = []ebiten.Key{
	ebiten.KeyA, ebiten.KeyB, ebiten.KeyC, ebiten.KeyD, ebiten.KeyE,
	ebiten.KeyF, ebiten.KeyG, ebiten.KeyH, ebiten.KeyI, ebiten.KeyJ,
	ebiten.KeyK, ebiten.KeyL, ebiten.KeyM, ebiten.KeyN, ebiten.KeyO,
	ebiten.KeyP, ebiten.KeyQ, ebiten.KeyR, ebiten.KeyS, ebiten.KeyT,
	ebiten.KeyU, ebiten.KeyV, ebiten.KeyW, ebiten.KeyX, ebiten.KeyY,
	ebiten.KeyZ,
	ebiten.KeyDigit0, ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3,
	ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6, ebiten.KeyDigit7,
	ebiten.KeyDigit8, ebiten.KeyDigit9,
	ebiten.KeySpace, ebiten.KeyEnter, ebiten.KeyEscape, ebiten.KeyTab,
	ebiten.KeyBackspace, ebiten.KeyDelete, ebiten.KeyInsert,
	ebiten.KeyHome, ebiten.KeyEnd, ebiten.KeyPageUp, ebiten.KeyPageDown,
	ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight,
	ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
	ebiten.KeyControlLeft, ebiten.KeyControlRight,
	ebiten.KeyAltLeft, ebiten.KeyAltRight,
	ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5, ebiten.KeyF6,
	ebiten.KeyF7, ebiten.KeyF8, ebiten.KeyF9, ebiten.KeyF10, ebiten.KeyF11, ebiten.KeyF12,
}
