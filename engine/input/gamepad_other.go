//go:build !windows

package input

// На не-Windows платформах нативный опрос геймпада недоступен
// (ebiten-бэкенд использует свой кроссплатформенный API).

// PollNativeGamepad — заглушка: геймпад не подключён.
func (s *State) PollNativeGamepad() {
	s.SetPadConnected(false)
}
