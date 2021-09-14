package templates

import (
	"github.com/kelseyhightower/envconfig"
)

type GlobalVariables struct {
	WebAppUrl   string         `envconfig:"TIDEPOOL_WEBAPP_URL"`
	AssetUrl    string         `envconfig:"TIDEPOOL_ASSET_URL"`
}

func NewGlobalVariables() (*GlobalVariables, error) {
	vars := &GlobalVariables{}
	return vars, envconfig.Process("", vars)
}
