package funcs

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type patchUInt32Value struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func LogFatalIfError(err error, out string, exclude ...string) {
	if out != "" {
		fmt.Println(out)
	}
	if err != nil {
		if len(exclude) > 0 {
			for _, excludeError := range exclude {
				if err.Error() == excludeError {
					return
				}
			}
		}
		log.Fatal(err.Error())
	}
}

func GiBToMiB(gib float64) float64 {
	return gib * 1024 // 1 GiB = 1024 MiB
}

func MiBToGiB(mib float64) float64 {
	return mib / 1024 // 1 MiB = 1/1024 GiB
}

func GiBToKiB(gib float64) float64 {
	return gib * (1 << 20) // 1 GiB = 2^20 KiB
}

func KiBToGiB(kib float64) float64 {
	return kib / (1 << 20) // 1 KiB = 1/2^20 GiB
}

func MiBToKiB(mib float64) float64 {
	return mib * 1024 // 1 MiB = 1024 KiB
}

func KiBToMiB(kib float64) float64 {
	return kib / 1024 // 1 KiB = 1/1024 MiB
}

func parseValueUnit(input string) (float64, string, error) {
	re := regexp.MustCompile(`^([0-9.]+)([a-zA-Z]+)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(input))
	if len(matches) != 3 {
		return 0, "", errors.New("invalid input format")
	}
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, "", err
	}
	unit := strings.ToUpper(matches[2])
	return value, unit, nil
}

func ConvertUnit(fromValue, toUnit string) (string, error) {
	var convertedValue float64

	value, fromUnit, err := parseValueUnit(fromValue)
	if err != nil {
		fmt.Println("Ошибка:", err)
		return "", errors.New(fmt.Sprintf("incorrect value %s", fromValue))
	}

	switch fromUnit {
	case "GI":
		switch toUnit {
		case "MI":
			convertedValue = GiBToMiB(value)
		case "KI":
			convertedValue = GiBToKiB(value)
		case "GI":
			convertedValue = value
		default:
			return "", errors.New(fmt.Sprintf("unsupported unit conversion: %s to %s", fromUnit, toUnit))
		}

	case "MI":
		switch toUnit {
		case "GI":
			convertedValue = MiBToGiB(value)
		case "KI":
			convertedValue = MiBToKiB(value)
		case "MI":
			convertedValue = value
		default:
			return "", errors.New(fmt.Sprintf("unsupported unit conversion: %s to %s", fromUnit, toUnit))
		}

	case "KI":
		switch toUnit {
		case "GI":
			convertedValue = KiBToGiB(value)
		case "MI":
			convertedValue = KiBToMiB(value)
		case "KI":
			convertedValue = value
		default:
			return "", errors.New(fmt.Sprintf("unsupported unit conversion: %s to %s", fromUnit, toUnit))
		}

	default:
		return "", errors.New(fmt.Sprintf("unsupported unit conversion: %s to %s", fromUnit, toUnit))
	}

	return fmt.Sprintf("%.2f%s", convertedValue, toUnit), nil
}
