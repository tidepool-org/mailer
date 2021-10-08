package templates_test

import (
	"github.com/tidepool-org/mailer/templates"
	"testing"
)

func Test_Load_Success(t *testing.T) {
	_, err := templates.Load()
	if err != nil {
		t.Fatalf(`Error is "%s", but should be nil`, err)
	}
}

func Test_Load_ExpectedTemplates(t *testing.T) {
	expectedNames := map[string]struct{}{
		"migrate_patient": {},
		"clinic_created": {},
		"clinic_migration_complete": {},
		"clinician_permissions_updated": {},
		"share_invitation_received": {},
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
