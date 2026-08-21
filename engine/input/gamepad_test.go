package input

import "testing"

func TestPadButtonJustPressed(t *testing.T) {
	s := NewState()
	s.SetPadConnected(true)
	s.SetPadButton(PadA, true)

	if !s.IsPadButtonDown(PadA) {
		t.Fatal("кнопка A не удерживается")
	}
	if !s.IsPadButtonJustPressed(PadA) {
		t.Fatal("первое нажатие A не детектировано")
	}

	// Кадр прошёл — уже не «только что»
	s.EndFrame()
	if s.IsPadButtonJustPressed(PadA) {
		t.Fatal("JustPressed сработал второй кадр подряд")
	}
	if !s.IsPadButtonDown(PadA) {
		t.Fatal("удержание потерялось после EndFrame")
	}

	// Отпускание
	s.SetPadButton(PadA, false)
	if !s.IsPadButtonJustReleased(PadA) {
		t.Fatal("отпускание не детектировано")
	}
}

func TestPadAxisAndDeadzone(t *testing.T) {
	s := NewState()
	s.SetPadAxis(PadAxisLX, 0.9)
	if v := s.PadAxisValue(PadAxisLX); v != 0.9 {
		t.Fatalf("ось LX=%f, ожидалось 0.9", v)
	}
	if v := s.PadAxisValue(PadAxisRT); v != 0 {
		t.Fatalf("не заданная ось должна быть 0, got %f", v)
	}
	if v := Deadzone(0.1, 0.2); v != 0 {
		t.Fatalf("deadzone не обнулила малое значение: %f", v)
	}
	if v := Deadzone(0.5, 0.2); v != 0.5 {
		t.Fatalf("deadzone изменила большое значение: %f", v)
	}
}

func TestPadDisconnected(t *testing.T) {
	s := NewState()
	if s.IsPadConnected() {
		t.Fatal("по умолчанию геймпада быть не должно")
	}
	s.PollNativeGamepad() // в тестовой среде геймпада нет; не должно паниковать
}
