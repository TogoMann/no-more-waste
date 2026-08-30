package siret

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrInvalidFormat = errors.New("le SIRET doit contenir 14 chiffres")
var ErrInvalidChecksum = errors.New("le numero SIRET est invalide (cle de controle)")
var ErrNotFound = errors.New("aucun etablissement trouve pour ce SIRET")

type Company struct {
	Siret       string `json:"siret"`
	Siren       string `json:"siren"`
	Name        string `json:"name"`
	LegalName   string `json:"legal_name"`
	Address     string `json:"address"`
	City        string `json:"city"`
	PostalCode  string `json:"postal_code"`
	Activity    string `json:"activity"`
	Active      bool   `json:"active"`
	Verified    bool   `json:"verified"`
	CheckedOnly bool   `json:"checked_only"`
}

func Normalize(value string) string {
	var builder strings.Builder
	for _, character := range value {
		if character >= '0' && character <= '9' {
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

func ValidateFormat(value string) error {
	normalized := Normalize(value)
	if len(normalized) != 14 {
		return ErrInvalidFormat
	}
	if !hasValidLuhnChecksum(normalized) {
		return ErrInvalidChecksum
	}
	return nil
}

func hasValidLuhnChecksum(digits string) bool {
	sum := 0
	length := len(digits)
	for index := 0; index < length; index++ {
		value := int(digits[length-1-index] - '0')
		if index%2 == 1 {
			value *= 2
			if value > 9 {
				value -= 9
			}
		}
		sum += value
	}
	return sum%10 == 0
}

type apiResponse struct {
	Results []struct {
		Siren                  string `json:"siren"`
		NomComplet             string `json:"nom_complet"`
		NomRaisonSociale       string `json:"nom_raison_sociale"`
		ActivitePrincipale     string `json:"activite_principale"`
		MatchingEtablissements []struct {
			Siret             string `json:"siret"`
			AdresseComplete   string `json:"adresse"`
			Libelle           string `json:"libelle_commune"`
			CodePostal        string `json:"code_postal"`
			EtatAdministratif string `json:"etat_administratif"`
		} `json:"matching_etablissements"`
	} `json:"results"`
}

func Lookup(value string) (*Company, error) {
	normalized := Normalize(value)
	if err := ValidateFormat(normalized); err != nil {
		return nil, err
	}
	company := &Company{Siret: normalized, Siren: normalized[:9]}

	client := &http.Client{Timeout: 6 * time.Second}
	endpoint := fmt.Sprintf("https://recherche-entreprises.api.gouv.fr/search?q=%s&per_page=1",
		url.QueryEscape(normalized))
	response, err := client.Get(endpoint)
	if err != nil {
		company.CheckedOnly = true
		return company, nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		company.CheckedOnly = true
		return company, nil
	}

	var parsed apiResponse
	if err := json.NewDecoder(response.Body).Decode(&parsed); err != nil {
		company.CheckedOnly = true
		return company, nil
	}
	if len(parsed.Results) == 0 {
		return nil, ErrNotFound
	}

	result := parsed.Results[0]
	company.Name = result.NomComplet
	company.LegalName = result.NomRaisonSociale
	company.Activity = result.ActivitePrincipale
	company.Verified = true
	company.Active = true

	for _, establishment := range result.MatchingEtablissements {
		if Normalize(establishment.Siret) == normalized {
			company.Address = establishment.AdresseComplete
			company.City = establishment.Libelle
			company.PostalCode = establishment.CodePostal
			company.Active = establishment.EtatAdministratif != "F"
			return company, nil
		}
	}
	return nil, ErrNotFound
}
