package v1

type device struct {
	Identifier   string
	Capabilities []string
	Gateway      string
}

type gateway struct {
	Identifier   string
	Capabilities []string
	SelfDevice   string
}
