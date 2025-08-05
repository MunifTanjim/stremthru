package tvdb

type CompanyTypeId int

type CompanyType struct {
	CompanyTypeId   int    `json:"companyTypeId"`
	CompanyTypeName string `json:"companyTypeName"`
}

type ParentCompany struct {
	Id       int    `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Relation struct {
		Id       int    `json:"id,omitempty"`
		TypeName string `json:"typeName,omitempty"`
	} `json:"relation"`
}

type Company struct {
	Id                   int           `json:"id"`
	Name                 string        `json:"name"`
	Slug                 string        `json:"slug"`
	NameTranslations     []string      `json:"nameTranslations"`
	OverviewTranslations []string      `json:"overviewTranslations"`
	Aliases              []Alias       `json:"aliases"`
	Country              string        `json:"country"`
	PrimaryCompanyType   int           `json:"primaryCompanyType"`
	ActiveDate           any           `json:"activeDate"`
	InactiveDate         any           `json:"inactiveDate"`
	CompanyType          CompanyType   `json:"companyType"`
	ParentCompany        ParentCompany `json:"parentCompany"`
	TagOptions           []TagOption   `json:"tagOptions"`
}

type Companies struct {
	Studio         []Company `json:"studio"`
	Network        []Company `json:"network"`
	Production     []Company `json:"production"`
	Distributor    []Company `json:"distributor"`
	SpecialEffects []Company `json:"special_effects"`
}
