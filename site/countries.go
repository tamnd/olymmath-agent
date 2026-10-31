package site

// countries maps the ISO 3166-1 code of each country in the competition register to its English short name. XK (Kosovo) is a user-assigned code.
var countries = map[string]string{
	"AD": "Andorra", "AE": "United Arab Emirates", "AL": "Albania", "AM": "Armenia", "AR": "Argentina", "AT": "Austria", "AU": "Australia", "AZ": "Azerbaijan",
	"BA": "Bosnia and Herzegovina", "BD": "Bangladesh", "BE": "Belgium", "BG": "Bulgaria", "BH": "Bahrain", "BN": "Brunei", "BR": "Brazil", "BW": "Botswana", "BY": "Belarus",
	"CA": "Canada", "CH": "Switzerland", "CI": "Côte d'Ivoire", "CL": "Chile", "CM": "Cameroon", "CN": "China", "CO": "Colombia", "CR": "Costa Rica", "CU": "Cuba", "CY": "Cyprus", "CZ": "Czechia",
	"DE": "Germany", "DK": "Denmark", "DZ": "Algeria", "EC": "Ecuador", "EE": "Estonia", "EG": "Egypt", "ES": "Spain", "ET": "Ethiopia", "FI": "Finland", "FR": "France",
	"GB": "United Kingdom", "GE": "Georgia", "GH": "Ghana", "GR": "Greece", "HK": "Hong Kong", "HR": "Croatia", "HU": "Hungary",
	"ID": "Indonesia", "IE": "Ireland", "IL": "Israel", "IN": "India", "IQ": "Iraq", "IR": "Iran", "IS": "Iceland", "IT": "Italy",
	"JO": "Jordan", "JP": "Japan", "KE": "Kenya", "KG": "Kyrgyzstan", "KH": "Cambodia", "KR": "South Korea", "KW": "Kuwait", "KZ": "Kazakhstan",
	"LA": "Laos", "LB": "Lebanon", "LI": "Liechtenstein", "LK": "Sri Lanka", "LT": "Lithuania", "LU": "Luxembourg", "LV": "Latvia",
	"MA": "Morocco", "MC": "Monaco", "MD": "Moldova", "ME": "Montenegro", "MK": "North Macedonia", "MM": "Myanmar", "MN": "Mongolia", "MO": "Macau", "MT": "Malta", "MX": "Mexico", "MY": "Malaysia",
	"NG": "Nigeria", "NL": "Netherlands", "NO": "Norway", "NP": "Nepal", "NZ": "New Zealand", "OM": "Oman", "PE": "Peru", "PH": "Philippines", "PK": "Pakistan", "PL": "Poland", "PS": "Palestine", "PT": "Portugal",
	"QA": "Qatar", "RO": "Romania", "RS": "Serbia", "RU": "Russia", "RW": "Rwanda", "SA": "Saudi Arabia", "SE": "Sweden", "SG": "Singapore", "SI": "Slovenia", "SK": "Slovakia", "SN": "Senegal", "SY": "Syria",
	"TH": "Thailand", "TJ": "Tajikistan", "TM": "Turkmenistan", "TN": "Tunisia", "TR": "Türkiye", "TW": "Taiwan", "TZ": "Tanzania",
	"UA": "Ukraine", "UG": "Uganda", "US": "United States", "UY": "Uruguay", "UZ": "Uzbekistan", "VE": "Venezuela", "VN": "Vietnam", "XK": "Kosovo", "ZA": "South Africa", "ZW": "Zimbabwe",
}

// countryName returns the name of a country code, or the code when the name is not known.
func countryName(code string) string {
	if n, ok := countries[code]; ok {
		return n
	}
	return code
}
