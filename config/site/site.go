package site

type SiteConfig struct {
	Title       string     `yaml:"title"`
	Author      string     `yaml:"author"`
	BaseURL     string     `yaml:"baseUrl"`
	Description string     `yaml:"description"`
	Footer      FooterInfo `yaml:"footer"`
	Logo        string     `yaml:"logo-image"`
	Headline    string     `yaml:"headline"`
}
type FooterInfo struct {
	Note          string `yaml:"note"`
	CopyrightData string `yaml:"copyright_date"`
}
