package pageengine

type SiteData struct {
	Pages      map[string]Page         `json:"pages"` // Flexible page names
	Components map[string]*PageElement `json:"components"`
}

type Page struct {
	Head              Section `json:"head"`
	Body              Section `json:"body"`
	RedirectForLogin  string  `json:"redirectForLogin,omitempty"`  // URL to redirect if session is active
	RedirectForLogout string  `json:"redirectForLogout,omitempty"` // URL to redirect if session is inactive
}

type Meta struct {
	Title string `json:"title"`
}

type Section struct {
	Elements []PageElement `json:"elements,omitempty"`
}

type PageElement struct {
	Type       string            `json:"type,omitempty"` // The "type" for each element (e.g., "Div")
	Attributes map[string]string `json:"attributes,omitempty"`
	Elements   []PageElement     `json:"elements,omitempty"` // Nested elements like "H1"
	Text       string            `json:"text,omitempty"`     // Text content for elements like "H1"
	Style      map[string]string `json:"style,omitempty"`    // For CSS styling properties
	Import     string            `json:"import,omitempty"`   // For component imports
	Private    bool              `json:"private,omitempty"`  // For private components. Will never show in /components export
	Pid        string            `json:"pid,omitempty"`      // For previewing components
}

type Message struct {
	Message string
	Type    string
}
