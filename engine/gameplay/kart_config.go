package gameplay

// CarID идентификатор машины
type CarID string

const (
	CarAtom        CarID = "atom"
	CarMoskvichM70 CarID = "moskvich_m70"
	CarMoskvichM90 CarID = "moskvich_m90"
)

// CarConfig параметры машины для аркадного картинга (Atom, M70, M90)
type CarConfig struct {
	ID             CarID
	DisplayName    string
	MaxSpeed       float32 // единиц/сек
	Acceleration   float32
	Handling       float32 // скорость поворота
	Mass           float32 // для столкновений
	SpecialAbility string  // "electric_boost" | ""
}

// Cars конфиги всех машин. Скорости в px/с для 2D-трассы 1280×720:
// круг овала ≈ 2150px, таргет ~13-15 с на круг.
var Cars = map[CarID]CarConfig{
	CarAtom: {
		ID:             CarAtom,
		DisplayName:    "Atom",
		MaxSpeed:       200,
		Acceleration:   160,
		Handling:       3.0,
		Mass:           1.0,
		SpecialAbility: "electric_boost",
	},
	CarMoskvichM70: {
		ID:             CarMoskvichM70,
		DisplayName:    "Moskvich M70",
		MaxSpeed:       210,
		Acceleration:   150,
		Handling:       2.4,
		Mass:           1.5,
		SpecialAbility: "",
	},
	CarMoskvichM90: {
		ID:             CarMoskvichM90,
		DisplayName:    "Moskvich M90",
		MaxSpeed:       220,
		Acceleration:   140,
		Handling:       1.8,
		Mass:           2.2,
		SpecialAbility: "",
	},
}

// CarConfigByID возвращает конфиг по строковому id
func CarConfigByID(id string) (CarConfig, bool) {
	c, ok := Cars[CarID(id)]
	return c, ok
}
