package pageengine

type SiteData struct {
	Pages      map[string]Page         `json:"pages"` // Flexible page names
	Components map[string]*PageElement `json:"components"`
}

type Page struct {
	Head              Section `json:"head"`
	Body              Section `json:"body"`
	RedirectForLogin  string  `json:"redirectForLogin"`  // URL to redirect if session is active
	RedirectForLogout string  `json:"redirectForLogout"` // URL to redirect if session is inactive
}

type Meta struct {
	Title string `json:"title"`
}

type Section struct {
	Elements []PageElement `json:"elements"`
}

type PageElement struct {
	Type       string            `json:"type"` // The "type" for each element (e.g., "Div")
	Attributes map[string]string `json:"attributes"`
	Elements   []PageElement     `json:"elements"` // Nested elements like "H1"
	Text       string            `json:"text"`     // Text content for elements like "H1"
	Style      map[string]string `json:"style"`    // For CSS styling properties
	Import     string            `json:"import"`   // For component imports
	Private    bool              `json:"private"`  // For private components. Will never show in /components export

}

type Message struct {
	Message string
	Type    string
}
