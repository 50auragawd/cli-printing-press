package naming

import "strings"

const (
	CurrentCLISuffix = "-pp-cli"
	LegacyCLISuffix  = "-cli"
)

func CLI(name string) string {
	return name + CurrentCLISuffix
}

func LegacyCLI(name string) string {
	return name + LegacyCLISuffix
}

func ValidationBinary(name string) string {
	return CLI(name) + "-validation"
}

func DogfoodBinary(name string) string {
	return CLI(name) + "-dogfood"
}

func IsCLIDirName(name string) bool {
	trimmed := trimNumericRunSuffix(name)
	return strings.HasSuffix(trimmed, CurrentCLISuffix) || strings.HasSuffix(trimmed, LegacyCLISuffix)
}

func TrimCLISuffix(name string) string {
	name = trimNumericRunSuffix(name)

	switch {
	case strings.HasSuffix(name, CurrentCLISuffix):
		return strings.TrimSuffix(name, CurrentCLISuffix)
	case strings.HasSuffix(name, LegacyCLISuffix):
		return strings.TrimSuffix(name, LegacyCLISuffix)
	default:
		return name
	}
}

func trimNumericRunSuffix(name string) string {
	idx := strings.LastIndex(name, "-")
	if idx == -1 {
		return name
	}

	suffix := name[idx+1:]
	if suffix == "" {
		return name
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return name
		}
	}
	return name[:idx]
}
