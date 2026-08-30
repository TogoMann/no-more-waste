package siret

import "testing"

func TestValidateFormat(t *testing.T) {
	cases := map[string]bool{
		"35247171800010": true,
		"55201802001808": true,
		"35247171800011": false,
		"123":            false,
		"abcdefghijklmn": false,
	}
	for value, expected := range cases {
		err := ValidateFormat(value)
		if (err == nil) != expected {
			t.Errorf("%s: attendu valide=%v, err=%v", value, expected, err)
		}
	}
}

func TestNormalize(t *testing.T) {
	if Normalize("352 471 718 00010") != "35247171800010" {
		t.Fatal("normalisation des espaces echouee")
	}
}

func TestLookupExisting(t *testing.T) {
	company, err := Lookup("35247171800010")
	if err != nil {
		t.Skipf("API indisponible: %v", err)
	}
	if company.Verified && company.Name == "" {
		t.Fatal("nom d entreprise manquant")
	}
	t.Logf("nom=%q ville=%q actif=%v verifie=%v", company.Name, company.City, company.Active, company.Verified)
}

func TestLookupUnknownSiret(t *testing.T) {
	if _, err := Lookup("00000000000000"); err == nil {
		t.Fatal("un SIRET inexistant doit etre rejete")
	}
}
