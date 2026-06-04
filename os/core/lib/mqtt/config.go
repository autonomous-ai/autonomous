package mqtt

type Config struct {
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	ClientID string `yaml:"client_id" json:"client_id"`
}
