package templates_test

import (
	"testing"

	"github.com/tidepool-org/mailer/templates"
)

func Test_Load_Success(t *testing.T) {
	_, err := templates.Load()
	if err != nil {
		t.Fatalf(`Error is "%s", but should be nil`, err)
	}
}

func Test_Load_ExpectedTemplates(t *testing.T) {
	expectedNames := map[string]struct{}{
		"migrate_patient":                             {},
		"clinic_created":                              {},
		"clinic_migration_complete":                   {},
		"clinician_permissions_updated":               {},
		"patient_upload_reminder":                     {},
		"prescription_access_code":                    {},
		"share_invitation_received":                   {},
		"request_dexcom_connect":                      {},
		"request_dexcom_reconnect":                    {},
		"request_dexcom_connect_custodial":            {},
		"request_twiist_connect":                      {},
		"request_twiist_reconnect":                    {},
		"request_twiist_connect_custodial":            {},
		"request_abbott_connect":                      {},
		"request_abbott_reconnect":                    {},
		"request_abbott_connect_custodial":            {},
		"clinic_merged_patient_notification":          {},
		"clinic_merged_source_clinician_notification": {},
		"clinic_merged_target_admin_notification":     {},
	}

	tmplts, err := templates.Load()
	if err != nil {
		t.Fatalf(`Error is "%s", but should be nil`, err)
	}
	if len(expectedNames) != len(tmplts) {
		t.Fatalf(`Expected to have %v templates, got %v`, len(expectedNames), len(tmplts))
	}
	for name, _ := range expectedNames {
		_, ok := tmplts[templates.TemplateName(name)]
		if !ok {
			t.Errorf("%s template doesn't exist", name)
		}
	}
}
