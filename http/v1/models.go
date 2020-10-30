package v1

type device struct {
	Identifier   string
	Capabilities map[string]interface{}
	Gateway      string
}

type gateway struct {
	Identifier   string
	Capabilities []string
	SelfDevice   string
}
