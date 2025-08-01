package cfg

type Metadata struct {
	Name        string
	Description string
	InputParam  string
	properties  map[string]any
}

func NewMetadata(name, description string) *Metadata {
	return &Metadata{
		Name:        name,
		Description: description,
		properties:  make(map[string]any),
	}
}

func (m *Metadata) WithChainInputParam(input string) *Metadata {
	m.InputParam = input
	return m
}

func (m *Metadata) Properties() map[string]any {
	return m.properties
}

func (m *Metadata) WithProperty(key string, value any) *Metadata {
	m.properties[key] = value
	return m
}

func (m *Metadata) WithProperties(properties map[string]any) *Metadata {
	for k, v := range properties {
		m.properties[k] = v
	}
	return m
}
