//go:build !android && !ios

package mobile

// Dummy — заглушка для сборки на desktop (ebitenmobile bind компилирует только android/ios).
func Dummy() {}
