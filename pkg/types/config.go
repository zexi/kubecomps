package types

// Config is used to read and store information from the cloud configuration file
type Config struct {
	AuthURL       string `json:"auth_url"`
	AdminUser     string `json:"admin_user"`
	AdminPassword string `json:"admin_password"`
	AdminProject  string `json:"admin_project"`
	Region        string `json:"region"`
	Cluster       string `json:"cluster"`
	InstanceType  string `json:"instance_type"`
}
