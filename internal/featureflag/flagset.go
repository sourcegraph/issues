package featureflag

type FlagSet map[string]bool

func (f FlagSet) GetBool(flag string) (val bool, ok bool) {
	v, ok := f[flag]
	return v, ok
}

func (f FlagSet) GetBoolOr(flag string, defaultVal bool) bool {
	if v, ok := f[flag]; ok {
		return v
	}
	return defaultVal
}
