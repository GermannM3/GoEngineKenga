package gameplay

// CarID идентификатор машины
type CarID string

const (
	CarAtom       CarID = "atom"
	CarMoskvichM70 CarID = "moskvich_m70"
	CarMoskvichM90 CarID = "moskvich_m90"
)

// CarConfig параметры машины для аркадного картинга (Atom, M70, M90)
type CarConfig struct {
	ID            CarID
	DisplayName   string
	MaxSpeed      float32 // единиц/сек
	Acceleration  float32
	Handling      float32 // скорость поворота
	Mass          float32 // для столкновений
	SpecialAbility string  // "electric_boost" | ""
}

// Cars конфиги всех машин
var Cars = map[CarID]CarConfig{
	CarAtom: {
		ID:              CarAtom,
		DisplayName:     "Atom",
		MaxSpeed:        24,
		Acceleration:    18,
		Handling:        2.5,
		Mass:            1.0,
		SpecialAbility:  "electric_boost",
	},
	CarMoskvichM70: {
		ID:            CarMoskvichM70,
		DisplayName:   "Moskvich M70",
		MaxSpeed:      26,
		Acceleration:  15,
		Handling:      2.0,
		Mass:          1.5,
		SpecialAbility: "",
	},
	CarMoskvichM90: {
		ID:            CarMoskvichM90,
		DisplayName:   "Moskvich M90",
		MaxSpeed:      28,
		Acceleration:  12,
		Handling:      1.5,
		Mass:          2.2,
		SpecialAbility: "",
	},
}

// CarConfigByID возвращает конфиг по строковому id
func CarConfigByID(id string) (CarConfig, bool) {
	c, ok := Cars[CarID(id)]
	return c, ok
}
